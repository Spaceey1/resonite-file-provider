package database

import (
	"database/sql"
	"fmt"
	"os"
	"resonite-file-provider/config"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var Db *sql.DB

func ensureActiveSessionsTable(db *sql.DB) {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS active_sessions (
		id INT AUTO_INCREMENT PRIMARY KEY,
		user_id INT NOT NULL,
		token VARCHAR(512) NOT NULL,
		expires_at DATETIME NOT NULL,
		last_seen DATETIME NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE KEY uniq_token (token),
		KEY idx_expires_at (expires_at),
		KEY idx_user_last_seen (user_id, last_seen),
		CONSTRAINT fk_active_sessions_user FOREIGN KEY (user_id) REFERENCES Users(id) ON DELETE CASCADE
	) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;`)
	if err != nil {
		panic(fmt.Sprintf("failed to ensure active_sessions table: %v", err))
	}
}

func Connect() {
	cfg := config.GetConfig().Database
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&loc=UTC",
		os.Getenv("MARIADB_USER"), os.Getenv("MARIADB_PASSWORD"), cfg.Host, cfg.Port, os.Getenv("MARIADB_DATABASE"),
	)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}
	for attempt := 0; attempt < config.GetConfig().Database.MaxTries; attempt++ {
		err := db.Ping()
		if err == nil {
			Db = db
			ensureActiveSessionsTable(Db)
			println("Connected to db")
			return
		}
		println(fmt.Sprintf("Couldn't connect: %s\nDb might still be starting, waiting 5 seconds", err.Error()))
		time.Sleep(time.Second * 5)
	}
	panic(fmt.Sprintf("Couldn't connect to database within %d tries, quitting", config.GetConfig().Database.MaxTries))
}








