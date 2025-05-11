package main
import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"resonite-file-provider/assethost"
	"resonite-file-provider/authentication"
	"resonite-file-provider/config"
	"resonite-file-provider/database"
	"resonite-file-provider/query"
	"resonite-file-provider/upload"
)


func main() {
	database.Connect()
	defer database.Db.Close()

	query.AddSearchListeners()
	authentication.AddAuthListeners()
	assethost.AddAssetListeners()
	upload.AddListeners()

	addr := fmt.Sprintf(":%d", config.GetConfig().Server.Port)

	server := &http.Server{
		Addr: addr,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}
	
	if 
	config.GetConfig().Server.Host == "localhost" ||
	config.GetConfig().Server.Host == "0.0.0.0" ||
	config.GetConfig().Server.Host == "127.0.0.1" {
		println("Server running locally, running it in http mode")
		log.Fatal(server.ListenAndServe())
	}else if _, err := os.Stat("./certs"); os.IsNotExist(err){
		println("Certs are missing, running server in http mode. Unless this is in a testing enviroment this is highly not recommended")
		log.Fatal(server.ListenAndServe())
	}else{
		log.Fatal(server.ListenAndServeTLS("certs/fullchain.pem", "certs/privkey.pem"))
	}
}
