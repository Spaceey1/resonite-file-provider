package assethost

import (
	"fmt"
	"net/http"
	"resonite-file-provider/authentication"
	"resonite-file-provider/database"
	"strings"
)
var assetPathConfig string
func isOwnedBy(owner int, url string) bool {
	var exists bool
	database.Db.QueryRow("SELECT EXISTS (SELECT 1 from Users where id = ? AND id = (SELECT user_id FROM users_inventories WHERE inventory_id = (SELECT inventory_id FROM Folders WHERE id = (SELECT folder_id FROM `Items` WHERE url = ?))))", owner, url).Scan(&exists)
	return exists
}
func handleRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = strings.TrimPrefix(r.URL.Path, assetPathConfig)
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

func AddAssetListeners(assetPath string){
	assetPathConfig = assetPath
	http.Handle(fmt.Sprintf("/%s/", assetPath), handleRequest(http.FileServer(http.Dir(fmt.Sprintf("./%s", assetPath)))))
}
