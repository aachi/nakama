package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cockroachdb/cockroach-go/crdb"
	"github.com/go-chi/chi"
	"github.com/lib/pq"
)

// CreateUserInput request body
type CreateUserInput struct {
	Email    string `json:"email"`
	Username string `json:"username"`
}

// User model
type User struct {
	ID        string  `json:"-"`
	Username  string  `json:"username"`
	AvatarURL *string `json:"avatarUrl"`
}

// Profile model
type Profile struct {
	Email           string    `json:"email,omitempty"`
	Username        string    `json:"username"`
	AvatarURL       *string   `json:"avatarUrl"`
	FollowersCount  int       `json:"followersCount"`
	FollowingCount  int       `json:"followingCount"`
	CreatedAt       time.Time `json:"createdAt"`
	Me              bool      `json:"me"`
	FollowerOfMine  bool      `json:"followerOfMine"`
	FollowingOfMine bool      `json:"followingOfMine"`
}

// ToggleFollowPayload response body
type ToggleFollowPayload struct {
	FollowingOfMine bool `json:"followingOfMine"`
	FollowersCount  int  `json:"followersCount"`
}

const queryGetUser = "SELECT username, avatar_url FROM users WHERE id = $1"

var errFollowingMyself = errors.New("Try following someone else")

func createUser(w http.ResponseWriter, r *http.Request) {
	// Decode request body
	var input CreateUserInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	email := input.Email
	username := input.Username
	// TODO: validate input
	// Insert the user into the db
	var user Profile
	err := db.QueryRowContext(r.Context(), `
		INSERT INTO users (email, username) VALUES ($1, $2)
		RETURNING created_at
	`, email, username).Scan(&user.CreatedAt)
	if errPq, ok := err.(*pq.Error); ok && errPq.Code.Name() == "unique_violation" {
		if strings.Contains(errPq.Error(), "users_email_key") {
			respondJSON(w, map[string]string{
				"email": "Email taken",
			}, http.StatusUnprocessableEntity)
			return
		}
		respondJSON(w, map[string]string{
			"username": "Username taken",
		}, http.StatusUnprocessableEntity)
		return
	} else if err != nil {
		respondError(w, fmt.Errorf("could not create user: %v", err))
		return
	}
	user.Email = email
	user.Username = username
	user.Me = true
	// Respond with the created user
	respondJSON(w, user, http.StatusCreated)
}

func getUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authUserID, authenticated := ctx.Value(keyAuthUserID).(string)
	username := strings.TrimSpace(r.URL.Query().Get("username"))
	// Validate non empty username query parameter
	if username == "" {
		http.Error(w, "Username required", http.StatusUnprocessableEntity)
		return
	}
	// Build query based on auth state, exclude myself from results
	query := `
		SELECT
			users.username,
			users.avatar_url,
			users.followers_count,
			users.following_count,
			users.created_at`
	args := []interface{}{username}
	if authenticated {
		query += `,
			followers.follower_id IS NOT NULL AS follower_of_mine,
			following.following_id IS NOT NULL AS following_of_mine`
		args = append(args, authUserID)
	}
	query += `
		FROM users`
	if authenticated {
		query += `
			LEFT JOIN follows AS followers
				ON followers.follower_id = $2
				AND followers.following_id = users.id
			LEFT JOIN follows AS following
				ON following.follower_id = users.id
				AND following.following_id = $2
			WHERE users.id != $2 AND`
	} else {
		query += `
			WHERE`
	}
	query += ` users.username ILIKE '%' || $1 || '%'
		ORDER BY users.username`
	// Fetch users
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		respondError(w, fmt.Errorf("could not query users: %v", err))
		return
	}
	defer rows.Close()
	// Scan each user
	users := make([]Profile, 0)
	for rows.Next() {
		var user Profile
		dest := []interface{}{
			&user.Username,
			&user.AvatarURL,
			&user.FollowersCount,
			&user.FollowingCount,
			&user.CreatedAt,
		}
		if authenticated {
			dest = append(dest,
				&user.FollowerOfMine,
				&user.FollowingOfMine,
			)
		}
		if err = rows.Scan(dest...); err != nil {
			respondError(w, fmt.Errorf("could not scan user: %v", err))
			return
		}
		users = append(users, user)
	}
	if err = rows.Err(); err != nil {
		respondError(w, fmt.Errorf("could not iterate over users: %v", err))
		return
	}
	// Respond with users array
	respondJSON(w, users, http.StatusOK)
}

func getUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authUserID, authenticated := ctx.Value(keyAuthUserID).(string)
	username := chi.URLParam(r, "username")
	// Find user on the database
	query := `
		SELECT
			id,
			email,
			avatar_url,
			followers_count,
			following_count,
			created_at`
	args := []interface{}{username}
	if authenticated {
		query += `,
			EXISTS (
				SELECT 1 FROM follows
				WHERE follower_id = (SELECT id FROM users WHERE username = $1)
					AND following_id = $2
			) AS follower_of_mine,
			EXISTS (
				SELECT 1 FROM follows
				WHERE follower_id = $2
					AND following_id = (SELECT id FROM users WHERE username = $1)
			) AS following_of_mine`
		args = append(args, authUserID)
	}
	query += `
		FROM users
		WHERE username = $1`
	var userID string
	var user Profile
	dest := []interface{}{
		&userID,
		&user.Email,
		&user.AvatarURL,
		&user.FollowersCount,
		&user.FollowingCount,
		&user.CreatedAt,
	}
	if authenticated {
		dest = append(dest,
			&user.FollowerOfMine,
			&user.FollowingOfMine,
		)
	}
	if err := db.QueryRowContext(ctx, query, args...).Scan(dest...); err == sql.ErrNoRows {
		http.Error(w,
			http.StatusText(http.StatusNotFound),
			http.StatusNotFound)
		return
	} else if err != nil {
		respondError(w, fmt.Errorf("could not get user: %v", err))
		return
	}

	// Hide details when the account isn't mine
	if !authenticated || authUserID != userID {
		user.Email = ""
	}
	user.Username = username
	user.Me = authenticated && userID == authUserID

	// Respond with the found user profile
	respondJSON(w, user, http.StatusOK)
}

func toggleFollow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authUserID := ctx.Value(keyAuthUserID).(string)
	username := chi.URLParam(r, "username")

	var followingOfMine bool
	var followersCount int
	if err := crdb.ExecuteTx(ctx, db, nil, func(tx *sql.Tx) error {
		// Extract ID from username
		var userID string
		if err := tx.QueryRow("SELECT id FROM users WHERE username = $1", username).
			Scan(&userID); err != nil {
			return err
		}
		// Early return if trying to follow myself
		if authUserID == userID {
			return errFollowingMyself
		}
		// Check if already following
		if err := tx.QueryRow(`SELECT EXISTS (
			SELECT 1 FROM follows
			WHERE follower_id = $1
				AND following_id = $2
		)`, authUserID, userID).Scan(&followingOfMine); err != nil {
			return err
		}
		// If following, delete follow, decrement follower's following_count
		// and decrement user's followers_count
		if followingOfMine {
			if _, err := tx.Exec(`
				DELETE FROM follows
				WHERE follower_id = $1
					AND following_id = $2
				RETURNING NOTHING
			`, authUserID, userID); err != nil {
				return err
			}
			if _, err := tx.Exec(`
				UPDATE users SET following_count = following_count - 1
				WHERE id = $1
				RETURNING NOTHING
			`, authUserID); err != nil {
				return err
			}
			return tx.QueryRow(`
				UPDATE users SET followers_count = followers_count - 1
				WHERE id = $1
				RETURNING followers_count
			`, userID).Scan(&followersCount)
		}

		// Else insert follow, increment follower's following_count
		// and increment user's followers_count
		if _, err := tx.Exec(`
			INSERT INTO follows (follower_id, following_id)
			VALUES ($1, $2)
			RETURNING NOTHING
		`, authUserID, userID); err != nil {
			return err
		}
		if _, err := tx.Exec(`
			UPDATE users SET following_count = following_count + 1
			WHERE id = $1
			RETURNING NOTHING
		`, authUserID); err != nil {
			return err
		}
		return tx.QueryRow(`
			UPDATE users SET followers_count = followers_count + 1
			WHERE id = $1
			RETURNING followers_count
		`, userID).Scan(&followersCount)
	}); err == errFollowingMyself {
		http.Error(w,
			http.StatusText(http.StatusForbidden),
			http.StatusForbidden)
		return
	} else if err != nil {
		respondError(w, fmt.Errorf("could not toggle follow: %v", err))
		return
	}

	followingOfMine = !followingOfMine
	// TODO: Send notification
	respondJSON(w, ToggleFollowPayload{followingOfMine, followersCount}, http.StatusOK)
}
