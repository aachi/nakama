package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	_ "github.com/lib/pq"
)

var db *sql.DB

func main() {
	var err error
	// Database connection
	databaseURL := env("DATABASE_URL", "postgresql://root@127.0.0.1:26257/nakama?sslmode=disable")
	db, err = sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}
	// Router
	mux := chi.NewMux()
	mux.Use(middleware.Recoverer)
	mux.Route("/api", func(api chi.Router) {
		jsonRequired := middleware.AllowContentType("application/json")
		api.With(jsonRequired).Post("/login", login)
		api.Post("/logout", logout)
		api.With(jsonRequired).Post("/users", createUser)
		api.With(maybeAuthUserID).Get("/users", getUsers)
		api.With(maybeAuthUserID).Get("/users/{username}", getUser)
		api.With(mustAuthUser).Post("/users/{username}/toggle_follow", toggleFollow)
		api.With(jsonRequired, mustAuthUser).Post("/posts", createPost)
		api.With(maybeAuthUserID).Get("/users/{username}/posts", getPosts)
		api.With(maybeAuthUserID).Get("/posts/{post_id}", getPost)
		api.With(mustAuthUser).Get("/feed", getFeed)
		api.With(jsonRequired, mustAuthUser).Post("/posts/{post_id}/comments", createComment)
		api.With(maybeAuthUserID).Get("/posts/{post_id}/comments", getComments)
		api.With(mustAuthUser).Post("/posts/{post_id}/toggle_like", togglePostLike)
		api.With(mustAuthUser).Post("/comments/{comment_id}/toggle_like", toggleCommentLike)
	})
	// Server
	port := env("PORT", "80")
	log.Printf("Server listenning on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func env(key, fallbackValue string) string {
	value, present := os.LookupEnv(key)
	if !present {
		return fallbackValue
	}
	return value
}

func respondError(w http.ResponseWriter, err error) {
	log.Println(err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func respondJSON(w http.ResponseWriter, v interface{}, code int) {
	b, err := json.Marshal(v)
	if err != nil {
		respondError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	w.Write(b)
}
