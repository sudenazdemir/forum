package handlers

import "database/sql"

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
