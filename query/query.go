package query

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"resonite-file-provider/animxmaker"
	"resonite-file-provider/authentication"
	"resonite-file-provider/database"
	"strconv"
	"strings"
)

func GetChildFolders(folderId int) ([]int, []string, int, error) {
	childFolders, err := database.Db.Query("SELECT id, name FROM Folders where parent_folder_id = ?", folderId);
	if err != nil {
		return nil, nil, -1, err
	}
	var parentFolderId int
	if err := database.Db.QueryRow("SELECT parent_folder_id FROM Folders WHERE id = ?", folderId).Scan(&parentFolderId); err != nil {
		return nil, nil, -1, err
	}
	var childFoldersIds []int
	var childFoldersNames []string
	defer childFolders.Close()

	for childFolders.Next() {
		var id int
		var name string
		if err := childFolders.Scan(&id, &name); err != nil {
			return nil, nil, -1, err
		}
		childFoldersIds = append(childFoldersIds, id)
		childFoldersNames = append(childFoldersNames, name)
	}

	return childFoldersIds, childFoldersNames, parentFolderId, nil
}

func GetChildItems(folderId int) ([]int, []string, []string, error) {
	items, err := database.Db.Query("SELECT id, name, url FROM Items where folder_id = ?", folderId);
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

func GetSearchResults(query string, inventoryId int) ([]int, []string, []string, error){
	items, err := database.Db.Query(
		`select Items.id, Items.name, Items.url
		from Items
		inner join Folders on Items.folder_id = Folders.id
		where Folders.inventory_id = ? AND INSTR(Items.name, ?)`)
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
	rows, err := database.Db.Query("SELECT id from Users WHERE id = (SELECT user_id from users_inventories where inventory_id = (SELECT inventory_id FROM Folders WHERE id = ?))", folderId)
	if err != nil {
		return false, err
	}
	for rows.Next(){
		var currectUserId int
		if err := rows.Scan(&currectUserId); err != nil{
			return false, err
		}
		if currectUserId == userId {
			return true, nil
		}
	}
	return false, nil
}

func IsInventoryOwner(inventoryId int, userId int) (bool, error){
	// TODO
	return true, nil
}

func listFolders(w http.ResponseWriter, r *http.Request) {
	folderId, err := strconv.Atoi(r.URL.Query().Get("folderId"))
	if err != nil {
		http.Error(w, "folderId is either not specified or is invalid", http.StatusBadRequest)
		return
	}
	authKey := r.URL.Query().Get("auth")
	claims, err := authentication.ParseToken(authKey)
	if err != nil {
		http.Error(w, "Auth token invalid or missing", http.StatusUnauthorized)
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
	}else{
		var items []map[string]any
		for i := 0; i < len(names) && i < len(ids); i++{
			items = append(items, map[string]any{
				"name": names[i],
				"id": ids[i],
			})
		}
		data := map[string]any{
			"results": items,
			"parentId": parentID,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
	}
}

func listItems(w http.ResponseWriter, r *http.Request) {
	folderId, err := strconv.Atoi(r.URL.Query().Get("folderId"))
	if err != nil {
		http.Error(w, "folderId is either not specified or is invalid", http.StatusBadRequest)
	}
	authKey := r.URL.Query().Get("auth")
	claims, err := authentication.ParseToken(authKey)
	if err != nil {
		http.Error(w, "Auth token invalid or missing", http.StatusUnauthorized)
		return
	}
	if allowed, err := IsFolderOwner(folderId, claims.UID); !allowed || err != nil {
		http.Error(w, "You don't have access to this folder", http.StatusForbidden)
		return
	}
	ids, names, urls, err := GetChildItems(folderId)
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
	}else{
		var items []map[string]any
		for i := 0; i < len(names) && i < len(ids); i++{
			items = append(items, map[string]any{
				"name": names[i],
				"id": ids[i],
				"url": urls[i],
			})
		}
		data := map[string]any{
			"results": items,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
	}
}

func listInventories(w http.ResponseWriter, r *http.Request){
	auth := r.URL.Query().Get("auth")
	claims, err := authentication.ParseToken(auth)
	if err != nil {
		http.Error(w, "Auth token invalid or missing", http.StatusUnauthorized)
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
	}else{
		var items []map[string]any
		for i := 0; i < len(inventoryNames) && i < len(inventoryIds); i++{
			items = append(items, map[string]any{
				"name": inventoryNames[i],
				"id": inventoryIds[i],
			})
		}
		data := map[string]any{
			"results": items,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
	}
}

func listFolderContents(w http.ResponseWriter, r *http.Request) {
	folderId, err := strconv.Atoi(r.URL.Query().Get("folderId"))
	if err != nil {
		http.Error(w, "folderId is either not specified or is invalid", http.StatusBadRequest)
	}
	authKey := r.URL.Query().Get("auth")
	claims, err := authentication.ParseToken(authKey)
	if err != nil {
		http.Error(w, "Auth token invalid or missing", http.StatusUnauthorized)
		return
	}
	if allowed, err := IsFolderOwner(folderId, claims.UID); !allowed || err != nil {
		http.Error(w, "You don't have access to this folder", http.StatusForbidden)
		return
	}
	itemIdsTrack, itemNamesTrack, itemUrlsTrack, err := GetChildItems(folderId)
	if err != nil {
		http.Error(w, "Error while getting items", http.StatusInternalServerError)
		return
	}
	folderIdsTrack, folderNamesTrack, parentFolder, err := GetChildFolders(folderId)
	if err != nil {
		http.Error(w, "Error while getting folders", http.StatusInternalServerError)
		return
	}
	if strings.HasPrefix(r.UserAgent(), "Resonite"){
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
	}else{
		var items []map[string]any
		var folders []map[string]any
		for i := 0; i < len(itemIdsTrack); i++{
			items = append(items, map[string]any{
				"id": itemIdsTrack[i],
				"name": itemNamesTrack[i],
				"url": itemUrlsTrack[i],
			})
		}
		for i := 0; i < len(folderIdsTrack); i++{
			folders = append(folders, map[string]any{
				"id": folderIdsTrack[i],
				"name": folderNamesTrack[i],
			})
		}
		data := map[string]any{
			"items": items,
			"folders": folders,
			"parent": parentFolder,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
	}
}



func getInventoryRootFolder(w http.ResponseWriter, r *http.Request){
	inventoryId, err := strconv.Atoi(r.URL.Query().Get("inventoryId"))
	if err != nil {
		http.Error(w, "inventoryId is either not specified or is invalid", http.StatusBadRequest)
	}
	authKey := r.URL.Query().Get("auth")
	claims, err := authentication.ParseToken(authKey)
	if err != nil {
		http.Error(w, "Auth token invalid or missing", http.StatusUnauthorized)
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

func searchInventory(w http.ResponseWriter, r *http.Request){
	inventoryId, err := strconv.Atoi(r.URL.Query().Get("inventoryId"))
	if err != nil {
		http.Error(w, "inventoryId is either not specified or is invalid", http.StatusBadRequest)
	}
	query := strings.TrimSpace(r.URL.Query().Get("query"))
	if query == "" {
		http.Error(w, "query is either not specified or is invalid", http.StatusBadRequest)
	}
	authKey := r.URL.Query().Get("auth")
	claims, err := authentication.ParseToken(authKey)
	if err != nil {
		http.Error(w, "Auth token invalid or missing", http.StatusUnauthorized)
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
	}else{
		var items []map[string]any
		for i := 0; i < len(itemIds); i++{
			items = append(items, map[string]any{
				"name": itemNames[i],
				"id": itemIds[i],
				"url": itemUrls[i],
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
