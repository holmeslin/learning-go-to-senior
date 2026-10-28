// Package userdb 是第二版的資料層：啟動時，各功能自己確認需要的資料表存在。
package userdb

import (
	"context"
	"database/sql"
	"errors"

	_ "modernc.org/sqlite"
)

// ANCHOR: open

// Open 開啟資料庫，並讓每個功能各自建立它需要的資料表。
func Open(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	for _, ensure := range []func(context.Context, *sql.DB) error{ensureUsers, ensureAuditLog} {
		if err := ensure(ctx, db); err != nil {
			return nil, errors.Join(err, db.Close())
		}
	}
	return db, nil
}

// ANCHOR_END: open
