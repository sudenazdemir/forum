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
func HandleModRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Kullanıcı kimliği için cookie
	cookie, err := r.Cookie("user_id")
	if err != nil {
		http.Error(w, "User not logged in", http.StatusUnauthorized)
		return
	}
	userID := cookie.Value

	// Veritabanı bağlantısını aç
	db, err := sql.Open("sqlite3", "./Back-end/database/forum.db")
	if err != nil {
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Kullanıcı ID'sinin geçerli olduğunu doğrula
	var userCount int
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE id = ?", userID).Scan(&userCount)
	if err != nil {
		http.Error(w, "Failed to check user existence: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if userCount == 0 {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Kullanıcının zaten bir moderatör talebi olup olmadığını kontrol et
	var requestCount int
	err = db.QueryRow("SELECT COUNT(*) FROM mod_requests WHERE user_id = ? AND status = 'pending'", userID).Scan(&requestCount)
	if err != nil {
		http.Error(w, "Failed to check mod request: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if requestCount > 0 {
		http.Error(w, "You have already requested to be a moderator", http.StatusBadRequest)
		return
	}

	// Moderatör talebini ekle
	_, err = db.Exec("INSERT INTO mod_requests (user_id, status) VALUES (?, 'pending')", userID)
	if err != nil {
		http.Error(w, "Failed to create mod request: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}
