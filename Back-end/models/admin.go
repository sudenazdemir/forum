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
	modRequests, err := handlers.FetchModRequests(db)
	if err != nil {
		http.Error(w, "Could not retrieve moderator requests", http.StatusInternalServerError)
		return
	}

	// Check if user is logged in
	_, err = r.Cookie("user_id")
	loggedIn := err == nil
	data := map[string]interface{}{
		"LoggedIn":    loggedIn,
		"Posts":       posts,
		"Users":       users,
		"ModRequests": modRequests,
	}

	if r.Method == http.MethodPost {
		action := r.FormValue("action")
		username := r.FormValue("username")
		switch action {
		case "delete_user":

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

		case "approve_mod":
			if err := approveModRequest(db, username); err != nil {
				http.Error(w, "Failed to approve moderator request: "+err.Error(), http.StatusInternalServerError)
				log.Println("Failed to approve moderator request:", err)
				return
			}

		case "reject_mod":
			if err := rejectModRequest(db, username); err != nil {
				http.Error(w, "Failed to reject moderator request: "+err.Error(), http.StatusInternalServerError)
				log.Println("Failed to reject moderator request:", err)
				return
			}
		default:
			http.Error(w, "Invalid action", http.StatusBadRequest)
			log.Println("Invalid action:", action) // Log the invalid action
			return

		}

		http.Redirect(w, r, "/panel", http.StatusSeeOther)
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

		http.Redirect(w, r, "/panel	", http.StatusSeeOther)
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

		http.Redirect(w, r, "/panel", http.StatusSeeOther)
	} else {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}

func deleteUser(db *sql.DB, username string) error {
	// Kullanıcıyı silme işlemini buraya ekleyin
	_, err := db.Exec("DELETE FROM users WHERE username = ?", username)
	return err
}

func approveModRequest(db *sql.DB, username string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	// Get the user ID from the username
	var userID int
	if err := tx.QueryRow("SELECT id FROM users WHERE username = ?", username).Scan(&userID); err != nil {
		tx.Rollback()
		return err
	}

	// Update the user role to 'moderator'
	if _, err := tx.Exec("UPDATE users SET role = 'moderator' WHERE id = ?", userID); err != nil {
		tx.Rollback()
		return err
	}

	// Update the mod request status to 'approved'
	if _, err := tx.Exec("UPDATE mod_requests SET status = 'approved' WHERE user_id = ?", userID); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func rejectModRequest(db *sql.DB, username string) error {
	var userID int
	if err := db.QueryRow("SELECT id FROM users WHERE username = ?", username).Scan(&userID); err != nil {
		return err
	}

	_, err := db.Exec("UPDATE mod_requests SET status = 'rejected' WHERE user_id = ?", userID)
	return err
}
