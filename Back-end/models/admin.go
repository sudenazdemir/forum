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
