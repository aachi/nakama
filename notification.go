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
	ID       string    `json:"id"`
	UserID   string    `json:"-"`
	ActorID  string    `json:"-"`
	Verb     string    `json:"verb"`
	ObjectID *string   `json:"objectId,omitempty"`
	TargetID *string   `json:"targetId,omitempty"`
	IssuedAt time.Time `json:"issuedAt"`
	Read     bool      `json:"read"`
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
	created := !exists

	if created {
		// TODO: broadcast notification
		log.Printf("follow notification created: %v\n", notification)
	}
}
