package models

import (
	"database/sql"
	"net/http"
	"strconv"

	"forum/Back-end/handlers"
)

func HandleHome(w http.ResponseWriter, r *http.Request) {
	tmpl, ok := tmplCache["home"]
	if !ok {
		http.Error(w, "Could not load home template", http.StatusInternalServerError)
		return
	}

	db, err := sql.Open("sqlite3", "./Back-end/database/forum.db")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	categories, err := handlers.FetchCategories(db)
	if err != nil {
		http.Error(w, "Could not retrieve categories", http.StatusInternalServerError)
		return
	}

	posts, err := handlers.FetchPosts(db)
	if err != nil {
		http.Error(w, "Could not retrieve posts", http.StatusInternalServerError)
		return
	}
	// Kullanıcı giriş yapmış mı diye kontrol edelim
	_, err = r.Cookie("user_id")
	loggedIn := err == nil

	// Şablonu render et
	data := map[string]interface{}{
		"LoggedIn":   loggedIn,
		"Categories": categories,
		"Posts":      posts,
	}

	// Kullanıcı giriş yapmamışsa Login ve Register bağlantılarını ekle
	if !loggedIn {
		data["ShowLoginRegister"] = true
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Could not execute template", http.StatusInternalServerError)
		return
	}
}

func HandleLikeComment(w http.ResponseWriter, r *http.Request) {
	HandleCommentLikeDislike(w, r, "like")
}

func HandleDislikeComment(w http.ResponseWriter, r *http.Request) {
	HandleCommentLikeDislike(w, r, "dislike")
}

func HandleCommentLikeDislike(w http.ResponseWriter, r *http.Request, likeType string) {
	r.ParseForm()
	commentID, err := strconv.Atoi(r.FormValue("commentId"))
	if err != nil {
		http.Error(w, "Invalid comment ID", http.StatusBadRequest)
		return
	}

	userID, err := r.Cookie("user_id")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	db, err := sql.Open("sqlite3", "./Back-end/database/forum.db")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var existingLikeType string
	err = db.QueryRow("SELECT like_type FROM comment_likes WHERE user_id = ? AND comment_id = ?", userID.Value, commentID).Scan(&existingLikeType)

	if err == sql.ErrNoRows {
		// Kullanıcı daha önce bu yorumu beğenmemiş veya beğenmemiş
		_, err = db.Exec("INSERT INTO comment_likes (user_id, comment_id, like_type) VALUES (?, ?, ?)", userID.Value, commentID, likeType)
		if err != nil {
			http.Error(w, "Could not insert like/dislike", http.StatusInternalServerError)
			return
		}
	} else if err != nil {
		http.Error(w, "Could not query like/dislike", http.StatusInternalServerError)
		return
	} else if existingLikeType == likeType {
		// Kullanıcı zaten bu yorumu beğenmiş veya beğenmeme yapmış, geri al
		_, err = db.Exec("DELETE FROM comment_likes WHERE user_id = ? AND comment_id = ?", userID.Value, commentID)
		if err != nil {
			http.Error(w, "Could not remove like/dislike", http.StatusInternalServerError)
			return
		}
	} else {
		// Kullanıcı farklı bir işlem yapmış, güncelle
		_, err = db.Exec("UPDATE comment_likes SET like_type = ? WHERE user_id = ? AND comment_id = ?", likeType, userID.Value, commentID)
		if err != nil {
			http.Error(w, "Could not update like/dislike", http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusFound)
}
