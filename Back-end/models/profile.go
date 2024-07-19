package models

import (
	"database/sql"
	"net/http"
	"strconv"

	"forum/Back-end/handlers"
)

func HandleProfile(w http.ResponseWriter, r *http.Request) {
	var user handlers.User

	cookie, err := r.Cookie("user_id")
	if err != nil {
		http.Error(w, "User ID not provided", http.StatusBadRequest)
		return
	}

	userID, err := strconv.ParseInt(cookie.Value, 10, 64)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	db, err := sql.Open("sqlite3", "./Back-end/database/forum.db")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	err = db.QueryRow("SELECT id, username, email, role FROM users WHERE id = ?", userID).Scan(&user.ID, &user.Username, &user.Email, &user.Role)
	if err != nil {
		http.Error(w, "User not found", http.StatusInternalServerError)
		return
	}

	posts, err := handlers.FetchUserPosts(db, userID)
	if err != nil {
		http.Error(w, "Could not retrieve posts", http.StatusInternalServerError)
		return
	}

	likedPosts, err := handlers.FetchLikedPosts(db, userID)
	if err != nil {
		http.Error(w, "Could not retrieve liked posts", http.StatusInternalServerError)
		return
	}

	dislikedPosts, err := handlers.FetchDislikedPosts(db, userID)
	if err != nil {
		http.Error(w, "Could not retrieve disliked posts", http.StatusInternalServerError)
		return
	}

	comments, err := handlers.FetchUserComments(db, userID)
	if err != nil {
		http.Error(w, "Could not retrieve comments", http.StatusInternalServerError)
		return
	}

	tmpl, ok := tmplCache["profile"]
	if !ok {
		http.Error(w, "Could not load profile template", http.StatusInternalServerError)
		return
	}

	// Kullanıcı bilgilerini ve gönderilerini template'e ekleyelim
	tmpl.Execute(w, map[string]interface{}{
		"Username":      user.Username,
		"Email":         user.Email,
		"Role":          user.Role,
		"LoggedIn":      true,
		"Posts":         posts,
		"LikedPosts":    likedPosts,
		"DislikedPosts": dislikedPosts,
		"Comments":      comments,
	})
}
