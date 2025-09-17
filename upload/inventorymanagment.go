package upload

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"resonite-file-provider/authentication"
	"resonite-file-provider/config"
	"resonite-file-provider/database"
	"resonite-file-provider/query"
	"strconv"
	"strings"
)

func AddInventory(userID int, inventoryName string) (int64, int64, error) {
	res, err := database.Db.Exec(`INSERT INTO Inventories (name) VALUES (?)`, inventoryName)
	if err != nil {
		return -1, -1, err
	}
	invID, err := res.LastInsertId()
	if err != nil {
		return -1, -1, err
	}
	_, err = database.Db.Exec(`INSERT INTO users_inventories (user_id, inventory_id) VALUES (?, ?)`, userID, invID)
	if err != nil {
		return -1, -1, err
	}
	res, err = database.Db.Exec(`INSERT INTO Folders (name, parent_folder_id, inventory_id) VALUES (?, ?, ?)`, "root", -1, invID)

	if err != nil {
		return -1, -1, err
	}
	folderID, err := res.LastInsertId()
	if err != nil {
		return -1, -1, err
	}
	return invID, folderID, nil
}

func AddFolder(parentFolderID int, folderName string) (int64, error) {
	if folderName == "" {
		return -1, fmt.Errorf("Folder name was not specified")
	}
	result, err := database.Db.Exec(`
		INSERT INTO Folders (name, parent_folder_id, inventory_id)
		SELECT ?, ?, t.inventory_id
		FROM (SELECT inventory_id FROM Folders WHERE id = ?) AS t;
		`,
		folderName, parentFolderID, parentFolderID,
	)
	if err != nil {
		return -1, err
	}
	newFolderId, err := result.LastInsertId()
	if err != nil {
		return -1, err
	}
	return newFolderId, nil
}

func handleAddFolder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
	// Log cookies
	cookies := r.Cookies()
	fmt.Println("[FOLDER] Request cookies:", cookies)
	var auth string
	// First try cookie (preferred)
	authCookie, err := r.Cookie("auth_token")
	if err == nil {
		auth = authCookie.Value
		fmt.Println("[FOLDER] Found auth_token cookie:", auth[:10]+"...")
	} else {
		// Fallback to query parameter
		auth = r.URL.Query().Get("auth")
		if auth != "" {
			fmt.Println("[FOLDER] Found auth in query param:", auth[:10]+"...")
		}
	}
	if auth == "" {
		// Log debug information
		fmt.Println("[FOLDER] No auth token found in cookie or query param")

		// Return JSON error instead of HTML error
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Auth token missing",
		})
		return
	}
	claims, err := authentication.ParseToken(auth)
	if err != nil {
		http.Error(w, "Auth token missing or invalid", http.StatusUnauthorized)
		fmt.Println("[FOLDER] Auth token invalid:", err.Error())
		// Return JSON error instead of HTML error
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Auth token invalid: " + err.Error(),
		})
		return
	}
	fmt.Println("[FOLDER] Auth successful for user ID:", claims.UID, "Username:", claims.Username)
	folderId, err := strconv.Atoi(r.URL.Query().Get("folderId"))
	if err != nil {
		http.Error(w, "folderId missing or invalid", http.StatusBadRequest)
		fmt.Println("[FOLDER] Invalid folder ID:", err.Error())
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "folderId missing or invalid",
		})
		return
	}
	if allowed, err := query.IsFolderOwner(folderId, claims.UID); err != nil || !allowed {
		http.Error(w, "Forbidden", http.StatusForbidden)
		fmt.Println("[FOLDER] Access denied to folder ID:", folderId, "for user:", claims.Username)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "You don't have permission to create folders here",
		})
		return
	}
	folderName := r.URL.Query().Get("folderName")
	if folderName == "" {
		http.Error(w, "folderName missing", http.StatusBadRequest)
		fmt.Println("[FOLDER] Missing folder name")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "folderName parameter is missing",
		})
		return
	}
	fmt.Println("[FOLDER] Creating folder:", folderName, "in parent folder ID:", folderId)
	newFolderId, err := AddFolder(folderId, folderName)
	if err != nil {
		http.Error(w, "Error while creating folder", http.StatusInternalServerError)
		fmt.Println("[FOLDER] Database error creating folder:", err.Error())
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to add folder: " + err.Error(),
		})
		return
	}
	if strings.HasPrefix(r.UserAgent(), "Resonite") {
		w.Write([]byte(strconv.FormatInt(newFolderId, 10)))
	} else {
		// Return success response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"folderId": newFolderId,
			"name":     folderName,
			"parentId": folderId,
		})
	}
	fmt.Println("[FOLDER] AddFolder request received:", r.Method, r.URL.String())
	fmt.Println("[FOLDER] Request headers:", r.Header)

	if r.Method != http.MethodPost {
		fmt.Println("[FOLDER] Invalid method:", r.Method)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid request method",
		})
		return
	}

}

func handleAddInventory(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[INVENTORY] AddInventory request received:", r.Method, r.URL.String())
	fmt.Println("[INVENTORY] Request headers:", r.Header)
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		fmt.Println("[INVENTORY] Invalid method:", r.Method)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid request method",
		})
		return
	}
	// Log cookies
	cookies := r.Cookies()
	fmt.Println("[INVENTORY] Request cookies:", cookies)

	// Try to get auth token from multiple sources
	var auth string

	// First try cookie (preferred)
	authCookie, err := r.Cookie("auth_token")
	if err == nil {
		auth = authCookie.Value
		fmt.Println("[INVENTORY] Found auth_token cookie:", auth[:10]+"...")
	} else {
		// Fallback to query parameter
		auth = r.URL.Query().Get("auth")
		if auth != "" {
			fmt.Println("[INVENTORY] Found auth in query param:", auth[:10]+"...")
		}
	}
	if auth == "" {
		// Log debug information
		fmt.Println("[INVENTORY] No auth token found in cookie or query param")

		// Return JSON error instead of HTML error
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Auth token missing",
		})
		return
	}

	claims, err := authentication.ParseToken(auth)
	if err != nil {
		http.Error(w, "Auth token missing or invalid", http.StatusUnauthorized)
		fmt.Println("[INVENTORY] Auth token invalid:", err.Error())
		// Return JSON error instead of HTML error
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Auth token invalid: " + err.Error(),
		})
		return
	}
	fmt.Println("[INVENTORY] Auth successful for user ID:", claims.UID, "Username:", claims.Username)

	inventoryName := r.URL.Query().Get("inventoryName")
	if inventoryName == "" {
		http.Error(w, "inventoryName missing", http.StatusBadRequest)
		fmt.Println("[INVENTORY] inventoryName missing in request")
		// Return JSON error instead of HTML error
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "inventoryName parameter is missing",
		})
		return
	}
	invID, folderID, err := AddInventory(claims.UID, inventoryName)
	if err != nil {
		http.Error(w, "Failed to create the inventory", http.StatusInternalServerError)
		fmt.Println("[INVENTORY] Failed to add inventory:", err.Error())
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to add inventory: " + err.Error(),
		})
		return
	}

	if strings.HasPrefix(r.UserAgent(), "Resonite") {
		w.Write(
			[]byte(
				fmt.Sprintf("%d\n%d", invID, folderID),
			),
		)
	} else {
		// Return success response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"inventoryId":  invID,
			"rootFolderId": folderID,
		})
	}
}

func RemoveItem(itemId int) error {
	var affectedAssetIds []int
	rows, err := database.Db.Query("SELECT asset_id FROM `hash-usage` WHERE item_id = ?", itemId)
	if err != nil {
		return err
	}
	for rows.Next() {
		var assetId int
		rows.Scan(&assetId)
		affectedAssetIds = append(affectedAssetIds, assetId)
	}
	_, err = database.Db.Exec("DELETE FROM `hash-usage` WHERE item_id = ?", itemId)
	_, err = database.Db.Exec("DELETE FROM Items WHERE id = ?", itemId)
	for _, affectedId := range affectedAssetIds {
		var assetHash string
		err := database.Db.QueryRow("SELECT hash FROM `Assets` WHERE ID = ?", affectedId).Scan(&assetHash)
		if err != nil {
			return err
		}
		var deleteAsset bool
		err = database.Db.QueryRow("SELECT NOT EXISTS(SELECT 1 FROM `hash-usage` WHERE `asset_id` = ?)", affectedId).Scan(&deleteAsset)
		if deleteAsset {
			_, err := database.Db.Exec("DELETE FROM `Assets` WHERE id = ?", affectedId)
			if err != nil {
				return err
			}
			os.Remove(filepath.Join(config.GetConfig().Server.AssetsPath, assetHash))
			os.Remove(filepath.Join(config.GetConfig().Server.AssetsPath, assetHash) + ".brson")
		}
	}
	return nil
}
func RemoveFolder(folderId int) error {
	var affectedFolders []int
	folders, err := database.Db.Query("SELECT id FROM Folders where parent_folder_id = ?", folderId)
	if err != nil {
		return err
	}
	for folders.Next() {
		var id int
		folders.Scan(&id)
		affectedFolders = append(affectedFolders, id)
	}
	for i := 0; i < len(affectedFolders); i++ {
		folders := affectedFolders[i]
		subfolders, err := database.Db.Query("SELECT id, name FROM Folders where parent_folder_id = ?", folders)
		if err != nil {
			return err
		}
		var subfolderIds []int
		for subfolders.Next() {
			var id int
			subfolders.Scan(&id)
			subfolderIds = append(subfolderIds, id)
		}
		for _, subfolder := range subfolderIds {
			affectedFolders = append(affectedFolders, subfolder)
		}
	}
	for _, folder := range affectedFolders {
		var affectedItems []int
		items, err := database.Db.Query("SELECT id FROM Items where folder_id = ?", folder)
		if err != nil {
			return err
		}
		for items.Next() {
			var itemId int
			items.Scan(&itemId)
			affectedItems = append(affectedItems, itemId)
		}
		for _, item := range affectedItems {
			RemoveItem(item)
			if err != nil {
				return err
			}
		}
	}
	for _, folder := range affectedFolders {
		_, err = database.Db.Exec("DELETE FROM Folders WHERE id = ?", folder)
		if err != nil {
			return err
		}
	}
	return nil
}
func handleRemoveFolder(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[FOLDER] RemoveFolder request received:", r.Method, r.URL.String())
	fmt.Println("[FOLDER] Request headers:", r.Header)
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		fmt.Println("[FOLDER] Invalid method:", r.Method)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Invalid request method",
		})
		println(r.Method)
		return
	}
	// Log cookies
	cookies := r.Cookies()
	fmt.Println("[FOLDER] Request cookies:", cookies)

	// Try to get auth token from multiple sources
	var auth string

	// First try cookie (preferred)
	authCookie, err := r.Cookie("auth_token")
	if err == nil {
		auth = authCookie.Value
		fmt.Println("[FOLDER] Found auth_token cookie:", auth[:10]+"...")
	} else {
		// Fallback to query parameter
		auth = r.URL.Query().Get("auth")
		if auth != "" {
			fmt.Println("[FOLDER] Found auth in query param:", auth[:10]+"...")
		}
	}

	if auth == "" {
		// Log debug information
		fmt.Println("[FOLDER] No auth token found in cookie or query param")

		// Return JSON error instead of HTML error
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Auth token missing",
		})
		return
	}

	claims, err := authentication.ParseToken(auth)
	if err != nil {
		http.Error(w, "Auth token missing or invalid", http.StatusUnauthorized)
		fmt.Println("[FOLDER] Auth token invalid:", err.Error())
		// Return JSON error instead of HTML error
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Auth token invalid: " + err.Error(),
		})
		return
	}
	fmt.Println("[FOLDER] Auth successful for user ID:", claims.UID, "Username:", claims.Username)

	folderId, err := strconv.Atoi(r.URL.Query().Get("folderId"))
	if err != nil {
		http.Error(w, "folderId missing or invalid", http.StatusBadRequest)
		fmt.Println("[FOLDER] Invalid folder ID:", err.Error())
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "folderId missing or invalid",
		})
		return
	}
	if allowed, err := query.IsFolderOwner(folderId, claims.UID); err != nil || !allowed {
		http.Error(w, "Forbidden", http.StatusForbidden)
		fmt.Println("[FOLDER] Error finding folder in database:", err.Error())
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Folder not found: " + err.Error(),
		})
		return
	}
	err = RemoveFolder(folderId)
	if err != nil {
		http.Error(w, "Failed to remove folder", http.StatusInternalServerError)
		fmt.Println("[FOLDER] Error removing folder:", err.Error())
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Failed to remove folder: " + err.Error(),
		})
		return
	}
	fmt.Println("[FOLDER] Successfully removed folder ID:", folderId)

}
func handleRemoveItem(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[ITEM] RemoveItem request received:", r.Method, r.URL.String())
	fmt.Println("[ITEM] Request headers:", r.Header)
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		fmt.Println("[ITEM] Invalid method:", r.Method)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid request method",
		})
		println(r.Method)
		return
	}
	// Log cookies
	cookies := r.Cookies()
	fmt.Println("[ITEM] Request cookies:", cookies)

	// Try to get auth token from multiple sources
	var auth string

	// First try cookie (preferred)
	authCookie, err := r.Cookie("auth_token")
	if err == nil {
		auth = authCookie.Value
		fmt.Println("[ITEM] Found auth_token cookie:", auth[:10]+"...")
	} else {
		// Fallback to query parameter
		auth = r.URL.Query().Get("auth")
		if auth != "" {
			fmt.Println("[ITEM] Found auth in query param:", auth[:10]+"...")
		}
	}

	if auth == "" {
		// Log debug information
		fmt.Println("[ITEM] No auth token found in cookie or query param")

		// Return JSON error instead of HTML error
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Auth token missing",
		})
		return
	}

	claims, err := authentication.ParseToken(auth)
	if err != nil {
		http.Error(w, "Auth token missing or invalid", http.StatusUnauthorized)
		fmt.Println("[ITEM] Auth token invalid:", err.Error())
		// Return JSON error instead of HTML error
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Auth token invalid: " + err.Error(),
		})
		return
	}
	fmt.Println("[ITEM] Auth successful for user ID:", claims.UID, "Username:", claims.Username)

	itemId, err := strconv.Atoi(r.URL.Query().Get("itemId"))
	if err != nil {
		http.Error(w, "itemId missing or invalid", http.StatusBadRequest)
		fmt.Println("[ITEM] Invalid item ID:", err.Error())
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "itemId missing or invalid",
		})
		return
	}
	var folderId int
	database.Db.QueryRow("SELECT folder_id FROM Items WHERE id = ?", itemId).Scan(&folderId)
	if allowed, err := query.IsFolderOwner(folderId, claims.UID); err != nil || !allowed {
		http.Error(w, "Forbidden", http.StatusForbidden)
		fmt.Println("[ITEM] Error finding item in database:", err.Error())
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Item not found: " + err.Error(),
		})
		return
	}
	err = RemoveItem(itemId)
	if err != nil {
		http.Error(w, "Failed to remove item", http.StatusInternalServerError)
		fmt.Println("[ITEM] Error removing item:", err.Error())
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to remove item: " + err.Error(),
		})
		return
	}
	fmt.Println("[ITEM] Successfully removed item ID:", itemId)
}
