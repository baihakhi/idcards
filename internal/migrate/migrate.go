package migrate

import (
	"idcard/internal/config"
)

func ExecOrFail(db config.DB, query string) error {
	_, err := db.Exec(query)
	return err
}

func CreateTable(db config.DB) error {
	query := `CREATE TABLE IF NOT EXISTS users (
		id VARCHAR(4) PRIMARY KEY NOT NULL,
		nik VARCHAR(16) NOT NULL UNIQUE,
		status CHAR(1) NOT NULL,
		name VARCHAR(255) NOT NULL,
		phone VARCHAR(20) NOT NULL,
		address VARCHAR(255) NOT NULL,
		rating INTEGER DEFAULT 0,
		notes TEXT DEFAULT NULL,
		photo VARCHAR(255) NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	trigger := `CREATE TRIGGER IF NOT EXISTS update_users_updated_at
		AFTER UPDATE ON users
		FOR EACH ROW
		BEGIN
		UPDATE users SET updated_at = CURRENT_TIMESTAMP WHERE id = OLD.id;
		END;`

	idxNik := `CREATE INDEX IF NOT EXISTS idx_users_nik ON users(nik);`

	idxStatus := `CREATE INDEX IF NOT EXISTS idx_users_sopir ON users(status)
		WHERE status = 'S';`

	if err := ExecOrFail(db, query); err != nil {
		return err
	}
	if err := ExecOrFail(db, trigger); err != nil {
		return err
	}
	if err := ExecOrFail(db, idxNik); err != nil {
		return err
	}
	if err := ExecOrFail(db, idxStatus); err != nil {
		return err
	}
	return nil
}
