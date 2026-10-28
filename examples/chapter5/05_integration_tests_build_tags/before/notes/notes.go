// Package notes 把短筆記存在一個 SQLite 檔案裡。
package notes

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	_ "modernc.org/sqlite" // 註冊 "sqlite" driver
)

// ErrEmpty 表示筆記去掉前後空白之後是空的。
var ErrEmpty = errors.New("empty note")

// Normalize 去掉前後空白，回傳可以存的內容；空的筆記回傳 ErrEmpty。
func Normalize(text string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", ErrEmpty
	}
	return text, nil
}

// Store 把筆記存在 SQLite 檔案裡。
type Store struct {
	db *sql.DB
}

// Open 開啟 path 的資料庫檔（不存在就建立），並確認連得上。
// 資料表只有一張，用 CREATE TABLE IF NOT EXISTS 建；schema 會演進時，
// 改用第 4 章〈migration〉的做法。
func Open(ctx context.Context, path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	const schema = `CREATE TABLE IF NOT EXISTS notes (id INTEGER PRIMARY KEY, text TEXT NOT NULL)`
	if _, err := db.ExecContext(ctx, schema); err != nil {
		return nil, errors.Join(fmt.Errorf("init %s: %w", path, err), db.Close())
	}
	return &Store{db: db}, nil
}

// Add 正規化 text 後存起來，回傳新筆記的 ID；空的筆記回傳包著 ErrEmpty 的錯誤。
func (s *Store) Add(ctx context.Context, text string) (int64, error) {
	text, err := Normalize(text)
	if err != nil {
		return 0, fmt.Errorf("add note: %w", err)
	}
	res, err := s.db.ExecContext(ctx, `INSERT INTO notes (text) VALUES (?)`, text)
	if err != nil {
		return 0, fmt.Errorf("insert note: %w", err)
	}
	return res.LastInsertId()
}

// List 依新增的順序回傳所有筆記。
func (s *Store) List(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT text FROM notes ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list notes: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var text string
		if err := rows.Scan(&text); err != nil {
			return nil, fmt.Errorf("scan note: %w", err)
		}
		out = append(out, text)
	}
	return out, rows.Err()
}

// Close 關閉資料庫。
func (s *Store) Close() error {
	return s.db.Close()
}
