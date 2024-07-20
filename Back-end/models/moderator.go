package models

import (
	"database/sql"
	"forum/Back-end/handlers"
	"net/http"
	"strconv"
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

func HandlePostReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// Context'ten user_id'yi almak için doğru kontrol
	cookie, err := r.Cookie("user_id")
	if err != nil {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	userID, err := strconv.Atoi(cookie.Value)
	if err != nil || userID == 0 {
		http.Error(w, "Invalid user ID", http.StatusUnauthorized)
		return
	}

	postID := r.FormValue("post_id")
	reason := r.FormValue("reason")
	if postID == "" || reason == "" {
		http.Error(w, "Missing parameters", http.StatusBadRequest)
		return
	}
	db, err := sql.Open("sqlite3", "./Back-end/database/forum.db")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	_, err = db.Exec("INSERT INTO reports (user_id, post_id, reason) VALUES (?, ?, ?)", userID, postID, reason)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/moderatorpanel", http.StatusSeeOther)
}
