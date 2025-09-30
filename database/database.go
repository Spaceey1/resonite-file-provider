package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
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
	if err := ensureInviteSystemTables(db); err != nil {
		return err
	}
	if err := populateExistingAssetSizes(db); err != nil {
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

func ensureInviteSystemTables(db *sql.DB) error {
	// Create system_settings table
	if err := ensureSystemSettingsTable(db); err != nil {
		return err
	}
	
	// Create invite_codes table
	if err := ensureInviteCodesTable(db); err != nil {
		return err
	}
	
	// Create invite_code_usage table
	if err := ensureInviteCodeUsageTable(db); err != nil {
		return err
	}
	
	// Add invite_code_used column to Users table
	if err := ensureUsersInviteColumn(db); err != nil {
		return err
	}
	
	return nil
}

func ensureSystemSettingsTable(db *sql.DB) error {
	exists, err := tableExists(db, "system_settings")
	if err != nil {
		return fmt.Errorf("failed to check system_settings table: %w", err)
	}
	if exists {
		return nil
	}
	
	createStmt := `CREATE TABLE IF NOT EXISTS system_settings (
		id INT(11) NOT NULL AUTO_INCREMENT,
		setting_key VARCHAR(100) NOT NULL UNIQUE,
		setting_value TEXT NOT NULL,
		description TEXT,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		PRIMARY KEY (id)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;`
	
	if _, err := db.Exec(createStmt); err != nil {
		return fmt.Errorf("failed to create system_settings table: %w", err)
	}
	
	// Insert default settings
	defaultSettings := [][]string{
		{"registration_enabled", "true", "Enable/disable public registration"},
		{"require_invite_code", "false", "Require invite code for registration"},
	}
	
	for _, setting := range defaultSettings {
		_, err := db.Exec(`
			INSERT IGNORE INTO system_settings (setting_key, setting_value, description) 
			VALUES (?, ?, ?)
		`, setting[0], setting[1], setting[2])
		if err != nil {
			return fmt.Errorf("failed to insert default setting %s: %w", setting[0], err)
		}
	}
	
	fmt.Println("Created system_settings table with default values")
	return nil
}

func ensureInviteCodesTable(db *sql.DB) error {
	exists, err := tableExists(db, "invite_codes")
	if err != nil {
		return fmt.Errorf("failed to check invite_codes table: %w", err)
	}
	if exists {
		return nil
	}
	
	createStmt := `CREATE TABLE IF NOT EXISTS invite_codes (
		id INT(11) NOT NULL AUTO_INCREMENT,
		code VARCHAR(32) NOT NULL UNIQUE,
		created_by INT(11) NOT NULL,
		max_uses INT(11) DEFAULT 1,
		current_uses INT(11) DEFAULT 0,
		expires_at TIMESTAMP NULL,
		is_active BOOLEAN DEFAULT TRUE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		PRIMARY KEY (id),
		KEY idx_created_by (created_by),
		KEY idx_code (code),
		CONSTRAINT invite_codes_ibfk_1 FOREIGN KEY (created_by) REFERENCES Users (id) ON DELETE CASCADE
	) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;`
	
	if _, err := db.Exec(createStmt); err != nil {
		return fmt.Errorf("failed to create invite_codes table: %w", err)
	}
	
	fmt.Println("Created invite_codes table")
	return nil
}

func ensureInviteCodeUsageTable(db *sql.DB) error {
	exists, err := tableExists(db, "invite_code_usage")
	if err != nil {
		return fmt.Errorf("failed to check invite_code_usage table: %w", err)
	}
	if exists {
		return nil
	}
	
	createStmt := `CREATE TABLE IF NOT EXISTS invite_code_usage (
		id INT(11) NOT NULL AUTO_INCREMENT,
		invite_code_id INT(11) NOT NULL,
		used_by INT(11) NOT NULL,
		used_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (id),
		KEY invite_code_id (invite_code_id),
		KEY used_by (used_by),
		CONSTRAINT invite_code_usage_ibfk_1 FOREIGN KEY (invite_code_id) REFERENCES invite_codes (id) ON DELETE CASCADE,
		CONSTRAINT invite_code_usage_ibfk_2 FOREIGN KEY (used_by) REFERENCES Users (id) ON DELETE CASCADE
	) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;`
	
	if _, err := db.Exec(createStmt); err != nil {
		return fmt.Errorf("failed to create invite_code_usage table: %w", err)
	}
	
	fmt.Println("Created invite_code_usage table")
	return nil
}

func ensureUsersInviteColumn(db *sql.DB) error {
	exists, err := columnExists(db, "Users", "invite_code_used")
	if err != nil {
		return fmt.Errorf("failed to check Users.invite_code_used column: %w", err)
	}
	if exists {
		return nil
	}
	
	// Add the column
	alter := "ALTER TABLE `Users` ADD COLUMN `invite_code_used` INT(11) NULL"
	if _, err := db.Exec(alter); err != nil {
		return fmt.Errorf("failed to add Users.invite_code_used column: %w", err)
	}
	
	// Add the foreign key constraint
	constraint := "ALTER TABLE `Users` ADD CONSTRAINT `users_invite_code_fk` FOREIGN KEY (`invite_code_used`) REFERENCES `invite_codes` (`id`) ON DELETE SET NULL"
	if _, err := db.Exec(constraint); err != nil {
		// If the constraint fails (e.g., because invite_codes table doesn't exist yet), 
		// we'll ignore it as it will be added when the invite_codes table is created
		fmt.Printf("Warning: Could not add foreign key constraint for invite_code_used: %v\n", err)
	}
	
	fmt.Println("Added invite_code_used column to Users table")
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

func populateExistingAssetSizes(db *sql.DB) error {
	fmt.Println("Checking for assets with missing file sizes...")
	
	// Check if we have any assets with file_size_bytes = 0
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM Assets WHERE file_size_bytes = 0 OR file_size_bytes IS NULL").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to count assets with missing sizes: %w", err)
	}
	
	if count == 0 {
		fmt.Println("All assets already have file sizes populated")
		return nil
	}
	
	fmt.Printf("Found %d assets with missing file sizes, calculating from filesystem...\n", count)
	
	// Get all assets that need size calculation
	rows, err := db.Query("SELECT id, hash FROM Assets WHERE file_size_bytes = 0 OR file_size_bytes IS NULL")
	if err != nil {
		return fmt.Errorf("failed to query assets with missing sizes: %w", err)
	}
	defer rows.Close()
	
	updated := 0
	skipped := 0
	
	for rows.Next() {
		var assetId int
		var hash string
		if err := rows.Scan(&assetId, &hash); err != nil {
			fmt.Printf("Error scanning asset row: %v\n", err)
			continue
		}
		
		// Calculate file path based on hash (assuming assets are stored in ./assets/ directory)
		assetPath := filepath.Join("assets", hash)
		
		// Get file size from filesystem
		fileInfo, err := os.Stat(assetPath)
		if err != nil {
			// File might not exist or be in a different location
			fmt.Printf("Warning: Could not stat file for asset %d (hash: %s): %v\n", assetId, hash, err)
			skipped++
			continue
		}
		
		fileSize := fileInfo.Size()
		
		// Update the database with the calculated file size
		_, err = db.Exec("UPDATE Assets SET file_size_bytes = ? WHERE id = ?", fileSize, assetId)
		if err != nil {
			fmt.Printf("Error updating asset %d size: %v\n", assetId, err)
			continue
		}
		
		updated++
		if updated%100 == 0 {
			fmt.Printf("Updated %d assets so far...\n", updated)
		}
	}
	
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating asset rows: %w", err)
	}
	
	fmt.Printf("Asset size population complete: %d updated, %d skipped\n", updated, skipped)
	
	// Also populate storage_usage table for existing assets if it's empty
	if err := populateStorageUsage(db); err != nil {
		return fmt.Errorf("failed to populate storage usage: %w", err)
	}
	
	return nil
}

func populateStorageUsage(db *sql.DB) error {
	// Check if storage_usage table has any data
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM storage_usage").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to count storage usage records: %w", err)
	}
	
	if count > 0 {
		fmt.Println("Storage usage table already has data")
		return nil
	}
	
	fmt.Println("Populating storage usage for existing items...")
	
	// Insert storage usage records for all existing items
	insertStmt := `
		INSERT INTO storage_usage (user_id, asset_hash, file_size_bytes)
		SELECT DISTINCT 
			ui.user_id,
			a.hash,
			a.file_size_bytes
		FROM Items i
		JOIN Folders f ON i.folder_id = f.id
		JOIN Inventories inv ON f.inventory_id = inv.id
		JOIN users_inventories ui ON inv.id = ui.inventory_id
		JOIN ` + "`hash-usage`" + ` hu ON i.id = hu.item_id
		JOIN Assets a ON hu.asset_id = a.id
		WHERE a.file_size_bytes > 0
	`
	
	result, err := db.Exec(insertStmt)
	if err != nil {
		return fmt.Errorf("failed to populate storage usage: %w", err)
	}
	
	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("Populated %d storage usage records\n", rowsAffected)
	
	return nil
}
