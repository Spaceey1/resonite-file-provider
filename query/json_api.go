package query

import (
	"encoding/json"
	"net/http"
	"resonite-file-provider/authentication"
	"resonite-file-provider/database"
	"strconv"
)

// JSON response structures for web API
type InventoriesResponse struct {
	Success bool                `json:"success"`
	Data    []InventoryListItem `json:"data"`
}

type InventoryListItem struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	RootFolderId int    `json:"rootFolderId"`
}

type InventoryRootResponse struct {
	Success     bool `json:"success"`
	RootFolderId int  `json:"rootFolderId"`
}

type FoldersResponse struct {
	Success bool                `json:"success"`
	Data    []FolderListItem    `json:"data"`
	Parent  *ParentFolderInfo   `json:"parent,omitempty"`
}

type FolderListItem struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type ParentFolderInfo struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type ItemsResponse struct {
	Success bool             `json:"success"`
	Data    []ItemListItem   `json:"data"`
}

type ItemListItem struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

type FolderContentsResponse struct {
	Success bool             `json:"success"`
	Folders []FolderListItem `json:"folders"`
	Items   []ItemListItem   `json:"items"`
	Parent  *ParentFolderInfo `json:"parent,omitempty"`
}

// Handler for JSON API endpoints for web interface

// getInventoryRootFolder handles GET /api/inventory/rootFolder
func getInventoryRootFolderAPI(w http.ResponseWriter, r *http.Request) {
    inventoryId, err := strconv.Atoi(r.URL.Query().Get("inventoryId"))
    if err != nil {
        http.Error(w, "inventoryId is either not specified or is invalid", http.StatusBadRequest)
        return
    }
    
	claims := authentication.AuthCheck(w, r)
	if claims == nil {
		http.Error(w, "[FolderContents] Failed Auth", http.StatusUnauthorized)
		return
	}
    
    // Check if user has access to this inventory
    var hasAccess bool
    err = database.Db.QueryRow(`
        SELECT EXISTS(
            SELECT 1 
            FROM users_inventories 
            WHERE user_id = ? AND inventory_id = ?
        )
    `, claims.UID, inventoryId).Scan(&hasAccess)
    
    if err != nil {
        http.Error(w, "Error checking access: "+err.Error(), http.StatusInternalServerError)
        return
    }
    
    if !hasAccess {
        http.Error(w, "You don't have access to this inventory", http.StatusForbidden)
        return
    }
    
    // Set JSON content type
    w.Header().Set("Content-Type", "application/json")
    
    // Get the root folder ID
    var rootFolderId int
    err = database.Db.QueryRow(
        "SELECT id FROM Folders WHERE `inventory_id` = ? AND parent_folder_id = -1",
		inventoryId,
    ).Scan(&rootFolderId)
    
    if err != nil {
        response := InventoryRootResponse{
            Success: false,
        }
        json.NewEncoder(w).Encode(response)
        return
    }
    
    response := InventoryRootResponse{
        Success: true,
        RootFolderId: rootFolderId,
    }
    
    json.NewEncoder(w).Encode(response)
}

// getUserStorageInfo handles GET /api/user/storage
func getUserStorageInfoAPI(w http.ResponseWriter, r *http.Request) {
	claims := authentication.AuthCheck(w, r)
	if claims == nil {
		return
	}

	// Set JSON content type
	w.Header().Set("Content-Type", "application/json")

	// Get user's admin status
	var isAdmin bool
	var storageUsedMB float64
	
	// Get admin status
	err := database.Db.QueryRow(`
		SELECT is_admin 
		FROM Users 
		WHERE id = ?
	`, claims.UID).Scan(&isAdmin)

	if err != nil {
		isAdmin = false
	}

	// Calculate actual storage used by this user (deduplicated)
	err = database.Db.QueryRow(`
		SELECT COALESCE(SUM(DISTINCT su.file_size_bytes) / 1048576.0, 0)
		FROM storage_usage su
		WHERE su.user_id = ?
	`, claims.UID).Scan(&storageUsedMB)
	
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
		`, claims.UID).Scan(&storageUsedMB)
		
		if err != nil {
			// If we can't calculate, default to 0
			storageUsedMB = 0
		}
	}

	response := map[string]interface{}{
		"success":         true,
		"storage_used_mb": storageUsedMB,
		"is_admin":        isAdmin,
	}

	json.NewEncoder(w).Encode(response)
}

// AddJSONAPIListeners registers the JSON API endpoints
func AddJSONAPIListeners() {
	http.HandleFunc("/api/inventory/rootFolder", getInventoryRootFolderAPI)
	http.HandleFunc("/api/user/storage", getUserStorageInfoAPI)
}
