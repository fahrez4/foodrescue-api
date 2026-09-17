package database

import "log"

// Migrate menjalankan perubahan skema secara idempotent saat startup,
// sehingga deploy image baru otomatis meng-upgrade database tanpa langkah manual.
func Migrate() {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS toko_bank_accounts (
			id CHAR(36) PRIMARY KEY,
			toko_id CHAR(36) NOT NULL,
			bank_name VARCHAR(100) NOT NULL,
			bank_code VARCHAR(20) NOT NULL DEFAULT '',
			account_number VARCHAR(50) NOT NULL,
			account_holder_name VARCHAR(150) NOT NULL,
			is_primary BOOLEAN NOT NULL DEFAULT FALSE,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			INDEX idx_bank_toko (toko_id),
			CONSTRAINT fk_bank_toko FOREIGN KEY (toko_id) REFERENCES toko_profiles(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS toko_api_keys (
			id CHAR(36) PRIMARY KEY,
			toko_id CHAR(36) NOT NULL,
			label VARCHAR(100) NOT NULL DEFAULT '',
			api_key VARCHAR(100) NOT NULL UNIQUE,
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			last_used_at DATETIME NULL,
			created_at DATETIME NOT NULL,
			INDEX idx_apikey_toko (toko_id),
			CONSTRAINT fk_apikey_toko FOREIGN KEY (toko_id) REFERENCES toko_profiles(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	}

	for _, s := range stmts {
		if _, err := DB.Exec(s); err != nil {
			log.Fatalf("Migration failed: %v\nStatement: %s", err, s)
		}
	}

	// Chat komunitas: tambah kolom community_post_id + buat order_id opsional.
	if !columnExists("chats", "community_post_id") {
		if _, err := DB.Exec("ALTER TABLE chats ADD COLUMN community_post_id CHAR(36) NULL AFTER order_id"); err != nil {
			log.Fatalf("Migration failed (chats.community_post_id): %v", err)
		}
	}

	var isNullable string
	_ = DB.QueryRow(`SELECT IS_NULLABLE FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'chats' AND COLUMN_NAME = 'order_id'`).Scan(&isNullable)
	if isNullable == "NO" {
		if _, err := DB.Exec("ALTER TABLE chats MODIFY order_id CHAR(36) NULL"); err != nil {
			log.Fatalf("Migration failed (chats.order_id nullable): %v", err)
		}
	}

	if !indexExists("chats", "uq_chat_community_post") {
		if _, err := DB.Exec("ALTER TABLE chats ADD UNIQUE INDEX uq_chat_community_post (community_post_id)"); err != nil {
			log.Printf("Migration warning (unique community_post index): %v", err)
		}
	}

	log.Println("Database migration completed")
}

func columnExists(table, column string) bool {
	var n int
	_ = DB.QueryRow(`SELECT COUNT(*) FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ?`, table, column).Scan(&n)
	return n > 0
}

func indexExists(table, index string) bool {
	var n int
	_ = DB.QueryRow(`SELECT COUNT(*) FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND INDEX_NAME = ?`, table, index).Scan(&n)
	return n > 0
}
