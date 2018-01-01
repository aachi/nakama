package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
)

// LoginInput request body
type LoginInput struct {
	Email string `json:"email"`
}

// LoginPayload response body
type LoginPayload struct {
	User      User      `json:"user"`
	JWT       string    `json:"jwt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// ContextKey used in middlewares
type ContextKey int

const (
	keyAuthUserID ContextKey = iota
	keyAuthUser
)

const year = time.Hour * 24 * 365

var jwtKey = []byte(env("JWT_KEY", "secret"))

func jwtKeyfunc(*jwt.Token) (interface{}, error) {
	return jwtKey, nil
}

// insecure login
func login(w http.ResponseWriter, r *http.Request) {
	// Decode request body
	var input LoginInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	email := input.Email
	// TODO: validate input, passwordless
	// Find user on the database with the given email
	var user User
	if err := db.QueryRowContext(r.Context(), `
		SELECT id, username, avatar_url
		FROM users
		WHERE email = $1
	`, email).Scan(
		&user.ID,
		&user.Username,
		&user.AvatarURL,
	); err == sql.ErrNoRows {
		http.Error(w,
			http.StatusText(http.StatusNotFound),
			http.StatusNotFound)
		return
	} else if err != nil {
		respondError(w, err)
		return
	}
	// Isuue a JWT
	expiresAt := time.Now().Add(year) // One year
	tokenString, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.StandardClaims{
		Subject:   user.ID,
		ExpiresAt: expiresAt.Unix(),
	}).SignedString(jwtKey)
	if err != nil {
		respondError(w, err)
		return
	}
	// Respond with the JWT
	http.SetCookie(w, &http.Cookie{
		Name:     "jwt",
		Value:    tokenString,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		// Secure:   true,
	})
	respondJSON(w, LoginPayload{user, tokenString, expiresAt}, http.StatusOK)
}

func logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:   "jwt",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	w.WriteHeader(http.StatusNoContent)
}

func maybeAuthUserID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract JWT from Header or Cookie
		var tokenString string
		if a := r.Header.Get("Authorization"); strings.HasPrefix(a, "Bearer ") {
			tokenString = a[7:]
		} else if c, err := r.Cookie("jwt"); err == nil {
			tokenString = c.Value
		} else {
			next.ServeHTTP(w, r)
			return
		}
		// Parse and validate JWT
		p := jwt.Parser{ValidMethods: []string{jwt.SigningMethodHS256.Name}}
		token, err := p.ParseWithClaims(tokenString, &jwt.StandardClaims{}, jwtKeyfunc)
		if err != nil {
			http.Error(w,
				http.StatusText(http.StatusUnauthorized),
				http.StatusUnauthorized)
			return
		}
		claims, ok := token.Claims.(*jwt.StandardClaims)
		if !ok || !token.Valid {
			http.Error(w,
				http.StatusText(http.StatusUnauthorized),
				http.StatusUnauthorized)
			return
		}
		// Set the auth user ID in the request context
		authUserID := claims.Subject
		ctx := r.Context()
		ctx = context.WithValue(ctx, keyAuthUserID, authUserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func mustAuthUser(next http.Handler) http.Handler {
	return maybeAuthUserID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if authenticated
		ctx := r.Context()
		authUserID, authenticated := ctx.Value(keyAuthUserID).(string)
		if !authenticated {
			http.Error(w,
				http.StatusText(http.StatusUnauthorized),
				http.StatusUnauthorized)
			return
		}
		// Find the user on the db
		var authUser User
		if err := db.QueryRowContext(ctx, queryGetUser, authUserID).Scan(
			&authUser.Username,
			&authUser.AvatarURL,
		); err == sql.ErrNoRows {
			http.Error(w,
				http.StatusText(http.StatusTeapot),
				http.StatusTeapot)
			return
		} else if err != nil {
			respondError(w, err)
			return
		}
		authUser.ID = authUserID
		// Put the auth user in the request context
		ctx = context.WithValue(ctx, keyAuthUser, authUser)
		next.ServeHTTP(w, r.WithContext(ctx))
	}))
}
