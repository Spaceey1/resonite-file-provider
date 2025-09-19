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
	"resonite-file-provider/environment"
	"resonite-file-provider/query"
	"resonite-file-provider/upload"
)

func main() {
	database.Connect()
	defer database.Db.Close()

	query.AddSearchListeners()  // For VR UI
	query.AddJSONAPIListeners() // JSON API endpoints for web interface
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

	go upload.StartWebServer()

	if _, err := os.Stat("./certs"); os.IsNotExist(err) ||
		config.GetConfig().Server.Host == "localhost" ||
		config.GetConfig().Server.Host == "0.0.0.0" ||
		config.GetConfig().Server.Host == "127.0.0.1" ||
		environment.GetEnvAsBool("BEHIND_PROXY", false) == true {
		println("HTTP Mode is running. Unless testing or behind a Reverse Proxy this is not recomended!")
		log.Fatal(server.ListenAndServe())
	} else {
		println("HTTPS Mode is running.")
		log.Fatal(server.ListenAndServeTLS("certs/fullchain.pem", "certs/privkey.pem"))
	}
}
