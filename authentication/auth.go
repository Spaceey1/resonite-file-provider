package authentication

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"resonite-file-provider/database"
	"strings"

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
	_, err = database.Db.Exec("INSERT INTO `Users`(`username`, `auth`) VALUES (?, ?); ", username, hashedPassword)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		fmt.Println("Insert error:", err)
		return
	}
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
	token, err := GenerateToken(username, uId);
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
	}
	
	w.Write([]byte(token))
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	// Handle logout logic here
	w.Write([]byte("Logout handler"))
}
// Call this before starting the server
func AddAuthListeners() {
	http.HandleFunc("/auth/login", loginHandler)
	http.HandleFunc("/auth/register", registerHandler)
}
