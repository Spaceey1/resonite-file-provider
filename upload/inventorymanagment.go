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
	claims := authentication.AuthCheck(w, r)
	if claims == nil {
		if strings.HasPrefix(r.UserAgent(), "Resonite") {
			http.Error(w, "Failed Auth", http.StatusUnauthorized)
		} else {
			fmt.Println("[FOLDER] Failed Auth")
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "Failed Auth",
			})
		}
		return
	}
	folderId, err := strconv.Atoi(r.URL.Query().Get("folderId"))
	if err != nil {
		if strings.HasPrefix(r.UserAgent(), "Resonite") {
			http.Error(w, "folderId missing or invalid", http.StatusBadRequest)
		} else {
			fmt.Println("[FOLDER] Invalid folder ID:", err.Error())
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "folderId missing or invalid",
			})
		}
		return
	}
	if allowed, err := query.IsFolderOwner(folderId, claims.UID); err != nil || !allowed {
		if strings.HasPrefix(r.UserAgent(), "Resonite") {
			http.Error(w, "Forbidden", http.StatusForbidden)
		} else {
			fmt.Println("[FOLDER] Access denied to folder ID:", folderId, "for user:", claims.Username)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "You don't have permission to create folders here",
			})
		}
		return
	}
	folderName := r.URL.Query().Get("folderName")
	if folderName == "" {
		if strings.HasPrefix(r.UserAgent(), "Resonite") {
			http.Error(w, "folderName missing", http.StatusBadRequest)
		} else {
			fmt.Println("[FOLDER] Missing folder name")
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "folderName parameter is missing",
			})
		}
		return
	}
	fmt.Println("[FOLDER] Creating folder:", folderName, "in parent folder ID:", folderId)
	newFolderId, err := AddFolder(folderId, folderName)
	if err != nil {
		if strings.HasPrefix(r.UserAgent(), "Resonite") {
			http.Error(w, "Error while creating folder", http.StatusInternalServerError)
		} else {
			fmt.Println("[FOLDER] Database error creating folder:", err.Error())
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "Failed to add folder: " + err.Error(),
			})
		}
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
func handleAddInventory(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[INVENTORY] AddInventory request received:", r.Method, r.URL.String())
	fmt.Println("[INVENTORY] Request headers:", r.Header)
	if r.Method != http.MethodPost {
		if strings.HasPrefix(r.UserAgent(), "Resonite") {
			http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		} else {
			fmt.Println("[INVENTORY] Invalid method:", r.Method)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "Invalid request method",
			})
		}
		return
	}
	claims := authentication.AuthCheck(w, r)
	if claims == nil {
		if strings.HasPrefix(r.UserAgent(), "Resonite") {
			http.Error(w, "Failed Auth", http.StatusUnauthorized)
		} else {
			fmt.Println("[INVENTORY] Failed Auth")
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "Failed Auth",
			})
		}
		return
	}

	inventoryName := r.URL.Query().Get("inventoryName")
	if inventoryName == "" {
		if strings.HasPrefix(r.UserAgent(), "Resonite") {
			http.Error(w, "inventoryName missing", http.StatusBadRequest)
		} else {
			fmt.Println("[INVENTORY] inventoryName missing in request")
			// Return JSON error instead of HTML error
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "inventoryName parameter is missing",
			})
		}
		return
	}
	invID, folderID, err := AddInventory(claims.UID, inventoryName)
	if err != nil {
		if strings.HasPrefix(r.UserAgent(), "Resonite") {
			http.Error(w, "Failed to create the inventory", http.StatusInternalServerError)
		} else {
			fmt.Println("[INVENTORY] Failed to add inventory:", err.Error())
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "Failed to add inventory: " + err.Error(),
			})
		}
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
	// Get the user ID for this item to clean up storage usage
	var userId int
	err := database.Db.QueryRow(`
		SELECT ui.user_id 
		FROM Items i
		JOIN Folders f ON i.folder_id = f.id
		JOIN Inventories inv ON f.inventory_id = inv.id
		JOIN users_inventories ui ON inv.id = ui.inventory_id
		WHERE i.id = ?
	`, itemId).Scan(&userId)
	if err != nil {
		return err
	}

	var affectedAssetIds []int
	var affectedAssetHashes []string
	rows, err := database.Db.Query("SELECT asset_id FROM `hash-usage` WHERE item_id = ?", itemId)
	if err != nil {
		return err
	}
	for rows.Next() {
		var assetId int
		rows.Scan(&assetId)
		
		// Get the asset hash for storage cleanup
		var assetHash string
		err := database.Db.QueryRow("SELECT hash FROM `Assets` WHERE id = ?", assetId).Scan(&assetHash)
		if err != nil {
			return err
		}
		
		duplicate, err := database.Db.Query("SELECT item_id FROM `hash-usage` WHERE asset_id = ?", assetId)
		if err != nil {
			return err
		}
		var i int
		for duplicate.Next() {
			i++
		}
		if i <= 1 {
			affectedAssetIds = append(affectedAssetIds, assetId)
			affectedAssetHashes = append(affectedAssetHashes, assetHash)
		}
	}
	
	// Delete the item and its hash usage
	_, err = database.Db.Exec("DELETE FROM `hash-usage` WHERE item_id = ?", itemId)
	_, err = database.Db.Exec("DELETE FROM Items WHERE id = ?", itemId)
	
	// Clean up assets and storage usage
	for i, affectedId := range affectedAssetIds {
		assetHash := affectedAssetHashes[i]
		
		var deleteAsset bool
		err = database.Db.QueryRow("SELECT NOT EXISTS(SELECT 1 FROM `hash-usage` WHERE `asset_id` = ?)", affectedId).Scan(&deleteAsset)
		if deleteAsset {
			// Delete from storage_usage table for this user and asset
			_, err := database.Db.Exec("DELETE FROM `storage_usage` WHERE user_id = ? AND asset_hash = ?", userId, assetHash)
			if err != nil {
				fmt.Printf("Warning: Failed to clean up storage usage for user %d, asset %s: %v\n", userId, assetHash, err)
			}
			
			// Delete the asset
			_, err = database.Db.Exec("DELETE FROM `Assets` WHERE id = ?", affectedId)
			if err != nil {
				return err
			}
			
			// Delete the physical files
			os.Remove(filepath.Join(config.GetConfig().Server.AssetsPath, assetHash))
			os.Remove(filepath.Join(config.GetConfig().Server.AssetsPath, assetHash) + ".brson")
		}
	}
	return nil
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
	claims := authentication.AuthCheck(w, r)
	if claims == nil {
		if strings.HasPrefix(r.UserAgent(), "Resonite") {
			http.Error(w, "Failed Auth", http.StatusUnauthorized)
		} else {
			fmt.Println("[ITEM] Failed Auth")
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "Failed Auth",
			})
		}
		return
	}

	itemId, err := strconv.Atoi(r.URL.Query().Get("itemId"))
	if err != nil {
		if strings.HasPrefix(r.UserAgent(), "Resonite") {
			http.Error(w, "itemId missing or invalid", http.StatusBadRequest)
		} else {
			fmt.Println("[ITEM] Invalid item ID:", err.Error())
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "itemId missing or invalid",
			})
		}
		return
	}
	var folderId int
	database.Db.QueryRow("SELECT folder_id FROM Items WHERE id = ?", itemId).Scan(&folderId)
	if allowed, err := query.IsFolderOwner(folderId, claims.UID); err != nil || !allowed {
		if strings.HasPrefix(r.UserAgent(), "Resonite") {
			http.Error(w, "Forbidden", http.StatusForbidden)
		} else {
			fmt.Println("[ITEM] Error finding item in database:", err.Error())
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "Item not found: " + err.Error(),
			})
		}
		return
	}
	err = RemoveItem(itemId)
	if err != nil {
		if strings.HasPrefix(r.UserAgent(), "Resonite") {
			http.Error(w, "Failed to remove item", http.StatusInternalServerError)
		} else {
			fmt.Println("[ITEM] Error removing item:", err.Error())
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "Failed to remove item: " + err.Error(),
			})
		}
		return
	}
	fmt.Println("[ITEM] Successfully removed item ID:", itemId)
}

func RemoveFolder(folderId int) error {
	folders, err := database.Db.Query("SELECT id FROM Folders where parent_folder_id = ?", folderId)
	if err != nil {
		return err
	}
	var affectedFolders []int
	for folders.Next() {
		var id int
		folders.Scan(&id)
		affectedFolders = append(affectedFolders, id)
	}
	for i := 0; i < len(affectedFolders); {
		currentFolder := affectedFolders[i]
		subfolders, err := database.Db.Query("SELECT id FROM Folders WHERE parent_folder_id = ?", currentFolder)
		if err != nil {
			return err
		}
		for subfolders.Next() {
			var id int
			subfolders.Scan(&id)
			affectedFolders = append(affectedFolders, id)
		}
		i++
	}
	affectedFolders = append(affectedFolders, folderId)
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
			err := RemoveItem(item)
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
	claims := authentication.AuthCheck(w, r)
	if claims == nil {
		if strings.HasPrefix(r.UserAgent(), "Resonite") {
			http.Error(w, "Failed Auth", http.StatusUnauthorized)
		} else {
			fmt.Println("[FOLDER] Failed Auth")
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "Failed Auth",
			})
		}
		return
	}

	folderId, err := strconv.Atoi(r.URL.Query().Get("folderId"))
	if err != nil {
		if strings.HasPrefix(r.UserAgent(), "Resonite") {
			http.Error(w, "folderId missing or invalid", http.StatusBadRequest)
		} else {
			fmt.Println("[FOLDER] Invalid folder ID:", err.Error())
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "folderId missing or invalid",
			})
		}
		return
	}
	if allowed, err := query.IsFolderOwner(folderId, claims.UID); err != nil || !allowed {
		if strings.HasPrefix(r.UserAgent(), "Resonite") {
			http.Error(w, "Forbidden", http.StatusForbidden)
		} else {
			fmt.Println("[FOLDER] Error finding folder in database:", err.Error())
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "Folder not found: " + err.Error(),
			})
		}
		return
	}
	err = RemoveFolder(folderId)
	if err != nil {
		if strings.HasPrefix(r.UserAgent(), "Resonite") {
			http.Error(w, "Failed to remove folder", http.StatusInternalServerError)
		} else {
			fmt.Println("[FOLDER] Error removing folder:", err.Error())
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "Failed to remove folder: " + err.Error(),
			})
		}
		return
	}
	fmt.Println("[FOLDER] Successfully removed folder ID:", folderId)

}

func RemoveInventory(inventoryId int) error {
	folders, err := database.Db.Query("SELECT id FROM Folders WHERE inventory_id = ?", inventoryId)
	if err != nil {
		return err
	}
	var affectedFolders []int
	for folders.Next() {
		var id int
		folders.Scan(&id)
		affectedFolders = append(affectedFolders, id)
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
			err := RemoveItem(item)
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
	_, err = database.Db.Exec("DELETE FROM Inventories WHERE id = ?", inventoryId)
	return nil
}
func handleRemoveInventory(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[INVENTORY] RemoveInventory request received:", r.Method, r.URL.String())
	fmt.Println("[INVENTORY] Request headers:", r.Header)
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		fmt.Println("[INVENTORY] Invalid method:", r.Method)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Invalid request method",
		})
		println(r.Method)
		return
	}
	claims := authentication.AuthCheck(w, r)
	if claims == nil {
		if strings.HasPrefix(r.UserAgent(), "Resonite") {
			http.Error(w, "Failed Auth", http.StatusUnauthorized)
		} else {
			fmt.Println("[INVENTORY] Failed Auth")
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "Failed Auth",
			})
		}
		return
	}

	inventoryId, err := strconv.Atoi(r.URL.Query().Get("inventoryId"))
	if err != nil {
		if strings.HasPrefix(r.UserAgent(), "Resonite") {
			http.Error(w, "inventoryId missing or invalid", http.StatusBadRequest)
		} else {
			fmt.Println("[INVENTORY] Invalid folder ID:", err.Error())
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "invetoryId missing or invalid",
			})
		}
		return
	}
	if allowed, err := query.IsFolderOwner(inventoryId, claims.UID); err != nil || !allowed {
		if strings.HasPrefix(r.UserAgent(), "Resonite") {
			http.Error(w, "Forbidden", http.StatusForbidden)
		} else {
			fmt.Println("[INVENTORY] Error finding inventory in database:", err.Error())
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "Inventory not found: " + err.Error(),
			})
		}
		return
	}
	err = RemoveInventory(inventoryId)
	if err != nil {
		if strings.HasPrefix(r.UserAgent(), "Resonite") {
			http.Error(w, "Failed to remove folder", http.StatusInternalServerError)
		} else {
			fmt.Println("[INVENTORY] Error removing folder:", err.Error())
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "Failed to remove Inventory: " + err.Error(),
			})
		}
		return
	}
	fmt.Println("[INVENTORY] Successfully removed folder ID:", inventoryId)
}

func MakeAssetPublic(itemId int) error {
       _, err := database.Db.Exec("UPDATE `Items` SET `isPublic` = b'1' WHERE `id` = ?;", itemId)
       return err;
}

func MakeAssetPrivate(itemId int) error {
       _, err := database.Db.Exec("UPDATE `Items` SET `isPublic` = b'0' WHERE `id` = ?;", itemId)
       return err;
}
func HandleChangeItemVisibility(w http.ResponseWriter, r *http.Request){
       if r.Method != http.MethodPost {
               http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
               return
       }
       auth := r.URL.Query().Get("auth")
       claims, err := authentication.ParseToken(auth)
       if err != nil {
               http.Error(w, "Auth token missing or invalid", http.StatusUnauthorized)
       }
       itemId, err := strconv.Atoi(r.URL.Query().Get("itemId"))
       if err != nil {
               http.Error(w, "itemId missing or invalid", http.StatusBadRequest)
               return
       }
       visibility, err:= strconv.ParseBool(r.URL.Query().Get("public"))
       if err != nil{
               http.Error(w, "visibility is missing or invalid (Can be 1/0, true/false etc.)", http.StatusBadRequest)
               return
       }
       var folderId int
       database.Db.QueryRow("SELECT folder_id FROM Items WHERE id = ?", itemId).Scan(&folderId)
       if allowed, err := query.IsFolderOwner(folderId, claims.UID); err != nil || !allowed {
               http.Error(w, "Forbidden", http.StatusForbidden)
               return
       }
       if visibility{
               err = MakeAssetPublic(itemId)
       }else{
               err = MakeAssetPrivate(itemId)
       }
       if err != nil {
               http.Error(w, "Internal server error", http.StatusInternalServerError)
	       fmt.Println("[INVENTORY] ", err)
               return
       }
}
