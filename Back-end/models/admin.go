package models

import (
	"database/sql"
	"fmt"
	"forum/Back-end/handlers"
	"log"
	"net/http"
)

func HandleAdmin(w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open("sqlite3", "./Back-end/database/forum.db")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()
	tmpl, ok := tmplCache["panel"]
	if !ok {
		http.Error(w, "Could not load panel template", http.StatusInternalServerError)
		return
	}

	// Gönderileri çekmek için sorgu
	rows1, err := db.Query(`
		SELECT 
			posts.id, posts.user_id, posts.title, posts.content, posts.image, posts.category_id, posts.created_at, posts.total_likes, posts.total_dislikes,
			categories.name, users.username
		FROM posts
		JOIN categories ON posts.category_id = categories.id
		JOIN users ON posts.user_id = users.id`)
	if err != nil {
		http.Error(w, "Could not retrieve posts", http.StatusInternalServerError)
		return
	}
	defer rows1.Close()

	var posts []handlers.Post
	for rows1.Next() {
		var post handlers.Post
		err := rows1.Scan(&post.ID, &post.UserID, &post.Title, &post.Content, &post.Image, &post.Category, &post.CreatedAt, &post.Likes, &post.Dislikes, &post.CategoryName, &post.Username)
		if err != nil {
			http.Error(w, "Could not scan post", http.StatusInternalServerError)
			return
		}

		// Görüntü URL'sini oluşturun
		if post.Image != "" {
			post.Image = "/" + post.Image
		}

		// Yorumları çekmek için sorgu
		rows2, err := db.Query(`
			SELECT 
				comments.id, comments.post_id, comments.user_id, comments.content, comments.created_at, users.username,
				IFNULL(SUM(CASE WHEN comment_likes.like_type = 'like' THEN 1 ELSE 0 END), 0) AS likes,
				IFNULL(SUM(CASE WHEN comment_likes.like_type = 'dislike' THEN 1 ELSE 0 END), 0) AS dislikes
			FROM comments 
			LEFT JOIN comment_likes ON comments.id = comment_likes.comment_id
			JOIN users ON comments.user_id = users.id
			WHERE comments.post_id = ?
			GROUP BY comments.id`, post.ID)
		if err != nil {
			http.Error(w, "Could not retrieve comments", http.StatusInternalServerError)
			return
		}
		defer rows2.Close()

		var comments []handlers.Comment
		for rows2.Next() {
			var comment handlers.Comment
			err := rows2.Scan(&comment.ID, &comment.PostID, &comment.UserID, &comment.Content, &comment.CreatedAt, &comment.Username, &comment.Likes, &comment.Dislikes)
			if err != nil {
				http.Error(w, "Could not scan comment", http.StatusInternalServerError)
				return
			}
			comments = append(comments, comment)
		}
		post.Comments = comments

		posts = append(posts, post)
	}

	// Kullanıcıları çekmek için sorgu
	rows3, err := db.Query(`SELECT id, username, email, role FROM users`)
	if err != nil {
		http.Error(w, "Could not retrieve users", http.StatusInternalServerError)
		return
	}
	defer rows3.Close()

	var users []handlers.User
	for rows3.Next() {
		var user handlers.User
		err := rows3.Scan(&user.ID, &user.Username, &user.Email, &user.Role)
		if err != nil {
			http.Error(w, "Could not scan user", http.StatusInternalServerError)
			return
		}
		users = append(users, user)
	}

	// Kullanıcı giriş yapmış mı diye kontrol edelim
	_, err = r.Cookie("user_id")
	loggedIn := err == nil
	data := map[string]interface{}{
		"LoggedIn": loggedIn,
		"Posts":    posts,
		"Users":    users, // Kullanıcı verilerini data'ya ekliyoruz
	}
	if r.Method == http.MethodPost {
		username := r.FormValue("username")

		if err := deletetable(db, username); err != nil {
			http.Error(w, "Failed to delete user: "+err.Error(), http.StatusInternalServerError)
			log.Println("Failed to delete user:", err)
			return
		}

		fmt.Fprintf(w, "Username %s deleted successfully", username)
	} else {
		if err := tmpl.Execute(w, data); err != nil {
			http.Error(w, "Could not execute template", http.StatusInternalServerError)
			return
		}
	}
}

func deletetable(database *sql.DB, username string) error {
	// Transaction başlat
	tx, err := database.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Kullanıcıyı sil
	deleteStmt := "DELETE FROM users WHERE username = ?;"
	if _, err := tx.Exec(deleteStmt, username); err != nil {
		return err
	}

	// Tabloda kalan satır sayısını kontrol et
	rowCount := 0
	countStmt := "SELECT COUNT(*) FROM users;"
	if err := tx.QueryRow(countStmt).Scan(&rowCount); err != nil {
		return err
	}

	// Eğer tablo boş ise, otomatik artan değeri sıfırla
	if rowCount == 0 {
		resetAIStmt := "DELETE FROM SQLITE_SEQUENCE WHERE NAME = 'users';"
		if _, err := tx.Exec(resetAIStmt); err != nil {
			return err
		}
	}

	// Transaction'ı commit et
	return tx.Commit()
}
