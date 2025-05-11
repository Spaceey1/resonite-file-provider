package assethost

import (
	"net/http"
	"resonite-file-provider/authentication"
	"resonite-file-provider/config"
	"resonite-file-provider/database"
	"strings"
)
func isOwnedBy(owner int, url string) bool {
	var exists bool
	url = strings.TrimSuffix(url, ".brson")
	database.Db.QueryRow(`
	SELECT EXISTS (
			SELECT 1 
			FROM Users u
			INNER JOIN users_inventories ui ON u.id = ui.user_id
			INNER JOIN Inventories i ON ui.inventory_id = i.id
			INNER JOIN Folders f ON f.inventory_id = i.id
			INNER JOIN Items it ON it.folder_id = f.id
			WHERE u.id = ? AND it.url = ?
		)
	`, owner, url).Scan(&exists)
	return exists
}
func handleRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = strings.TrimPrefix(r.URL.Path, "/assets/")
		if !strings.HasSuffix(r.URL.Path, ".brson") {
			next.ServeHTTP(w, r)
			return
		}
		authToken := r.URL.Query().Get("auth")
		claims, err := authentication.ParseToken(authToken)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		uId := claims.UID
		if !isOwnedBy(uId, r.URL.Path) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func AddAssetListeners(){
	http.Handle("/assets/", handleRequest(http.FileServer(http.Dir(config.GetConfig().Server.AssetsPath))))
}
