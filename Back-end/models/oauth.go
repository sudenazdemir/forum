package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

var (
	githubOauthConfig = &oauth2.Config{
		ClientID:     "Ov23liJECUYvrVc2Bxyi",
		ClientSecret: "855297a4c778a144aa97607374162fd084acfd87",
		RedirectURL:  "http://localhost:8080/auth/github/callback",
		Scopes:       []string{"user:email"},
		Endpoint:     github.Endpoint,
	}
	googleOauthConfig = &oauth2.Config{
		ClientID:     "YOUR_GOOGLE_CLIENT_ID",
		ClientSecret: "YOUR_GOOGLE_CLIENT_SECRET",
		RedirectURL:  "http://localhost:8080/auth/google/callback",
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email"},
		Endpoint:     google.Endpoint,
	}
	oauthStateString = "randomstring"
)

func HandleGitHubLogin(w http.ResponseWriter, r *http.Request) {
	url := githubOauthConfig.AuthCodeURL(oauthStateString)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func HandleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	url := googleOauthConfig.AuthCodeURL(oauthStateString)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func HandleGitHubCallback(w http.ResponseWriter, r *http.Request) {
	state := r.FormValue("state")
	if state != oauthStateString {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	code := r.FormValue("code")
	token, err := githubOauthConfig.Exchange(oauth2.NoContext, code)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	client := githubOauthConfig.Client(oauth2.NoContext, token)
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	defer resp.Body.Close()

	var user struct {
		Login string `json:"login"`
		Email string `json:"email"`
	}
	err = json.NewDecoder(resp.Body).Decode(&user)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	if user.Email == "" {
		emailsResp, err := client.Get("https://api.github.com/user/emails")
		if err != nil {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}
		defer emailsResp.Body.Close()

		var emails []struct {
			Email    string `json:"email"`
			Primary  bool   `json:"primary"`
			Verified bool   `json:"verified"`
		}
		err = json.NewDecoder(emailsResp.Body).Decode(&emails)
		if err != nil {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}

		for _, email := range emails {
			if email.Primary && email.Verified {
				user.Email = email.Email
				break
			}
		}
	}

	// Store user in database
	userID := storeUserInDB(user.Login, user.Email, "", "github")

	// Set cookie with user ID
	cookie := http.Cookie{
		Name:     "user_id",
		Value:    fmt.Sprint(userID),
		HttpOnly: true,
		Path:     "/",
	}
	http.SetCookie(w, &cookie)

	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}

func HandleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	state := r.FormValue("state")
	if state != oauthStateString {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	code := r.FormValue("code")
	token, err := googleOauthConfig.Exchange(oauth2.NoContext, code)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	client := googleOauthConfig.Client(oauth2.NoContext, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	defer resp.Body.Close()

	var user struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	err = json.NewDecoder(resp.Body).Decode(&user)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	// Store user in database
	userID := storeUserInDB(user.Name, user.Email, "", "google")

	// Set cookie with user ID
	cookie := http.Cookie{
		Name:     "user_id",
		Value:    fmt.Sprint(userID),
		HttpOnly: true,
		Path:     "/",
	}
	http.SetCookie(w, &cookie)

	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}

func storeUserInDB(username, email, password, source string) int64 {
	db, err := sql.Open("sqlite3", "./Back-end/database/forum.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Check if user with email exists
	var existingUserID int64
	err = db.QueryRow("SELECT id FROM users WHERE email = ?", email).Scan(&existingUserID)
	switch {
	case err == sql.ErrNoRows: // User not found, insert new user
		insertUserQuery := `
				INSERT INTO users (username, email`
		if source == "github" || source == "google" {
			insertUserQuery += `, password`
		}
		insertUserQuery += `)
				VALUES (?, ?`
		if source == "github" || source == "google" {
			insertUserQuery += `, ?`
		}
		insertUserQuery += `)
			`

		var result sql.Result
		if source == "github" || source == "google" {
			result, err = db.Exec(insertUserQuery, username, email, password)
		} else {
			result, err = db.Exec(insertUserQuery, username, email)
		}
		if err != nil {
			log.Println("Error inserting user:", err)
			return 0
		}

		// Get last inserted user_id
		existingUserID, err = result.LastInsertId()
		if err != nil {
			log.Println("Error getting user ID:", err)
			return 0
		}

	case err != nil:
		log.Println("Error checking if user exists:", err)
		return 0
	}

	// Insert or update profile table
	insertProfileQuery := `
			INSERT INTO profile (user_id, username, email)
			VALUES (?, ?, ?)
			ON CONFLICT(user_id) DO UPDATE SET
			username = excluded.username,
			email = excluded.email
		`
	_, err = db.Exec(insertProfileQuery, existingUserID, username, email)
	if err != nil {
		log.Println("Error inserting profile:", err)
	}
	return existingUserID
}
