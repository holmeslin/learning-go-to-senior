package order_test

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

// schema 只供測試建表；資料表怎麼建、怎麼演進，本章〈migration〉會談。
const schema = `CREATE TABLE orders (
	id      INTEGER PRIMARY KEY,
	user_id INTEGER NOT NULL,
	item    TEXT    NOT NULL,
	status  TEXT    NOT NULL
)`

// mustOpenDB 開一個全新的 in-memory 資料庫並建好資料表。name 在同一個測試程式裡要唯一，
// 共享快取的具名資料庫才不會互相干擾。
func mustOpenDB(name string) *sql.DB {
	db, err := sql.Open("sqlite", "file:"+name+"?mode=memory&cache=shared")
	if err != nil {
		panic(err)
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		panic(err)
	}
	return db
}

// newTestDB 是測試用的 mustOpenDB：測試結束時關閉資料庫並檢查錯誤。
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db := mustOpenDB(t.Name())
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close db: %v", err)
		}
	})
	return db
}

// seed 直接寫入一筆訂單。訂單的狀態由訂單流程的其他部分改變，Store 目前只負責查詢。
func seed(db *sql.DB, userID int, item, status string) {
	_, err := db.Exec(`INSERT INTO orders (user_id, item, status) VALUES (?, ?, ?)`, userID, item, status)
	if err != nil {
		panic(err)
	}
}
