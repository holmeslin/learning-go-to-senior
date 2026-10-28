package orders_test

import (
	"database/sql"
	"errors"
	"fmt"
	"sync/atomic"

	_ "modernc.org/sqlite"
)

// ANCHOR: schema

const schema = `
CREATE TABLE orders (
	id       INTEGER PRIMARY KEY,
	customer TEXT NOT NULL
);
CREATE TABLE order_items (
	id       INTEGER PRIMARY KEY,
	order_id INTEGER NOT NULL REFERENCES orders (id),
	sku      TEXT NOT NULL,
	qty      INTEGER NOT NULL
);
CREATE INDEX order_items_order_id ON order_items (order_id);
`

// ANCHOR_END: schema

// seed 是四張訂單：第 3 張還沒放任何品項。
const seed = `
INSERT INTO orders (id, customer) VALUES (1, 'ann'), (2, 'bob'), (3, 'cara'), (4, 'dan');
INSERT INTO order_items (order_id, sku, qty) VALUES
	(1, 'pen', 2), (1, 'ink', 1),
	(2, 'pad', 5),
	(4, 'cap', 1), (4, 'tape', 3), (4, 'glue', 2);
`

var dbSeq atomic.Int64

// openDB 開一個全新的資料庫，建好 schema 後依序執行 fixtures。
// 共享快取的具名 in-memory DSN 讓連線池多開的連線看到同一份資料，
// 每次換一個名字，測試之間互不干擾。
func openDB(fixtures ...string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", fmt.Sprintf("file:orders%d?mode=memory&cache=shared", dbSeq.Add(1)))
	if err != nil {
		return nil, err
	}
	for _, stmt := range append([]string{schema}, fixtures...) {
		if _, err := db.Exec(stmt); err != nil {
			return nil, errors.Join(err, db.Close())
		}
	}
	return db, nil
}
