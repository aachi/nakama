package main

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/cockroachdb/cockroach-go/crdb"
)

// Notification model
type Notification struct {
	ID            string    `json:"id"`
	UserID        string    `json:"-"`
	ActorID       string    `json:"-"`
	Verb          string    `json:"verb"`
	ObjectID      *string   `json:"objectId,omitempty"`
	TargetID      *string   `json:"targetId,omitempty"`
	IssuedAt      time.Time `json:"issuedAt"`
	Read          bool      `json:"read"`
	ActorUsername string    `json:"actorUsername"`
}

func createFollowNotification(follower User, followingID string) {
	var exists bool
	var notification Notification
	if err := crdb.ExecuteTx(context.Background(), db, nil, func(tx *sql.Tx) error {
		if err := tx.QueryRow(`SELECT EXISTS (
			SELECT 1 FROM notifications
			WHERE user_id = $1
				AND actor_id = $2
				AND verb = 'follow'
		)`, followingID, follower.ID).Scan(&exists); err != nil {
			return err
		}

		if exists {
			return nil
		}

		return tx.QueryRow(`
			INSERT INTO notifications (user_id, actor_id, verb) VALUES ($1, $2, 'follow')
			RETURNING id, issued_at
		`, followingID, follower.ID).Scan(&notification.ID, &notification.IssuedAt)
	}); err != nil {
		log.Printf("could not create follow notification: %v\n", err)
		return
	}

	notification.UserID = followingID
	notification.ActorID = follower.ID
	notification.Verb = "follow"
	notification.ActorUsername = follower.Username
	created := !exists

	if created {
		// TODO: broadcast notification
		log.Printf("follow notification created: %v\n", notification)
	}
}

func commentNotificationFanout(comment Comment) {
	rows, err := db.Query(`
		INSERT INTO notifications (user_id, actor_id, verb, object_id, target_id)
		SELECT user_id, $1, 'comment', $2, $3
		FROM subscriptions
		WHERE user_id != $1 AND post_id = $3
		RETURNING id, user_id, issued_at
	`, comment.UserID, comment.ID, comment.PostID)
	if err != nil {
		log.Printf("could not query comment notification fanout: %v\n", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var notification Notification
		if err = rows.Scan(
			&notification.ID,
			&notification.UserID,
			&notification.IssuedAt,
		); err != nil {
			log.Printf("could not scan comment notification fanout: %v\n", err)
			return
		}

		notification.ActorID = comment.UserID
		notification.Verb = "comment"
		notification.ObjectID = &comment.ID
		notification.TargetID = &comment.PostID
		notification.ActorUsername = comment.User.Username

		// TODO: broadcast
		log.Printf("comment notification created: %v", notification)
	}

	if err = rows.Err(); err != nil {
		log.Printf("could not iterate over comment notification fanout: %v\n", err)
	}
}
