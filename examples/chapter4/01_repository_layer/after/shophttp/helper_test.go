package shophttp_test

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

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

// seed 直接寫入一筆訂單。
func seed(db *sql.DB, userID int, item, status string) {
	_, err := db.Exec(`INSERT INTO orders (user_id, item, status) VALUES (?, ?, ?)`, userID, item, status)
	if err != nil {
		panic(err)
	}
}

// call 對 h 送出一個請求，回傳「狀態碼 內容」。
func call(h http.Handler, method, path string) string {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return strings.TrimSpace(fmt.Sprintf("%d %s", rec.Code, strings.TrimSpace(rec.Body.String())))
}
