package models

import (
	"database/sql"
	"fmt"
	"forum/Back-end/handlers"
	"log"
	"net/http"
)

// HandleAdmin handles admin panel requests including role assignments
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

	// Fetch posts and users
	posts, err := handlers.FetchPosts(db)
	if err != nil {
		http.Error(w, "Could not retrieve posts", http.StatusInternalServerError)
		return
	}

	users, err := handlers.FetchUsers(db)
	if err != nil {
		http.Error(w, "Could not retrieve users", http.StatusInternalServerError)
		return
	}

	// Check if user is logged in
	_, err = r.Cookie("user_id")
	loggedIn := err == nil
	data := map[string]interface{}{
		"LoggedIn": loggedIn,
		"Posts":    posts,
		"Users":    users,
	}

	if r.Method == http.MethodPost {
		action := r.FormValue("action")
		switch action {
		case "delete_user":
			username := r.FormValue("username")
			if err := deleteUser(db, username); err != nil {
				http.Error(w, "Failed to delete user: "+err.Error(), http.StatusInternalServerError)
				log.Println("Failed to delete user:", err)
				return
			}
			fmt.Fprintf(w, "Username %s deleted successfully", username)

		case "assign_role":
			userID := r.FormValue("user_id")
			if err := assignRoleToUser(db, userID); err != nil {
				http.Error(w, "Failed to assign moderator role: "+err.Error(), http.StatusInternalServerError)
				log.Println("Failed to assign moderator role:", err)
				return
			}
			fmt.Fprintf(w, "User ID %s has been assigned moderator role", userID)
		}

		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Could not execute template", http.StatusInternalServerError)
		return
	}
}

func HandleAssignRole(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		userID := r.FormValue("user_id")

		db, err := sql.Open("sqlite3", "./Back-end/database/forum.db")
		if err != nil {
			http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer db.Close()

		err = assignRoleToUser(db, userID)
		if err != nil {
			http.Error(w, "Failed to assign role: "+err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	} else {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}

func assignRoleToUser(db *sql.DB, userID string) error {
	// Kullanıcı rolünü moderatör olarak güncelleme işlemini buraya ekleyin
	_, err := db.Exec("UPDATE users SET role = 'moderator' WHERE id = ?", userID)
	return err
}

func HandleDeleteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		username := r.FormValue("username")

		db, err := sql.Open("sqlite3", "./Back-end/database/forum.db")
		if err != nil {
			http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer db.Close()

		err = deleteUser(db, username)
		if err != nil {
			http.Error(w, "Failed to delete user: "+err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	} else {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}

func deleteUser(db *sql.DB, username string) error {
	// Kullanıcıyı silme işlemini buraya ekleyin
	_, err := db.Exec("DELETE FROM users WHERE username = ?", username)
	return err
}
