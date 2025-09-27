package query

import (
	"encoding/json"
	"database/sql"
	"log"
	"net/http"
	"path/filepath"
	"resonite-file-provider/animxmaker"
	"resonite-file-provider/authentication"
	"resonite-file-provider/database"
	"strconv"
	"strings"
)

func GetChildFolders(folderId int) ([]int, []string, int, error) {
	log.Printf("[GetChildFolders] Starting query for folderId: %d", folderId)
	
	childFolders, err := database.Db.Query("SELECT id, name FROM Folders where parent_folder_id = ?", folderId)
	if err != nil {
		log.Printf("[GetChildFolders] ERROR: Failed to query child folders for folderId: %d, error: %v", folderId, err)
		return nil, nil, -1, err
	}
	defer childFolders.Close()
	
	var parentFolderId int
	if err := database.Db.QueryRow("SELECT parent_folder_id FROM Folders WHERE id = ?", folderId).Scan(&parentFolderId); err != nil {
		log.Printf("[GetChildFolders] ERROR: Failed to get parent folder ID for folderId: %d, error: %v", folderId, err)
		return nil, nil, -1, err
	}
	log.Printf("[GetChildFolders] Found parent folder ID: %d for folderId: %d", parentFolderId, folderId)
	
	var childFoldersIds []int
	var childFoldersNames []string

	for childFolders.Next() {
		var id int
		var name string
		if err := childFolders.Scan(&id, &name); err != nil {
			log.Printf("[GetChildFolders] ERROR: Failed to scan child folder row for folderId: %d, error: %v", folderId, err)
			return nil, nil, -1, err
		}
		childFoldersIds = append(childFoldersIds, id)
		childFoldersNames = append(childFoldersNames, name)
		log.Printf("[GetChildFolders] Found child folder: %s (ID: %d) for parent folderId: %d", name, id, folderId)
	}

	log.Printf("[GetChildFolders] Successfully retrieved %d child folders for folderId: %d", len(childFoldersIds), folderId)
	return childFoldersIds, childFoldersNames, parentFolderId, nil
}

func GetChildItems(folderId int) ([]int, []string, []string, []int64, error) {
	log.Printf("[GetChildItems] Starting query for folderId: %d", folderId)
	
	// First, try the new schema with file_size_bytes
	items, err := database.Db.Query(`
		SELECT i.id, i.name, i.url, COALESCE(SUM(DISTINCT a.file_size_bytes), 0) as total_size
		FROM Items i
		LEFT JOIN `+"`hash-usage`"+` hu ON i.id = hu.item_id
		LEFT JOIN Assets a ON hu.asset_id = a.id
		WHERE i.folder_id = ?
		GROUP BY i.id, i.name, i.url
	`, folderId)
	
	// If the new schema query fails, fall back to the old schema
	if err != nil {
		log.Printf("[GetChildItems] New schema query failed for folderId: %d, trying fallback query. Error: %v", folderId, err)
		
		// Fallback query for older schema without file_size_bytes
		items, err = database.Db.Query(`
			SELECT i.id, i.name, i.url
			FROM Items i
			WHERE i.folder_id = ?
		`, folderId)
		
		if err != nil {
			log.Printf("[GetChildItems] ERROR: Both queries failed for folderId: %d, error: %v", folderId, err)
			return nil, nil, nil, nil, err
		}
		
		log.Printf("[GetChildItems] Using fallback query (old schema) for folderId: %d", folderId)
		defer items.Close()

		var itemsIds []int
		var itemsNames []string
		var itemsUrls []string
		var itemsSizes []int64

		for items.Next() {
			var id int
			var name string
			var url string
			if err := items.Scan(&id, &name, &url); err != nil {
				log.Printf("[GetChildItems] ERROR: Failed to scan child item row (fallback) for folderId: %d, error: %v", folderId, err)
				return nil, nil, nil, nil, err
			}
			itemsIds = append(itemsIds, id)
			itemsNames = append(itemsNames, name)
			itemsUrls = append(itemsUrls, filepath.Join("assets", url))
			itemsSizes = append(itemsSizes, 0) // Default size to 0 for old schema
			log.Printf("[GetChildItems] Found child item (fallback): %s (ID: %d, Size: unknown) for folderId: %d", name, id, folderId)
		}

		log.Printf("[GetChildItems] Successfully retrieved %d child items (fallback) for folderId: %d", len(itemsIds), folderId)
		return itemsIds, itemsNames, itemsUrls, itemsSizes, nil
	}
	
	// New schema query succeeded
	log.Printf("[GetChildItems] Using new schema query for folderId: %d", folderId)
	defer items.Close()

	var itemsIds []int
	var itemsNames []string
	var itemsUrls []string
	var itemsSizes []int64

	for items.Next() {
		var id int
		var name string
		var url string
		var size int64
		if err := items.Scan(&id, &name, &url, &size); err != nil {
			log.Printf("[GetChildItems] ERROR: Failed to scan child item row for folderId: %d, error: %v", folderId, err)
			return nil, nil, nil, nil, err
		}
		itemsIds = append(itemsIds, id)
		itemsNames = append(itemsNames, name)
		itemsUrls = append(itemsUrls, filepath.Join("assets", url))
		itemsSizes = append(itemsSizes, size)
		log.Printf("[GetChildItems] Found child item: %s (ID: %d, Size: %d bytes) for folderId: %d", name, id, size, folderId)
	}

	log.Printf("[GetChildItems] Successfully retrieved %d child items for folderId: %d", len(itemsIds), folderId)
	return itemsIds, itemsNames, itemsUrls, itemsSizes, nil
}

func GetSearchResults(query string, inventoryId int) ([]int, []string, []string, error) {
	items, err := database.Db.Query(
		`select Items.id, Items.name, Items.url
		from Items
		inner join Folders on Items.folder_id = Folders.id
		where Folders.inventory_id = ? AND INSTR(Items.name, ?)`, inventoryId, query)
	if err != nil {
		return nil, nil, nil, err
	}
	var itemsIds []int
	var itemsNames []string
	var itemsUrls []string
	defer items.Close()

	for items.Next() {
		var id int
		var name string
		var url string
		if err := items.Scan(&id, &name, &url); err != nil {
			return nil, nil, nil, err
		}
		itemsIds = append(itemsIds, id)
		itemsNames = append(itemsNames, name)
		itemsUrls = append(itemsUrls, filepath.Join("assets", url))
	}

	return itemsIds, itemsNames, itemsUrls, nil
}

func IsFolderOwner(folderId int, userId int) (bool, error) {
	log.Printf("[IsFolderOwner] Checking ownership for folderId: %d, userId: %d", folderId, userId)
	
	rows, err := database.Db.Query("SELECT id from Users WHERE id = (SELECT user_id from users_inventories where inventory_id = (SELECT inventory_id FROM Folders WHERE id = ?))", folderId)
	if err != nil {
		log.Printf("[IsFolderOwner] ERROR: Failed to query folder ownership for folderId: %d, userId: %d, error: %v", folderId, userId, err)
		return false, err
	}
	defer rows.Close()
	
	for rows.Next() {
		var currectUserId int
		if err := rows.Scan(&currectUserId); err != nil {
			log.Printf("[IsFolderOwner] ERROR: Failed to scan user ID for folderId: %d, userId: %d, error: %v", folderId, userId, err)
			return false, err
		}
		log.Printf("[IsFolderOwner] Found authorized user: %d for folderId: %d", currectUserId, folderId)
		if currectUserId == userId {
			log.Printf("[IsFolderOwner] Access granted: user %d owns folderId: %d", userId, folderId)
			return true, nil
		}
	}
	
	log.Printf("[IsFolderOwner] Access denied: user %d does not own folderId: %d", userId, folderId)
	return false, nil
}

func IsInventoryOwner(inventoryId int, userId int) (bool, error) {
	// TODO
	return true, nil
}

// handles GET /query/childfolders
func listFolders(w http.ResponseWriter, r *http.Request) {
	folderId, err := strconv.Atoi(r.URL.Query().Get("folderId"))
	if err != nil {
		http.Error(w, "folderId is either not specified or is invalid", http.StatusBadRequest)
		return
	}
	claims := authentication.AuthCheck(w, r)
	if claims == nil {
		http.Error(w, "[FolderContents] Failed Auth", http.StatusUnauthorized)
		return
	}
	if allowed, err := IsFolderOwner(folderId, claims.UID); !allowed || err != nil {
		http.Error(w, "You don't have access to this folder", http.StatusForbidden)
		return
	}
	ids, names, parentID, err := GetChildFolders(folderId)
	if strings.HasPrefix(r.UserAgent(), "Resonite") {
		animation := animxmaker.Animation{
			Tracks: []animxmaker.AnimationTrackWrapper{
				animxmaker.ListTrack(ids, "results", "id"),
				animxmaker.ListTrack(names, "results", "name"),
				animxmaker.ListTrack([]int{parentID}, "results", "parent"),
			},
		}
		encodedAnimaiton, err := animation.EncodeAnimation("response")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(encodedAnimaiton)
	} else {
		var items []map[string]any
		for i := 0; i < len(names) && i < len(ids); i++ {
			items = append(items, map[string]any{
				"name": names[i],
				"id":   ids[i],
			})
		}
					// Get parent folder info
		var parentInfo *ParentFolderInfo
		var parentID sql.NullInt64
		var parentName sql.NullString
	
		err = database.Db.QueryRow(`
			SELECT parent_folder_id, 
		       (SELECT name FROM Folders WHERE id = f.parent_folder_id) as parent_name
			FROM Folders f 
			WHERE id = ?
		`, folderId).Scan(&parentID, &parentName)
	
		if err == nil && parentID.Valid && parentName.Valid {
			parentInfo = &ParentFolderInfo{
				ID:   int(parentID.Int64),
				Name: parentName.String,
			}
		}
		data := map[string]any{
			"results":  items,
			"parentId": parentInfo,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
	}
}

// handles /query/childItems
func listItems(w http.ResponseWriter, r *http.Request) {
	folderId, err := strconv.Atoi(r.URL.Query().Get("folderId"))
	if err != nil {
		http.Error(w, "folderId is either not specified or is invalid", http.StatusBadRequest)
	}
	claims := authentication.AuthCheck(w, r)
	if claims == nil {
		http.Error(w, "[ChildItems] Failed Auth", http.StatusUnauthorized)
		return
	}
	if allowed, err := IsFolderOwner(folderId, claims.UID); !allowed || err != nil {
		http.Error(w, "You don't have access to this folder", http.StatusForbidden)
		return
	}
	ids, names, urls, sizes, err := GetChildItems(folderId)
	if strings.HasPrefix(r.UserAgent(), "Resonite") {
		animation := animxmaker.Animation{
			Tracks: []animxmaker.AnimationTrackWrapper{
				animxmaker.ListTrack(ids, "results", "id"),
				animxmaker.ListTrack(names, "results", "name"),
				animxmaker.ListTrack(urls, "results", "url"),
			},
		}
		encodedAnimaiton, err := animation.EncodeAnimation("response")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(encodedAnimaiton)
		return
	} else {
		var items []map[string]any
		for i := 0; i < len(names) && i < len(ids); i++ {
			items = append(items, map[string]any{
				"name": names[i],
				"id":   ids[i],
				"url":  urls[i],
				"size": sizes[i],
			})
		}
		data := map[string]any{
			"results": items,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
	}
}

// handles /query/inventories
func listInventories(w http.ResponseWriter, r *http.Request) {
	claims := authentication.AuthCheck(w, r)
	if claims == nil {
		http.Error(w, "[Inventories] Failed Auth", http.StatusUnauthorized)
		return
	}
	result, err := database.Db.Query("SELECT name, id FROM `Inventories` WHERE id in (SELECT id FROM users_inventories WHERE user_id = ?)", claims.UID)
	if err != nil {
		http.Error(w, "Failed to query the database", http.StatusInternalServerError)
	}
	var inventoryIds []int
	var inventoryNames []string
	for result.Next() {
		var name string
		var id int
		result.Scan(&name, &id)
		inventoryIds = append(inventoryIds, id)
		inventoryNames = append(inventoryNames, name)
	}
	if strings.HasPrefix(r.UserAgent(), "Resonite") {
		response := animxmaker.Animation{
			Tracks: []animxmaker.AnimationTrackWrapper{
				animxmaker.ListTrack(inventoryIds, "results", "id"),
				animxmaker.ListTrack(inventoryNames, "results", "name"),
			},
		}
		encodedResponse, err := response.EncodeAnimation("response")
		if err != nil {
			http.Error(w, "Error while encoding animx", http.StatusInternalServerError)
		}
		w.Write(encodedResponse)
		w.WriteHeader(http.StatusOK)
	} else {
		var items []map[string]any
		for i := 0; i < len(inventoryNames) && i < len(inventoryIds); i++ {
			items = append(items, map[string]any{
				"name": inventoryNames[i],
				"id":   inventoryIds[i],
			})
		}
		data := map[string]any{
			"results": items,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
	}
}

// handles /query/folderContent
func listFolderContents(w http.ResponseWriter, r *http.Request) {
	log.Printf("[FolderContents] Starting request for URL: %s", r.URL.String())
	
	folderId, err := strconv.Atoi(r.URL.Query().Get("folderId"))
	if err != nil {
		log.Printf("[FolderContents] ERROR: Invalid folderId parameter: %v", err)
		http.Error(w, "folderId is either not specified or is invalid", http.StatusBadRequest)
		return
	}
	log.Printf("[FolderContents] Processing request for folderId: %d", folderId)
	
	claims := authentication.AuthCheck(w, r)
	if claims == nil {
		log.Printf("[FolderContents] ERROR: Authentication failed for folderId: %d", folderId)
		http.Error(w, "[FolderContents] Failed Auth", http.StatusUnauthorized)
		return
	}
	log.Printf("[FolderContents] Authentication successful for user: %d, folderId: %d", claims.UID, folderId)
	
	allowed, err := IsFolderOwner(folderId, claims.UID)
	if err != nil {
		log.Printf("[FolderContents] ERROR: Failed to check folder ownership for folderId: %d, userId: %d, error: %v", folderId, claims.UID, err)
		http.Error(w, "Error checking folder access: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if !allowed {
		log.Printf("[FolderContents] ERROR: Access denied for user: %d, folderId: %d", claims.UID, folderId)
		http.Error(w, "You don't have access to this folder", http.StatusForbidden)
		return
	}
	log.Printf("[FolderContents] Access granted for user: %d, folderId: %d", claims.UID, folderId)
	
	log.Printf("[FolderContents] Fetching child items for folderId: %d", folderId)
	itemIdsTrack, itemNamesTrack, itemUrlsTrack, itemSizesTrack, err := GetChildItems(folderId)
	if err != nil {
		log.Printf("[FolderContents] ERROR: Failed to get child items for folderId: %d, error: %v", folderId, err)
		http.Error(w, "Error while getting items: "+err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("[FolderContents] Successfully fetched %d child items for folderId: %d", len(itemIdsTrack), folderId)
	
	log.Printf("[FolderContents] Fetching child folders for folderId: %d", folderId)
	folderIdsTrack, folderNamesTrack, parentFolder, err := GetChildFolders(folderId)
	if err != nil {
		log.Printf("[FolderContents] ERROR: Failed to get child folders for folderId: %d, error: %v", folderId, err)
		http.Error(w, "Error while getting folders: "+err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("[FolderContents] Successfully fetched %d child folders for folderId: %d, parentFolder: %d", len(folderIdsTrack), folderId, parentFolder)
	if strings.HasPrefix(r.UserAgent(), "Resonite") {
		response := animxmaker.Animation{
			Tracks: []animxmaker.AnimationTrackWrapper{
				animxmaker.ListTrack(itemIdsTrack, "items", "id"),
				animxmaker.ListTrack(itemNamesTrack, "items", "name"),
				animxmaker.ListTrack(itemUrlsTrack, "items", "url"),
				animxmaker.ListTrack(folderIdsTrack, "folders", "id"),
				animxmaker.ListTrack(folderNamesTrack, "folders", "name"),
				animxmaker.ListTrack([]int{parentFolder}, "folders", "parentFolder"),
			},
		}
		encodedResponse, err := response.EncodeAnimation("response")
		if err != nil {
			http.Error(w, "Error while encoding animx", http.StatusInternalServerError)
		}
		w.Write(encodedResponse)
	} else {
		var items []map[string]any
		var folders []map[string]any
		for i := 0; i < len(itemIdsTrack); i++ {
			items = append(items, map[string]any{
				"id":   itemIdsTrack[i],
				"name": itemNamesTrack[i],
				"url":  itemUrlsTrack[i],
				"size": itemSizesTrack[i],
			})
		}
		for i := 0; i < len(folderIdsTrack); i++ {
			folders = append(folders, map[string]any{
				"id":   folderIdsTrack[i],
				"name": folderNamesTrack[i],
			})
		}
		// Get parent folder info
		log.Printf("[FolderContents] Fetching parent folder info for folderId: %d", folderId)
		var parentInfo *ParentFolderInfo
		var parentID sql.NullInt64
		var parentName sql.NullString
	
		err = database.Db.QueryRow(`
			SELECT parent_folder_id, 
		       (SELECT name FROM Folders WHERE id = f.parent_folder_id) as parent_name
			FROM Folders f 
			WHERE id = ?
		`, folderId).Scan(&parentID, &parentName)
	
		if err != nil {
			log.Printf("[FolderContents] WARNING: Failed to get parent folder info for folderId: %d, error: %v", folderId, err)
		} else if parentID.Valid && parentName.Valid {
			parentInfo = &ParentFolderInfo{
				ID:   int(parentID.Int64),
				Name: parentName.String,
			}
			log.Printf("[FolderContents] Found parent folder: %s (ID: %d) for folderId: %d", parentName.String, parentID.Int64, folderId)
		} else {
			log.Printf("[FolderContents] No parent folder found for folderId: %d (likely root folder)", folderId)
		}
		
		data := map[string]any{
			"items":   items,
			"folders": folders,
			"parent":  parentInfo,
		}
		
		log.Printf("[FolderContents] Successfully completed request for folderId: %d, returning %d items and %d folders", folderId, len(items), len(folders))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
	}
}

// handles /query/inventoryRootFolder
func getInventoryRootFolder(w http.ResponseWriter, r *http.Request) {
	inventoryId, err := strconv.Atoi(r.URL.Query().Get("inventoryId"))
	if err != nil {
		http.Error(w, "inventoryId is either not specified or is invalid", http.StatusBadRequest)
	}
	claims := authentication.AuthCheck(w, r)
	if claims == nil {
		http.Error(w, "[RootFolder] Failed Auth", http.StatusUnauthorized)
		return
	}
	if allowed, err := IsInventoryOwner(inventoryId, claims.UID); !allowed || err != nil {
		http.Error(w, "You don't have access to this inventory", http.StatusForbidden)
		return
	}
	var rootFolderId int
	err = database.Db.QueryRow("SELECT id FROM Folders WHERE `inventory_id` = ? AND parent_folder_id = -1", inventoryId).Scan(&rootFolderId)
	if err != nil {
		http.Error(w, "Error while getting folder id", http.StatusInternalServerError)
		return
	}
	w.Write([]byte(strconv.Itoa(rootFolderId)))
}

// handles /query/search
func searchInventory(w http.ResponseWriter, r *http.Request) {
	inventoryId, err := strconv.Atoi(r.URL.Query().Get("inventoryId"))
	if err != nil {
		http.Error(w, "inventoryId is either not specified or is invalid", http.StatusBadRequest)
	}
	query := strings.TrimSpace(r.URL.Query().Get("query"))
	if query == "" {
		http.Error(w, "query is either not specified or is invalid", http.StatusBadRequest)
	}
	claims := authentication.AuthCheck(w, r)
	if claims == nil {
		http.Error(w, "[SearchInventory] Failed Auth", http.StatusUnauthorized)
		return
	}
	if allowed, err := IsInventoryOwner(inventoryId, claims.UID); !allowed || err != nil {
		http.Error(w, "You don't have access to this inventory", http.StatusForbidden)
		return
	}
	itemIds, itemNames, itemUrls, err := GetSearchResults(query, inventoryId)
	if strings.HasPrefix(r.UserAgent(), "Resonite") {
		response := animxmaker.Animation{
			Tracks: []animxmaker.AnimationTrackWrapper{
				animxmaker.ListTrack(itemIds, "items", "id"),
				animxmaker.ListTrack(itemNames, "items", "name"),
				animxmaker.ListTrack(itemUrls, "items", "url"),
			},
		}
		encodedResponse, err := response.EncodeAnimation("response")
		if err != nil {
			http.Error(w, "Error while encoding animx", http.StatusInternalServerError)
		}
		w.Write(encodedResponse)
		w.WriteHeader(http.StatusOK)
	} else {
		var items []map[string]any
		for i := 0; i < len(itemIds); i++ {
			items = append(items, map[string]any{
				"name": itemNames[i],
				"id":   itemIds[i],
				"url":  itemUrls[i],
			})
		}
		data := map[string]any{
			"items": items,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
	}
}
func AddSearchListeners() {
	http.HandleFunc("/query/childFolders", listFolders)
	http.HandleFunc("/query/childItems", listItems)
	http.HandleFunc("/query/folderContent", listFolderContents)
	http.HandleFunc("/query/inventories", listInventories)
	http.HandleFunc("/query/inventoryRootFolder", getInventoryRootFolder)
	http.HandleFunc("/query/search", searchInventory)
}
