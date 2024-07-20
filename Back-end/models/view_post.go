package models

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"forum/Back-end/handlers"
)

func HandleViewPost(w http.ResponseWriter, r *http.Request) {
	tmpl, ok := tmplCache["view_post"]
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

	idParam := r.URL.Query().Get("id")
	if idParam == "" {
		http.Error(w, "Post ID is required", http.StatusBadRequest)
		return
	}

	postID, err := strconv.Atoi(idParam)
	if err != nil {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	var post handlers.Post
	err = db.QueryRow("SELECT id, user_id, title, content, image, category_id, created_at, total_likes, total_dislikes FROM posts WHERE id = ?", postID).Scan(
		&post.ID, &post.UserID, &post.Title, &post.Content, &post.Image, &post.Category, &post.CreatedAt, &post.Likes, &post.Dislikes)
	if err != nil {
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	}

	err = db.QueryRow("SELECT name FROM categories WHERE id = ?", post.Category).Scan(&post.CategoryName)
	if err != nil {
		http.Error(w, "Category not found", http.StatusInternalServerError)
		return
	}
	err = db.QueryRow("SELECT username FROM users WHERE id = ?", post.UserID).Scan(&post.Username)
	if err != nil {
		http.Error(w, "User not found", http.StatusInternalServerError)
		return
	}

	if post.Image != "" {
		post.Image = "/" + post.Image
	}

	rows, err := db.Query("SELECT id, post_id, user_id, content, created_at FROM comments WHERE post_id = ?", post.ID)
	if err != nil {
		http.Error(w, "Could not retrieve comments", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var comments []handlers.Comment
	for rows.Next() {
		var comment handlers.Comment
		err := rows.Scan(&comment.ID, &comment.PostID, &comment.UserID, &comment.Content, &comment.CreatedAt)
		if err != nil {
			http.Error(w, "Could not scan comment", http.StatusInternalServerError)
			return
		}
		err = db.QueryRow("SELECT username FROM users WHERE id = ?", comment.UserID).Scan(&comment.Username)
		if err != nil {
			http.Error(w, "User not found for comment", http.StatusInternalServerError)
			return
		}

		err = db.QueryRow(`
			SELECT 
				IFNULL(SUM(CASE WHEN comment_likes.like_type = 'like' THEN 1 ELSE 0 END), 0) AS likes,
				IFNULL(SUM(CASE WHEN comment_likes.like_type = 'dislike' THEN 1 ELSE 0 END), 0) AS dislikes
			FROM comments
			LEFT JOIN comment_likes ON comments.id = comment_likes.comment_id
			WHERE comments.id = ?
			GROUP BY comments.id`, comment.ID).Scan(&comment.Likes, &comment.Dislikes)
		if err != nil {
			http.Error(w, "Could not retrieve comment likes and dislikes", http.StatusInternalServerError)
			return
		}

		comments = append(comments, comment)
	}

	post.Comments = comments

	cookie, err := r.Cookie("user_id")
	var loggedIn bool
	var userID int
	if err == nil {
		userID, err = strconv.Atoi(cookie.Value)
		loggedIn = err == nil
	}

	data := map[string]interface{}{
		"LoggedIn":          loggedIn,
		"UserID":            userID,
		"Post":              post,
		"ShowLoginRegister": !loggedIn,
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Could not execute template", http.StatusInternalServerError)
		return
	}
}

func HandleDeleteComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	db, err := sql.Open("sqlite3", "./Back-end/database/forum.db")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	commentID, err := getCommentID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	userID, role, err := getUserIDAndRole(db, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if !canDeleteComment(db, commentID, userID, role) {
		http.Error(w, "Unauthorized to delete comment", http.StatusForbidden)
		return
	}

	if err := deleteComment(db, commentID); err != nil {
		http.Error(w, "Failed to delete comment", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, r.Referer(), http.StatusSeeOther)
}

func getCommentID(r *http.Request) (int, error) {
	commentIDParam := r.FormValue("comment_id")
	if commentIDParam == "" {
		return 0, fmt.Errorf("comment ID is required")
	}

	commentID, err := strconv.Atoi(commentIDParam)
	if err != nil {
		return 0, fmt.Errorf("invalid comment ID")
	}
	return commentID, nil
}

func getUserIDAndRole(db *sql.DB, r *http.Request) (int, string, error) {
	cookie, err := r.Cookie("user_id")
	if err != nil {
		return 0, "", fmt.Errorf("user not logged in")
	}

	userID, err := strconv.Atoi(cookie.Value)
	if err != nil {
		return 0, "", fmt.Errorf("invalid user ID")
	}

	var role string
	err = db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&role)
	if err != nil {
		return 0, "", fmt.Errorf("failed to get user role")
	}

	return userID, role, nil
}

func canDeleteComment(db *sql.DB, commentID, userID int, role string) bool {
	if role == "admin" {
		return true
	}

	var commentUserID, postUserID int
	err := db.QueryRow("SELECT user_id FROM comments WHERE id = ?", commentID).Scan(&commentUserID)
	if err != nil {
		return false
	}

	var postID int
	err = db.QueryRow("SELECT post_id FROM comments WHERE id = ?", commentID).Scan(&postID)
	if err != nil {
		return false
	}

	err = db.QueryRow("SELECT user_id FROM posts WHERE id = ?", postID).Scan(&postUserID)
	if err != nil {
		return false
	}

	return userID == commentUserID || userID == postUserID
}

func deleteComment(db *sql.DB, commentID int) error {
	_, err := db.Exec("DELETE FROM comments WHERE id = ?", commentID)
	return err
}
