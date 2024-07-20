package handlers

import (
	"database/sql"
)

// Function to fetch users
func FetchUsers(db *sql.DB) ([]User, error) {
	rows, err := db.Query(`
		SELECT id, username, email, role FROM users
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.Role)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

// Function to fetch posts
func FetchPosts(db *sql.DB) ([]Post, error) {
	rows, err := db.Query(`
		SELECT 
			posts.id, posts.user_id, posts.title, posts.content, posts.image, posts.category_id, posts.created_at, posts.total_likes, posts.total_dislikes,
			categories.name, users.username
		FROM posts
		JOIN categories ON posts.category_id = categories.id
		JOIN users ON posts.user_id = users.id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var post Post
		err := rows.Scan(&post.ID, &post.UserID, &post.Title, &post.Content, &post.Image, &post.Category, &post.CreatedAt, &post.Likes, &post.Dislikes, &post.CategoryName, &post.Username)
		if err != nil {
			return nil, err
		}

		// Fetch comments for each post
		post.Comments, err = FetchComments(db, post.ID)
		if err != nil {
			return nil, err
		}

		posts = append(posts, post)
	}

	return posts, nil
}

// Function to fetch comments for a specific post
func FetchComments(db *sql.DB, postID int) ([]Comment, error) {
	rows, err := db.Query(`
		SELECT 
			comments.id, comments.post_id, comments.user_id, comments.content, comments.created_at, users.username,
			IFNULL(SUM(CASE WHEN comment_likes.like_type = 'like' THEN 1 ELSE 0 END), 0) AS likes,
			IFNULL(SUM(CASE WHEN comment_likes.like_type = 'dislike' THEN 1 ELSE 0 END), 0) AS dislikes
		FROM comments 
		LEFT JOIN comment_likes ON comments.id = comment_likes.comment_id
		JOIN users ON comments.user_id = users.id
		WHERE comments.post_id = ?
		GROUP BY comments.id`, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []Comment
	for rows.Next() {
		var comment Comment
		err := rows.Scan(&comment.ID, &comment.PostID, &comment.UserID, &comment.Content, &comment.CreatedAt, &comment.Username, &comment.Likes, &comment.Dislikes)
		if err != nil {
			return nil, err
		}
		comments = append(comments, comment)
	}

	return comments, nil
}

// FetchCategories, veri tabanından kategorileri çeker ve bir slice döner.
func FetchCategories(db *sql.DB) ([]Category, error) {
	// Kategorileri çekmek için sorgu
	rows, err := db.Query("SELECT id, name, link FROM categories")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var category Category
		err := rows.Scan(&category.ID, &category.Name, &category.Link)
		if err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return categories, nil
}
func FetchUserPosts(db *sql.DB, userID int64) ([]Post, error) {
	rows, err := db.Query("SELECT id, user_id, title, content, image, category_id, created_at, total_likes, total_dislikes FROM posts WHERE user_id = ?", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var post Post
		err := rows.Scan(&post.ID, &post.UserID, &post.Title, &post.Content, &post.Image, &post.Category, &post.CreatedAt, &post.Likes, &post.Dislikes)
		if err != nil {
			return nil, err
		}
		err = db.QueryRow("SELECT name FROM categories WHERE id = ?", post.Category).Scan(&post.CategoryName)
		if err != nil {
			return nil, err
		}
		if post.Image != "" {
			post.Image = "/" + post.Image
		}
		posts = append(posts, post)
	}
	return posts, rows.Err()
}
func FetchLikedPosts(db *sql.DB, userID int64) ([]Post, error) {
	rows, err := db.Query(`
		SELECT p.id, p.user_id, p.title, p.content, p.image, p.category_id, p.created_at, p.total_likes, p.total_dislikes
		FROM posts p
		INNER JOIN likes l ON p.id = l.post_id
		WHERE l.user_id = ? AND l.like_type = 'like'`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var likedPosts []Post
	for rows.Next() {
		var post Post
		err := rows.Scan(&post.ID, &post.UserID, &post.Title, &post.Content, &post.Image, &post.Category, &post.CreatedAt, &post.Likes, &post.Dislikes)
		if err != nil {
			return nil, err
		}
		err = db.QueryRow("SELECT name FROM categories WHERE id = ?", post.Category).Scan(&post.CategoryName)
		if err != nil {
			return nil, err
		}
		if post.Image != "" {
			post.Image = "/" + post.Image
		}
		likedPosts = append(likedPosts, post)
	}
	return likedPosts, rows.Err()
}
func FetchDislikedPosts(db *sql.DB, userID int64) ([]Post, error) {
	rows, err := db.Query(`
		SELECT p.id, p.user_id, p.title, p.content, p.image, p.category_id, p.created_at, p.total_likes, p.total_dislikes
		FROM posts p
		INNER JOIN likes l ON p.id = l.post_id
		WHERE l.user_id = ? AND l.like_type = 'dislike'`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dislikedPosts []Post
	for rows.Next() {
		var post Post
		err := rows.Scan(&post.ID, &post.UserID, &post.Title, &post.Content, &post.Image, &post.Category, &post.CreatedAt, &post.Likes, &post.Dislikes)
		if err != nil {
			return nil, err
		}
		err = db.QueryRow("SELECT name FROM categories WHERE id = ?", post.Category).Scan(&post.CategoryName)
		if err != nil {
			return nil, err
		}
		if post.Image != "" {
			post.Image = "/" + post.Image
		}
		dislikedPosts = append(dislikedPosts, post)
	}
	return dislikedPosts, rows.Err()
}
func FetchUserComments(db *sql.DB, userID int64) ([]Comment, error) {
	rows, err := db.Query(`
		SELECT c.id, c.post_id, c.user_id, c.content, c.created_at, u.username
		FROM comments c
		INNER JOIN users u ON c.user_id = u.id
		WHERE c.user_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []Comment
	for rows.Next() {
		var comment Comment
		err := rows.Scan(&comment.ID, &comment.PostID, &comment.UserID, &comment.Content, &comment.CreatedAt, &comment.Username)
		if err != nil {
			return nil, err
		}
		comments = append(comments, comment)
	}
	return comments, rows.Err()
}

func FetchModRequests(db *sql.DB) ([]ModRequest, error) {
	rows, err := db.Query("SELECT id, user_id, status FROM mod_requests WHERE status = 'pending'")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var modRequests []ModRequest
	for rows.Next() {
		var modRequest ModRequest
		err := rows.Scan(&modRequest.ID, &modRequest.UserID, &modRequest.Status)
		if err != nil {
			return nil, err
		}
		modRequests = append(modRequests, modRequest)
	}
	return modRequests, nil
}
