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
		if pingErr := db.Ping(); pingErr == nil {
			Db = db
			if err := runMigrations(Db); err != nil {
				panic(err)
			}
			println("Connected to db")
			return
		} else {
			println(fmt.Sprintf("Couldn't connect: %s\nDb might still be starting, waiting 5 seconds", pingErr.Error()))
			time.Sleep(time.Second * 5)
		}
	}
	panic(fmt.Sprintf("Couldn't connect to database within %d tries, quitting", config.GetConfig().Database.MaxTries))
}

func runMigrations(db *sql.DB) error {
	if err := ensureAssetsColumns(db); err != nil {
		return err
	}
	if err := ensureUsersColumns(db); err != nil {
		return err
	}
	if err := ensureStorageUsageTable(db); err != nil {
		return err
	}
	if err := ensureAdminAuditLogTable(db); err != nil {
		return err
	}
	if err := ensureActiveSessionsTable(db); err != nil {
		return err
	}
	return nil
}

func ensureAssetsColumns(db *sql.DB) error {
	columns := map[string]string{
		"file_size_bytes": "BIGINT DEFAULT 0",
	}
	for column, definition := range columns {
		exists, err := columnExists(db, "Assets", column)
		if err != nil {
			return fmt.Errorf("failed to check Assets.%s column: %w", column, err)
		}
		if exists {
			continue
		}
		alter := fmt.Sprintf("ALTER TABLE `Assets` ADD COLUMN `%s` %s", column, definition)
		if _, err := db.Exec(alter); err != nil {
			return fmt.Errorf("failed to add Assets.%s column: %w", column, err)
		}
		fmt.Printf("Added column %s to Assets table\n", column)
	}
	return nil
}

func ensureUsersColumns(db *sql.DB) error {
	columns := map[string]string{
		"is_admin":   "TINYINT(1) NOT NULL DEFAULT 0",
		"created_at": "DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP",
		"last_login": "DATETIME NULL",
	}
	for column, definition := range columns {
		exists, err := columnExists(db, "Users", column)
		if err != nil {
			return fmt.Errorf("failed to check Users.%s column: %w", column, err)
		}
		if exists {
			continue
		}
		alter := fmt.Sprintf("ALTER TABLE `Users` ADD COLUMN `%s` %s", column, definition)
		if _, err := db.Exec(alter); err != nil {
			return fmt.Errorf("failed to add Users.%s column: %w", column, err)
		}
	}
	return nil
}

func ensureStorageUsageTable(db *sql.DB) error {
	exists, err := tableExists(db, "storage_usage")
	if err != nil {
		return fmt.Errorf("failed to check storage_usage table: %w", err)
	}
	if exists {
		return nil
	}
	createStmt := `CREATE TABLE IF NOT EXISTS storage_usage (
		id INT(11) NOT NULL AUTO_INCREMENT,
		user_id INT(11) NOT NULL,
		asset_hash CHAR(64) NOT NULL,
		file_size_bytes BIGINT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (id),
		KEY user_id (user_id),
		KEY asset_hash (asset_hash)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;`
	if _, err := db.Exec(createStmt); err != nil {
		return fmt.Errorf("failed to create storage_usage table: %w", err)
	}
	return nil
}

func ensureAdminAuditLogTable(db *sql.DB) error {
	exists, err := tableExists(db, "admin_audit_log")
	if err != nil {
		return fmt.Errorf("failed to check admin_audit_log table: %w", err)
	}
	if exists {
		return nil
	}
	createStmt := `CREATE TABLE IF NOT EXISTS admin_audit_log (
		id INT AUTO_INCREMENT PRIMARY KEY,
		admin_user_id INT NOT NULL,
		action VARCHAR(64) NOT NULL,
		target_user_id INT NULL,
		target_resource VARCHAR(255) NULL,
		details TEXT NULL,
		ip_address VARCHAR(64) NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		INDEX idx_admin_user_id (admin_user_id),
		INDEX idx_target_user_id (target_user_id),
		CONSTRAINT fk_admin_audit_log_admin FOREIGN KEY (admin_user_id) REFERENCES Users(id) ON DELETE CASCADE,
		CONSTRAINT fk_admin_audit_log_target FOREIGN KEY (target_user_id) REFERENCES Users(id) ON DELETE SET NULL
	) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;`
	if _, err := db.Exec(createStmt); err != nil {
		return fmt.Errorf("failed to create admin_audit_log table: %w", err)
	}
	return nil
}

func ensureActiveSessionsTable(db *sql.DB) error {
	createStmt := `CREATE TABLE IF NOT EXISTS active_sessions (
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
	) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;`
	if _, err := db.Exec(createStmt); err != nil {
		return fmt.Errorf("failed to ensure active_sessions table: %w", err)
	}
	return nil
}

func tableExists(db *sql.DB, table string) (bool, error) {
	var name string
	err := db.QueryRow(`SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?`, table).Scan(&name)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func columnExists(db *sql.DB, table, column string) (bool, error) {
	var name string
	err := db.QueryRow(`SELECT COLUMN_NAME FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ?`, table, column).Scan(&name)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
