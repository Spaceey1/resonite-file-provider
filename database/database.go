package database

import (
	"database/sql"
	"fmt"
	"resonite-file-provider/config"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var Db *sql.DB
func Connect() {
	cfg := config.GetConfig().Database
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
	    cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name,
	)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}
	for attempt := 0; attempt < config.GetConfig().Database.MaxTries; attempt++{
		err := db.Ping()
		if err == nil {
			Db = db
			println("Connected to db")
			return
		}
		println(fmt.Sprintf("Couldn't connect: %s\nDb might still be starting, waiting 5 seconds", err.Error()))
		time.Sleep(time.Second * 5)
	}
	panic(fmt.Sprintf("Couldn't connect to database within %d tries, quitting", config.GetConfig().Database.MaxTries))
}
