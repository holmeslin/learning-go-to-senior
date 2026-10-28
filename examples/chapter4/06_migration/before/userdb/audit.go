package userdb

import (
	"context"
	"database/sql"
)

// 稽核紀錄是另一個功能，它的資料表也在自己的檔案裡建立。
func ensureAuditLog(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS audit_log (
	id     INTEGER PRIMARY KEY,
	action TEXT NOT NULL
)`)
	return err
}
