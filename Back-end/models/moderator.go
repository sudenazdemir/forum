package models

import (
	"database/sql"
	"forum/Back-end/handlers"
	"net/http"
)

func HandleModerator(w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open("sqlite3", "./Back-end/database/forum.db")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	tmpl, ok := tmplCache["moderatorpanel"]
	if !ok {
		http.Error(w, "Could not load moderatorpanel template", http.StatusInternalServerError)
		return
	}
	// Fetch posts and users
	posts, err := handlers.FetchPosts(db)
	if err != nil {
		http.Error(w, "Could not retrieve posts", http.StatusInternalServerError)
		return
	}
	// Check if user is logged in
	_, err = r.Cookie("user_id")
	loggedIn := err == nil
	data := map[string]interface{}{
		"LoggedIn": loggedIn,
		"Posts":    posts,
	}
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Could not execute template", http.StatusInternalServerError)
		return
	}
}
