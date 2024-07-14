package models

import (
	"database/sql"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"forum/Back-end/handlers"
)

func HandleCreatePost(w http.ResponseWriter, r *http.Request) {
	tmpl, ok := tmplCache["create_post"]
	if !ok {
		http.Error(w, "Could not load create_post template", http.StatusInternalServerError)
		return
	}
	db, err := sql.Open("sqlite3", "./Back-end/database/forum.db")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Kategorileri çekmek için sorgu
	rows, err := db.Query("SELECT id, name FROM categories")
	if err != nil {
		http.Error(w, "Could not retrieve categories", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var categories []handlers.Category
	for rows.Next() {
		var category handlers.Category
		err := rows.Scan(&category.ID, &category.Name)
		if err != nil {
			http.Error(w, "Error scanning category", http.StatusInternalServerError)
			return
		}
		categories = append(categories, category)
	}
	if err = rows.Err(); err != nil {
		http.Error(w, "Error iterating categories", http.StatusInternalServerError)
		return
	}

	// Kullanıcı giriş yapmış mı diye kontrol edelim
	_, err = r.Cookie("user_id")
	loggedIn := err == nil

	// Şablonu render et
	data := map[string]interface{}{
		"LoggedIn":   loggedIn,
		"Categories": categories,
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

func HandleSubmitPost(w http.ResponseWriter, r *http.Request) {
	// Parse the form data
	if err := r.ParseMultipartForm(20 << 20); err != nil { // 20 MB max memory
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	// Get form values
	content := r.FormValue("content")
	title := r.FormValue("title")
	categoryIDs := r.Form["category[]"]

	// Get user_id from cookie
	cookie, err := r.Cookie("user_id")
	if err != nil {
		http.Error(w, "User not logged in", http.StatusUnauthorized)
		return
	}
	userID := cookie.Value

	// Handle file upload
	var imagePath string
	file, handler, err := r.FormFile("image")
	if err == nil {
		defer file.Close()

		// Check file size (20 MB limit)
		if handler.Size > 20<<20 {
			http.Error(w, "File size exceeds 20 MB", http.StatusBadRequest)
			return
		}

		// Check file type
		allowedTypes := []string{".gif", ".png", ".jpg", ".jpeg", ".webp", ".svg"}
		fileExt := strings.ToLower(filepath.Ext(handler.Filename))
		isValidType := false
		for _, ext := range allowedTypes {
			if fileExt == ext {
				isValidType = true
				break
			}
		}
		if !isValidType {
			http.Error(w, "Invalid file type", http.StatusBadRequest)
			return
		}

		// Ensure the uploads directory exists
		uploadDir := filepath.Join("Back-end", "uploads")
		if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
			os.MkdirAll(uploadDir, os.ModePerm)
		}

		// Create file
		fileName := handler.Filename
		imagePath = filepath.Join(uploadDir, fileName)
		out, err := os.Create(imagePath)
		if err != nil {
			http.Error(w, "Unable to create the file for writing", http.StatusInternalServerError)
			return
		}
		defer out.Close()

		// Copy the file content
		if _, err := io.Copy(out, file); err != nil {
			http.Error(w, "Unable to save the file", http.StatusInternalServerError)
			return
		}

		// Store only the relative path in the database
		imagePath = filepath.Join("uploads", fileName)
	} else if err != http.ErrMissingFile {
		http.Error(w, "Error uploading file", http.StatusInternalServerError)
		return
	}

	// Insert post into the database
	db, err := sql.Open("sqlite3", "./Back-end/database/forum.db")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	stmt, err := db.Prepare("INSERT INTO posts (user_id, title, content, image, category_id, created_at) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		http.Error(w, "Error preparing query", http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	createdAt := time.Now()
	for _, categoryID := range categoryIDs {
		_, err = stmt.Exec(userID, title, content, imagePath, categoryID, createdAt)
		if err != nil {
			http.Error(w, "Error executing query", http.StatusInternalServerError)
			return
		}
	}

	// Redirect to a confirmation page or back to the home page
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
