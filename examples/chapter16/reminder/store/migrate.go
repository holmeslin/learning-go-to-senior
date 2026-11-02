package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
)

//go:embed schema/*.sql
var migrations embed.FS

// 版本表的名稱帶 reminder_ 前綴：示範用的資料庫檔和報名服務共用，不能和它的版本表撞名。
const createMigrations = `CREATE TABLE IF NOT EXISTS reminder_migrations (name TEXT PRIMARY KEY)`

// ANCHOR: migrate

// Migrate 依檔名順序套用 schema/ 底下還沒套用過的 SQL 檔，每個檔案和它的版本紀錄在同一個交易裡。
// 它只管提醒服務自己的表；報名服務的表與檢視由報名服務維護，這裡不建立。
//
// shortcut: 沒有加鎖，兩個行程同時啟動可能套用同一個檔案；服務只跑一個實例，
// 改成多副本時要在部署流程讓 migration 只跑一次。
func Migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, createMigrations); err != nil {
		return fmt.Errorf("create reminder_migrations: %w", err)
	}
	files, err := migrations.ReadDir("schema")
	if err != nil {
		return fmt.Errorf("list migrations: %w", err)
	}
	for _, f := range files {
		var applied int
		err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM reminder_migrations WHERE name = ?`, f.Name()).Scan(&applied)
		if err != nil {
			return fmt.Errorf("check migration %s: %w", f.Name(), err)
		}
		if applied > 0 {
			continue
		}
		if err := apply(ctx, db, f.Name()); err != nil {
			return fmt.Errorf("apply migration %s: %w", f.Name(), err)
		}
	}
	return nil
}

// ANCHOR_END: migrate

func apply(ctx context.Context, db *sql.DB, name string) error {
	body, err := migrations.ReadFile("schema/" + name)
	if err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, string(body)); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO reminder_migrations (name) VALUES (?)`, name); err != nil {
		return err
	}
	return tx.Commit()
}
