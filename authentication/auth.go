package authentication

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"resonite-file-provider/database"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func hashPassword(password string) string {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	return string(bytes)
}
func readBody(r *http.Request) (string, string, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return "", "", err
	}
	bodyString := string(body)
	// Non standard way to read the body for ease of use in Resonite
	creds := strings.Split(bodyString, "\n")
	if len(creds) < 2 {
		return "", "", fmt.Errorf("invalid credentials format")
	}
	username := creds[0]
	password := creds[1]
	return username, password, nil
}
func registerHandler(w http.ResponseWriter, r *http.Request) {
	username, password, err := readBody(r)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		fmt.Println("Read error:", err)
		return
	}
	if username == "" || password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}
	var exists bool
	err = database.Db.QueryRow("SELECT EXISTS(SELECT 1 FROM Users WHERE username = ?)", username).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		http.Error(w, "Server error", http.StatusInternalServerError)
		fmt.Println("Query error:", err)
		return
	}
	if exists {
		http.Error(w, "Username already exists", http.StatusConflict)
		return
	}
	hashedPassword := hashPassword(password)
	
	// Start transaction to create user and default inventory
	tx, err := database.Db.Begin()
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		fmt.Println("Transaction error:", err)
		return
	}
	defer tx.Rollback()
	
	// Insert user
	result, err := tx.Exec("INSERT INTO `Users`(`username`, `auth`) VALUES (?, ?)", username, hashedPassword)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		fmt.Println("Insert user error:", err)
		return
	}
	
	// Get the new user ID
	userID, err := result.LastInsertId()
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		fmt.Println("Get user ID error:", err)
		return
	}
	
	// Create default inventory
	inventoryResult, err := tx.Exec("INSERT INTO `Inventories`(`name`) VALUES (?)", "Inventory")
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		fmt.Println("Insert inventory error:", err)
		return
	}
	
	// Get the new inventory ID
	inventoryID, err := inventoryResult.LastInsertId()
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		fmt.Println("Get inventory ID error:", err)
		return
	}
	
	// Link user to inventory
	_, err = tx.Exec("INSERT INTO `users_inventories`(`user_id`, `inventory_id`) VALUES (?, ?)", userID, inventoryID)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		fmt.Println("Link user to inventory error:", err)
		return
	}
	
	// Create root folder for the inventory
	_, err = tx.Exec("INSERT INTO `Folders`(`name`, `parent_folder_id`, `inventory_id`) VALUES (?, ?, ?)", "Root", -1, inventoryID)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		fmt.Println("Create root folder error:", err)
		return
	}
	
	// Commit transaction
	if err = tx.Commit(); err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		fmt.Println("Commit error:", err)
		return
	}
	
	fmt.Printf("User registered successfully: %s with default inventory\n", username)
	w.Write([]byte("User registered successfully"))
}
func loginHandler(w http.ResponseWriter, r *http.Request) {
	username, password, err := readBody(r)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		fmt.Println("Read error:", err)
		return
	}
	var storedHash string
	var uId int
	err = database.Db.QueryRow("SELECT auth, id FROM Users WHERE username = ?", username).Scan(&storedHash, &uId)
	if err == sql.ErrNoRows {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	} else if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		fmt.Println("Query error:", err)
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password)); err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	_, err = database.Db.Exec("UPDATE Users SET last_login = ? WHERE id = ?", time.Now().UTC(), uId)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		fmt.Println("Update last_login error:", err)
		return
	}
	token, err := GenerateToken(username, uId)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	now := time.Now().UTC()
	database.Db.Exec("DELETE FROM active_sessions WHERE expires_at < ?", now)
	expiresAt := now.Add(24 * time.Hour)
	_, err = database.Db.Exec("INSERT INTO active_sessions (user_id, token, expires_at, last_seen) VALUES (?, ?, ?, ?) ON DUPLICATE KEY UPDATE expires_at = VALUES(expires_at), last_seen = VALUES(last_seen)", uId, token, expiresAt, now)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		fmt.Println("Upsert active session error:", err)
		return
	}

	// Set the auth token as a cookie with detailed settings for troubleshooting
	cookie := &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		MaxAge:   86400, // 1 day
		HttpOnly: false, // Allow JavaScript access for debugging
		SameSite: http.SameSiteLaxMode,
		Secure:   false, // Since we're in development
	}

	http.SetCookie(w, cookie)

	// Log the cookie details
	fmt.Printf("[AUTH] Setting cookie: %s=%s; Path=%s; MaxAge=%d; HttpOnly=%t; SameSite=%v\n",
		cookie.Name, cookie.Value[:10]+"...", cookie.Path, cookie.MaxAge, cookie.HttpOnly, cookie.SameSite)

	fmt.Printf("[AUTH] Login successful for user: %s\n", username)

	// Also return the token in the response body for non-browser clients
	w.Write([]byte(token))
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	token := ""
	if authCookie, err := r.Cookie("auth_token"); err == nil {
		token = authCookie.Value
	}
	if token == "" {
		token = r.URL.Query().Get("auth")
	}
	if token != "" {
		database.Db.Exec("DELETE FROM active_sessions WHERE token = ?", token)
	}

	http.SetCookie(w, &http.Cookie{
		Name:   "auth_token",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	w.Write([]byte("Logged out"))
}

func AuthCheck(w http.ResponseWriter, r *http.Request) *Claims {
	// Log cookies
	cookies := r.Cookies()
	fmt.Println("[AUTH] Request cookies:", cookies)
	var auth string
	// First try cookie (preferred)
	authCookie, err := r.Cookie("auth_token")
	if err == nil {
		auth = authCookie.Value
		fmt.Println("[AUTH] Found auth_token cookie:", auth[:10]+"...")
	} else {
		// Fallback to query parameter
		auth = r.URL.Query().Get("auth")
		if auth != "" {
			fmt.Println("[AUTH] Found auth in query param:", auth[:10]+"...")
		}
	}
	if auth == "" {
		// Log debug information
		fmt.Println("[AUTH] No auth token found in cookie or query param")

		if strings.HasPrefix(r.UserAgent(), "Resonite") {
			http.Error(w, "No auth token found in param", http.StatusUnauthorized)
		} else {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "Auth token missing",
			})
		}
		return nil
	}
	claims, err := ParseToken(auth)
	if err != nil {
		if strings.HasPrefix(r.UserAgent(), "Resonite") {
			http.Error(w, "Auth token missing or invalid", http.StatusUnauthorized)
		} else {
			fmt.Println("[AUTH] Auth token invalid:", err.Error())
			// Return JSON error instead of HTML error
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "Auth token invalid: " + err.Error(),
			})
		}
		return nil
	}
	fmt.Println("[AUTH] Auth successful for user ID:", claims.UID, "Username:", claims.Username)

	now := time.Now().UTC()
	expires := now.Add(24 * time.Hour)
	if _, err := database.Db.Exec("UPDATE active_sessions SET last_seen = ?, expires_at = ? WHERE token = ?", now, expires, auth); err != nil {
		fmt.Println("[AUTH] Failed to refresh active session:", err)
	}

	return claims
}

// Call this before starting the server
func AddAuthListeners() {
	http.HandleFunc("/auth/login", loginHandler)
	http.HandleFunc("/auth/register", registerHandler)
}








