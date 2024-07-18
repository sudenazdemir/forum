package models

import (
	"database/sql"
	"net/http"
	"os"
	"path/filepath"
)

// Gönderi silme işlemi
func HandleDeletePost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	postID := r.URL.Query().Get("id")
	if postID == "" {
		http.Error(w, "Post ID missing", http.StatusBadRequest)
		return
	}

	// Kullanıcı kimliği alınır
	userIDCookie, err := r.Cookie("user_id")
	if err != nil {
		http.Error(w, "User ID not provided", http.StatusUnauthorized)
		return
	}
	userID := userIDCookie.Value

	// Kullanıcı rolü alınır
	roleCookie, err := r.Cookie("user_role")
	if err != nil || roleCookie.Value == "" {
		http.Error(w, "Role not provided", http.StatusUnauthorized)

		return
	}
	userRole := roleCookie.Value

	db, err := sql.Open("sqlite3", "./Back-end/database/forum.db")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Eğer kullanıcı admin değilse gönderinin sahibini kontrol et
	if userRole != "admin" {
		var ownerID string
		err = db.QueryRow("SELECT user_id FROM posts WHERE id = ?", postID).Scan(&ownerID)
		if err != nil {
			http.Error(w, "Post not found", http.StatusNotFound)
			return
		}

		if ownerID != userID {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	// Gönderiye ait fotoğraf dosya yolunu al
	var photoPath string
	err = db.QueryRow("SELECT image FROM posts WHERE id = ?", postID).Scan(&photoPath)
	if err != nil {

		http.Error(w, "Could not retrieve photo path", http.StatusInternalServerError)

		return
	}

	// Transaction başlat
	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "Could not begin transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback() // Transaction işlemi başarısız olursa geri alma

	// Gönderiye ait yorumları sil
	_, err = tx.Exec("DELETE FROM comments WHERE post_id = ?", postID)
	if err != nil {
		http.Error(w, "Could not delete comments", http.StatusInternalServerError)
		return
	}

	// Gönderiyi sil
	_, err = tx.Exec("DELETE FROM posts WHERE id = ?", postID)
	if err != nil {
		http.Error(w, "Could not delete post", http.StatusInternalServerError)
		return
	}

	// Transaction commit et
	err = tx.Commit()
	if err != nil {
		http.Error(w, "Could not commit transaction", http.StatusInternalServerError)
		return
	}

	// Fotoğrafı sil
	if photoPath != "" {
		err = os.Remove(filepath.Join("./Back-end/", photoPath))
		if err != nil {
			http.Error(w, "Could not delete photo", http.StatusInternalServerError)
			return
		}
	}

	// Başarılı bir şekilde silindiğini belirt
	w.Write([]byte("Post, its comments, and associated photo deleted successfully"))
}
