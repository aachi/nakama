package main

import (
	"net/http"
)

// FeedItem model
type FeedItem struct {
	ID     string `json:"id"`
	UserID string `json:"-"`
	PostID string `json:"-"`
	Post   Post   `json:"post"`
}

func getFeed(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authUserID := ctx.Value(keyAuthUserID).(string)

	// Get feed, joined with post and user
	rows, err := db.QueryContext(ctx, `
		SELECT
			feed.id,
			posts.id,
			posts.content,
			posts.spoiler_of,
			posts.likes_count,
			posts.comments_count,
			posts.created_at,
			users.username,
			users.avatar_url,
			posts.user_id = $1 AS mine,
			likes.user_id IS NOT NULL AS liked,
			subscriptions.user_id IS NOT NULL AS subscribed
		FROM feed
		INNER JOIN posts ON feed.post_id = posts.id
		INNER JOIN users ON posts.user_id = users.id
		LEFT JOIN post_likes AS likes
			ON likes.user_id = $1
			AND likes.post_id = posts.id
		LEFT JOIN subscriptions
			ON subscriptions.user_id = $1
			AND subscriptions.post_id = posts.id
		WHERE feed.user_id = $1
		ORDER BY posts.created_at DESC
	`, authUserID)
	if err != nil {
		respondError(w, err)
		return
	}
	defer rows.Close()
	// Scan each row
	feed := make([]FeedItem, 0)
	for rows.Next() {
		var user User
		var post Post
		var feedItem FeedItem
		if err = rows.Scan(
			&feedItem.ID,
			&post.ID,
			&post.Content,
			&post.SpoilerOf,
			&post.LikesCount,
			&post.CommentsCount,
			&post.CreatedAt,
			&user.Username,
			&user.AvatarURL,
			&post.Mine,
			&post.Liked,
			&post.Subscribed,
		); err != nil {
			respondError(w, err)
			return
		}
		post.User = &user
		feedItem.Post = post
		feed = append(feed, feedItem)
	}
	if err = rows.Err(); err != nil {
		respondError(w, err)
		return
	}
	// Respond with feed array
	respondJSON(w, feed, http.StatusOK)
}
