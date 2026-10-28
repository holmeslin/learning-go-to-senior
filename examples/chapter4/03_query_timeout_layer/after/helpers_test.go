package orders_test

import (
	"database/sql"
	"fmt"
	"sync/atomic"
	"testing"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE orders (id INTEGER PRIMARY KEY, status TEXT NOT NULL, note TEXT NOT NULL);
INSERT INTO orders (status, note) VALUES
	('pending', 'gift wrap'), ('pending', 'rush'), ('pending', ''), ('done', 'gift wrap');`

var dbSeq atomic.Int64

// openDB 開一個獨立的 in-memory 資料庫，放進 3 筆待處理與 1 筆已完成的訂單。
// 每次呼叫的名稱都不同，測試之間不會互相干擾；共享快取讓同一個名稱的每條連線看到同一份資料。
// 連線池只留一條連線，hold 才能確定性地讓資料庫「忙碌」。
func openDB() *sql.DB {
	db, err := sql.Open("sqlite", fmt.Sprintf("file:orders%d?mode=memory&cache=shared", dbSeq.Add(1)))
	if err != nil {
		panic(err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schema); err != nil {
		panic(err)
	}
	return db
}

// hold 開一個不結束的交易，占住唯一的連線：之後的查詢只能排隊，直到交易結束或自己的 context 結束。
func hold(db *sql.DB) *sql.Tx {
	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}
	return tx
}

// newDB 是測試用的 openDB：測試結束時關閉資料庫並檢查錯誤。
func newDB(t *testing.T) *sql.DB {
	t.Helper()
	db := openDB()
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	})
	return db
}

// busy 讓 db 忙碌到測試結束。它在 newDB 之後註冊 cleanup，所以先放開連線，再關資料庫。
func busy(t *testing.T, db *sql.DB) {
	t.Helper()
	tx := hold(db)
	t.Cleanup(func() {
		if err := tx.Rollback(); err != nil {
			t.Error(err)
		}
	})
}
