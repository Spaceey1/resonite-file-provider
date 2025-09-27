package admin

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"resonite-file-provider/authentication"
	"resonite-file-provider/database"
	"strconv"
	"time"
)

// AdminUser represents an admin user with additional fields
type AdminUser struct {
	ID            int        `json:"id"`
	Username      string     `json:"username"`
	IsAdmin       bool       `json:"is_admin"`
	StorageUsedMB float64    `json:"storage_used_mb"`
	CreatedAt     time.Time  `json:"created_at"`
	LastLogin     *time.Time `json:"last_login,omitempty"`
}

// UserStats represents user statistics for admin dashboard
type UserStats struct {
	TotalUsers             int   `json:"total_users"`
	AdminUsers             int   `json:"admin_users"`
	ActiveUsers            int   `json:"active_users"`
	CurrentlyLoggedInUsers int   `json:"currently_logged_in_users"`
	TotalStorageUsed       int64 `json:"total_storage_used"`
	TotalStorageQuota      int64 `json:"total_storage_quota"`
}

// InviteCode represents an invite code
type InviteCode struct {
	ID          int        `json:"id"`
	Code        string     `json:"code"`
	CreatedBy   int        `json:"created_by"`
	CreatedByUsername string `json:"created_by_username"`
	MaxUses     int        `json:"max_uses"`
	CurrentUses int        `json:"current_uses"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
}

// InviteCodeUsage represents usage of an invite code
type InviteCodeUsage struct {
	ID       int       `json:"id"`
	Username string    `json:"username"`
	UsedAt   time.Time `json:"used_at"`
}

// SystemSettings represents system configuration
type SystemSettings struct {
	RegistrationEnabled bool `json:"registration_enabled"`
	RequireInviteCode   bool `json:"require_invite_code"`
}

// AdminCheck verifies if the current user is an admin
func AdminCheck(w http.ResponseWriter, r *http.Request) *authentication.Claims {
	claims := authentication.AuthCheck(w, r)
	if claims == nil {
		return nil
	}

	var isAdmin bool
	err := database.Db.QueryRow("SELECT is_admin FROM Users WHERE id = ?", claims.UID).Scan(&isAdmin)
	if err != nil || !isAdmin {
		http.Error(w, "Admin access required", http.StatusForbidden)
		return nil
	}

	return claims
}

// LogAdminAction logs admin actions for audit purposes
func LogAdminAction(adminUserID int, action string, targetUserID *int, targetResource *string, details string, ipAddress string) error {
	_, err := database.Db.Exec(`
		INSERT INTO admin_audit_log (admin_user_id, action, target_user_id, target_resource, details, ip_address)
		VALUES (?, ?, ?, ?, ?, ?)
	`, adminUserID, action, targetUserID, targetResource, details, ipAddress)
	return err
}

// GetUserStats returns overall user statistics
func getUserStatsHandler(w http.ResponseWriter, r *http.Request) {
	claims := AdminCheck(w, r)
	if claims == nil {
		return
	}

	var stats UserStats
	
	// Get total users
	err := database.Db.QueryRow("SELECT COUNT(*) FROM Users").Scan(&stats.TotalUsers)
	if err != nil {
		http.Error(w, "Failed to get user count", http.StatusInternalServerError)
		return
	}

	// Get admin users count
	err = database.Db.QueryRow("SELECT COUNT(*) FROM Users WHERE is_admin = TRUE").Scan(&stats.AdminUsers)
	if err != nil {
		http.Error(w, "Failed to get admin count", http.StatusInternalServerError)
		return
	}

	// Get active users (logged in within last 30 days)
	err = database.Db.QueryRow("SELECT COUNT(*) FROM Users WHERE last_login > DATE_SUB(UTC_TIMESTAMP(), INTERVAL 30 DAY)").Scan(&stats.ActiveUsers)
	if err != nil {
		http.Error(w, "Failed to get active user count", http.StatusInternalServerError)
		return
	}

	// Calculate actual storage used (deduplicated)
	err = database.Db.QueryRow(`
		SELECT COALESCE((SELECT SUM(DISTINCT file_size_bytes) FROM storage_usage) / 1048576, 0)
	`).Scan(&stats.TotalStorageUsed)
	if err != nil {
		// If storage_usage table doesn't exist yet, default to 0
		stats.TotalStorageUsed = 0
	}
	
	// No quota system, so set quota to 0
	stats.TotalStorageQuota = 0

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    stats,
	})
}

// GetAllUsers returns all users with their details
func getAllUsersHandler(w http.ResponseWriter, r *http.Request) {
	claims := AdminCheck(w, r)
	if claims == nil {
		return
	}

	rows, err := database.Db.Query(`
		SELECT id, username, is_admin, created_at, last_login
		FROM Users
		ORDER BY created_at DESC
	`)
	if err != nil {
		http.Error(w, "Failed to get users", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []AdminUser
	for rows.Next() {
		var user AdminUser
		var lastLogin sql.NullTime
		
		err := rows.Scan(&user.ID, &user.Username, &user.IsAdmin, 
			&user.CreatedAt, &lastLogin)
		if err != nil {
			http.Error(w, "Failed to scan user data", http.StatusInternalServerError)
			return
		}
		
		if lastLogin.Valid {
			user.LastLogin = &lastLogin.Time
		}
		
		// Calculate actual storage used by this user (deduplicated)
		err = database.Db.QueryRow(`
			SELECT COALESCE(SUM(DISTINCT su.file_size_bytes) / 1048576, 0)
			FROM storage_usage su
			WHERE su.user_id = ?
		`, user.ID).Scan(&user.StorageUsedMB)
		
		if err != nil {
			// Fallback: calculate based on user's items and asset hashes
			err = database.Db.QueryRow(`
				SELECT COALESCE(SUM(DISTINCT a.file_size_bytes) / 1048576, 0)
				FROM Items i
				JOIN Folders f ON i.folder_id = f.id
				JOIN Inventories inv ON f.inventory_id = inv.id
				JOIN users_inventories ui ON inv.id = ui.inventory_id
				JOIN `+"`hash-usage`"+` hu ON i.id = hu.item_id
				JOIN Assets a ON hu.asset_id = a.id
				WHERE ui.user_id = ?
			`, user.ID).Scan(&user.StorageUsedMB)
			
			if err != nil {
				// If we can't calculate, default to 0
				user.StorageUsedMB = 0
			}
		}
		
		users = append(users, user)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    users,
	})
}


// DeleteUser deletes a user and all their data
func deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	claims := AdminCheck(w, r)
	if claims == nil {
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		http.Error(w, "User ID required", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Prevent admin from deleting themselves
	if userID == claims.UID {
		http.Error(w, "Cannot delete your own account", http.StatusBadRequest)
		return
	}

	// Get username for logging
	var username string
	err = database.Db.QueryRow("SELECT username FROM Users WHERE id = ?", userID).Scan(&username)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Start transaction
	tx, err := database.Db.Begin()
	if err != nil {
		http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Delete user (cascading deletes will handle related data)
	_, err = tx.Exec("DELETE FROM Users WHERE id = ?", userID)
	if err != nil {
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	// Log the admin action
	LogAdminAction(claims.UID, "DELETE_USER", &userID, nil, 
		fmt.Sprintf("Deleted user: %s", username), r.RemoteAddr)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "User deleted successfully",
	})
}

// ToggleAdminStatus toggles admin status for a user
func toggleAdminStatusHandler(w http.ResponseWriter, r *http.Request) {
	claims := AdminCheck(w, r)
	if claims == nil {
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var request struct {
		UserID  int  `json:"user_id"`
		IsAdmin bool `json:"is_admin"`
	}

	if err := json.Unmarshal(body, &request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Prevent admin from removing their own admin status
	if request.UserID == claims.UID && !request.IsAdmin {
		http.Error(w, "Cannot remove your own admin status", http.StatusBadRequest)
		return
	}

	// Update admin status
	_, err = database.Db.Exec("UPDATE Users SET is_admin = ? WHERE id = ?", request.IsAdmin, request.UserID)
	if err != nil {
		http.Error(w, "Failed to update admin status", http.StatusInternalServerError)
		return
	}

	// Log the admin action
	action := "GRANT_ADMIN"
	if !request.IsAdmin {
		action = "REVOKE_ADMIN"
	}
	LogAdminAction(claims.UID, action, &request.UserID, nil, 
		fmt.Sprintf("Changed admin status to %t", request.IsAdmin), r.RemoteAddr)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Admin status updated successfully",
	})
}

// GetUserAssets returns all assets for a specific user
func getUserAssetsHandler(w http.ResponseWriter, r *http.Request) {
	claims := AdminCheck(w, r)
	if claims == nil {
		return
	}

	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		http.Error(w, "User ID required", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	rows, err := database.Db.Query(`
		SELECT DISTINCT i.id, i.name, i.url, f.name as folder_name, inv.name as inventory_name
		FROM Items i
		JOIN Folders f ON i.folder_id = f.id
		JOIN Inventories inv ON f.inventory_id = inv.id
		JOIN users_inventories ui ON inv.id = ui.inventory_id
		WHERE ui.user_id = ?
		ORDER BY i.name
	`, userID)
	if err != nil {
		http.Error(w, "Failed to get user assets", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type UserAsset struct {
		ID            int    `json:"id"`
		Name          string `json:"name"`
		URL           string `json:"url"`
		FolderName    string `json:"folder_name"`
		InventoryName string `json:"inventory_name"`
	}

	var assets []UserAsset
	for rows.Next() {
		var asset UserAsset
		err := rows.Scan(&asset.ID, &asset.Name, &asset.URL, &asset.FolderName, &asset.InventoryName)
		if err != nil {
			http.Error(w, "Failed to scan asset data", http.StatusInternalServerError)
			return
		}
		assets = append(assets, asset)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    assets,
	})
}

// DeleteUserAsset deletes a specific asset from a user
func deleteUserAssetHandler(w http.ResponseWriter, r *http.Request) {
	claims := AdminCheck(w, r)
	if claims == nil {
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	itemIDStr := r.URL.Query().Get("item_id")
	if itemIDStr == "" {
		http.Error(w, "Item ID required", http.StatusBadRequest)
		return
	}

	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		http.Error(w, "Invalid item ID", http.StatusBadRequest)
		return
	}

	// Get item details for logging
	var itemName string
	var userID int
	err = database.Db.QueryRow(`
		SELECT i.name, ui.user_id
		FROM Items i
		JOIN Folders f ON i.folder_id = f.id
		JOIN Inventories inv ON f.inventory_id = inv.id
		JOIN users_inventories ui ON inv.id = ui.inventory_id
		WHERE i.id = ?
	`, itemID).Scan(&itemName, &userID)
	if err != nil {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}

	// Delete the item
	_, err = database.Db.Exec("DELETE FROM Items WHERE id = ?", itemID)
	if err != nil {
		http.Error(w, "Failed to delete item", http.StatusInternalServerError)
		return
	}

	// Log the admin action
	LogAdminAction(claims.UID, "DELETE_ASSET", &userID, nil, 
		fmt.Sprintf("Deleted asset: %s (ID: %d)", itemName, itemID), r.RemoteAddr)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Asset deleted successfully",
	})
}

// GetSystemSettings returns current system settings
func getSystemSettingsHandler(w http.ResponseWriter, r *http.Request) {
	claims := AdminCheck(w, r)
	if claims == nil {
		return
	}

	var settings SystemSettings
	
	// Get registration enabled setting
	var regEnabled string
	err := database.Db.QueryRow("SELECT setting_value FROM system_settings WHERE setting_key = 'registration_enabled'").Scan(&regEnabled)
	if err != nil {
		settings.RegistrationEnabled = true // Default value
	} else {
		settings.RegistrationEnabled = regEnabled == "true"
	}
	
	// Get require invite code setting
	var requireInvite string
	err = database.Db.QueryRow("SELECT setting_value FROM system_settings WHERE setting_key = 'require_invite_code'").Scan(&requireInvite)
	if err != nil {
		settings.RequireInviteCode = false // Default value
	} else {
		settings.RequireInviteCode = requireInvite == "true"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    settings,
	})
}

// UpdateSystemSettings updates system settings
func updateSystemSettingsHandler(w http.ResponseWriter, r *http.Request) {
	claims := AdminCheck(w, r)
	if claims == nil {
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var settings SystemSettings
	if err := json.Unmarshal(body, &settings); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Update registration enabled setting
	regValue := "false"
	if settings.RegistrationEnabled {
		regValue = "true"
	}
	_, err = database.Db.Exec(`
		INSERT INTO system_settings (setting_key, setting_value, description) 
		VALUES ('registration_enabled', ?, 'Enable/disable public registration')
		ON DUPLICATE KEY UPDATE setting_value = VALUES(setting_value)
	`, regValue)
	if err != nil {
		http.Error(w, "Failed to update registration setting", http.StatusInternalServerError)
		return
	}

	// Update require invite code setting
	inviteValue := "false"
	if settings.RequireInviteCode {
		inviteValue = "true"
	}
	_, err = database.Db.Exec(`
		INSERT INTO system_settings (setting_key, setting_value, description) 
		VALUES ('require_invite_code', ?, 'Require invite code for registration')
		ON DUPLICATE KEY UPDATE setting_value = VALUES(setting_value)
	`, inviteValue)
	if err != nil {
		http.Error(w, "Failed to update invite code setting", http.StatusInternalServerError)
		return
	}

	// Log the admin action
	LogAdminAction(claims.UID, "UPDATE_SETTINGS", nil, nil, 
		fmt.Sprintf("Updated system settings: registration_enabled=%t, require_invite_code=%t", 
			settings.RegistrationEnabled, settings.RequireInviteCode), r.RemoteAddr)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Settings updated successfully",
	})
}

// GetInviteCodes returns all invite codes
func getInviteCodesHandler(w http.ResponseWriter, r *http.Request) {
	claims := AdminCheck(w, r)
	if claims == nil {
		return
	}

	rows, err := database.Db.Query(`
		SELECT ic.id, ic.code, ic.created_by, u.username, ic.max_uses, ic.current_uses, 
		       ic.expires_at, ic.is_active, ic.created_at
		FROM invite_codes ic
		JOIN Users u ON ic.created_by = u.id
		ORDER BY ic.created_at DESC
	`)
	if err != nil {
		http.Error(w, "Failed to get invite codes", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var inviteCodes []InviteCode
	for rows.Next() {
		var code InviteCode
		var expiresAt sql.NullTime
		
		err := rows.Scan(&code.ID, &code.Code, &code.CreatedBy, &code.CreatedByUsername,
			&code.MaxUses, &code.CurrentUses, &expiresAt, &code.IsActive, &code.CreatedAt)
		if err != nil {
			http.Error(w, "Failed to scan invite code data", http.StatusInternalServerError)
			return
		}
		
		if expiresAt.Valid {
			code.ExpiresAt = &expiresAt.Time
		}
		
		inviteCodes = append(inviteCodes, code)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    inviteCodes,
	})
}

// CreateInviteCode creates a new invite code
func createInviteCodeHandler(w http.ResponseWriter, r *http.Request) {
	claims := AdminCheck(w, r)
	if claims == nil {
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var request struct {
		MaxUses   int    `json:"max_uses"`
		ExpiresAt string `json:"expires_at,omitempty"`
	}

	if err := json.Unmarshal(body, &request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Generate a random invite code
	code := generateInviteCode()

	// Parse expiration date if provided
	var expiresAt *time.Time
	if request.ExpiresAt != "" {
		parsed, err := time.Parse("2006-01-02T15:04", request.ExpiresAt)
		if err != nil {
			http.Error(w, "Invalid expiration date format", http.StatusBadRequest)
			return
		}
		expiresAt = &parsed
	}

	// Insert the invite code
	_, err = database.Db.Exec(`
		INSERT INTO invite_codes (code, created_by, max_uses, expires_at)
		VALUES (?, ?, ?, ?)
	`, code, claims.UID, request.MaxUses, expiresAt)
	if err != nil {
		http.Error(w, "Failed to create invite code", http.StatusInternalServerError)
		return
	}

	// Log the admin action
	LogAdminAction(claims.UID, "CREATE_INVITE_CODE", nil, nil, 
		fmt.Sprintf("Created invite code: %s (max_uses: %d)", code, request.MaxUses), r.RemoteAddr)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Invite code created successfully",
		"code":    code,
	})
}

// DeleteInviteCode deletes an invite code
func deleteInviteCodeHandler(w http.ResponseWriter, r *http.Request) {
	claims := AdminCheck(w, r)
	if claims == nil {
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	codeIDStr := r.URL.Query().Get("code_id")
	if codeIDStr == "" {
		http.Error(w, "Code ID required", http.StatusBadRequest)
		return
	}

	codeID, err := strconv.Atoi(codeIDStr)
	if err != nil {
		http.Error(w, "Invalid code ID", http.StatusBadRequest)
		return
	}

	// Get code for logging
	var code string
	err = database.Db.QueryRow("SELECT code FROM invite_codes WHERE id = ?", codeID).Scan(&code)
	if err != nil {
		http.Error(w, "Invite code not found", http.StatusNotFound)
		return
	}

	// Delete the invite code
	_, err = database.Db.Exec("DELETE FROM invite_codes WHERE id = ?", codeID)
	if err != nil {
		http.Error(w, "Failed to delete invite code", http.StatusInternalServerError)
		return
	}

	// Log the admin action
	LogAdminAction(claims.UID, "DELETE_INVITE_CODE", nil, nil, 
		fmt.Sprintf("Deleted invite code: %s", code), r.RemoteAddr)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Invite code deleted successfully",
	})
}

// GetInviteCodeUsage returns usage details for an invite code
func getInviteCodeUsageHandler(w http.ResponseWriter, r *http.Request) {
	claims := AdminCheck(w, r)
	if claims == nil {
		return
	}

	codeIDStr := r.URL.Query().Get("code_id")
	if codeIDStr == "" {
		http.Error(w, "Code ID required", http.StatusBadRequest)
		return
	}

	codeID, err := strconv.Atoi(codeIDStr)
	if err != nil {
		http.Error(w, "Invalid code ID", http.StatusBadRequest)
		return
	}

	rows, err := database.Db.Query(`
		SELECT icu.id, u.username, icu.used_at
		FROM invite_code_usage icu
		JOIN Users u ON icu.used_by = u.id
		WHERE icu.invite_code_id = ?
		ORDER BY icu.used_at DESC
	`, codeID)
	if err != nil {
		http.Error(w, "Failed to get invite code usage", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var usage []InviteCodeUsage
	for rows.Next() {
		var u InviteCodeUsage
		err := rows.Scan(&u.ID, &u.Username, &u.UsedAt)
		if err != nil {
			http.Error(w, "Failed to scan usage data", http.StatusInternalServerError)
			return
		}
		usage = append(usage, u)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    usage,
	})
}

// generateInviteCode generates a random invite code
func generateInviteCode() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 8
	
	code := make([]byte, length)
	for i := range code {
		code[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		time.Sleep(1) // Simple way to get different values
	}
	return string(code)
}

// AddAdminListeners registers all admin endpoints
func AddAdminListeners() {
	http.HandleFunc("/admin/stats", getUserStatsHandler)
	http.HandleFunc("/admin/users", getAllUsersHandler)
	http.HandleFunc("/admin/users/delete", deleteUserHandler)
	http.HandleFunc("/admin/users/admin-status", toggleAdminStatusHandler)
	http.HandleFunc("/admin/users/assets", getUserAssetsHandler)
	http.HandleFunc("/admin/users/assets/delete", deleteUserAssetHandler)
	
	// System settings endpoints
	http.HandleFunc("/admin/settings", getSystemSettingsHandler)
	http.HandleFunc("/admin/settings/update", updateSystemSettingsHandler)
	
	// Invite code endpoints
	http.HandleFunc("/admin/invite-codes", getInviteCodesHandler)
	http.HandleFunc("/admin/invite-codes/create", createInviteCodeHandler)
	http.HandleFunc("/admin/invite-codes/delete", deleteInviteCodeHandler)
	http.HandleFunc("/admin/invite-codes/usage", getInviteCodeUsageHandler)
}
