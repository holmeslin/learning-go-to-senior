package loans_test

import (
	"database/sql"
	"errors"
	"fmt"
	"sync/atomic"

	_ "modernc.org/sqlite"
)

// ANCHOR: schema

const schema = `
CREATE TABLE loans (
	id     INTEGER PRIMARY KEY,
	member TEXT NOT NULL,
	title  TEXT NOT NULL
);
`

// ANCHOR_END: schema

// ANCHOR: seed

// seed 是三位會員的五筆借閱；其中一位的名字帶著單引號。
const seed = `
INSERT INTO loans (member, title) VALUES
	('ann', 'sqlite internals'),
	('ann', 'go concurrency'),
	('bob', 'distributed systems'),
	('bob', 'go concurrency'),
	('o''neil', 'compilers');
`

// ANCHOR_END: seed

var dbSeq atomic.Int64

// openDB 開一個全新的資料庫，建好 schema 並放入 seed。
// 共享快取的具名 in-memory DSN 讓連線池多開的連線看到同一份資料，
// 每次換一個名字，測試之間互不干擾。
func openDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite", fmt.Sprintf("file:loans%d?mode=memory&cache=shared", dbSeq.Add(1)))
	if err != nil {
		return nil, err
	}
	for _, stmt := range []string{schema, seed} {
		if _, err := db.Exec(stmt); err != nil {
			return nil, errors.Join(err, db.Close())
		}
	}
	return db, nil
}
