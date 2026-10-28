// Package migrate 依編號依序套用 SQL 檔，每個檔案只套用一次。
package migrate

import (
	"cmp"
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"slices"
	"strconv"
	"strings"
)

// 版本表是整個機制唯一不靠 migration 檔建立的表：它的形狀不會變，所以用 IF NOT EXISTS。
const createVersions = `CREATE TABLE IF NOT EXISTS schema_migrations (
	version INTEGER PRIMARY KEY,
	name    TEXT NOT NULL
)`

type migration struct {
	version int
	name    string
}

// ANCHOR: up

// Up 依編號從小到大套用 fsys 根目錄下還沒套用過的 *.sql，回傳這次套用的檔名。
// 檔名必須是「編號_說明.sql」，編號不可重複。每個檔案和它在 schema_migrations 的紀錄
// 在同一個交易裡：失敗的檔案不留任何痕跡，先前已套用的仍然有效。沒有新檔案時什麼都不做。
//
// shortcut: 沒有鎖，兩個程序同時啟動會搶著套用同一個檔案；有多個副本時再補。
func Up(ctx context.Context, db *sql.DB, fsys fs.FS) ([]string, error) {
	todo, err := load(fsys)
	if err != nil {
		return nil, err
	}
	if _, err := db.ExecContext(ctx, createVersions); err != nil {
		return nil, fmt.Errorf("create schema_migrations: %w", err)
	}
	done, err := appliedVersions(ctx, db)
	if err != nil {
		return nil, err
	}

	var applied []string
	for _, m := range todo {
		if done[m.version] {
			continue
		}
		if err := apply(ctx, db, fsys, m); err != nil {
			return applied, fmt.Errorf("apply %s: %w", m.name, err)
		}
		applied = append(applied, m.name)
	}
	return applied, nil
}

func apply(ctx context.Context, db *sql.DB, fsys fs.FS, m migration) error {
	body, err := fs.ReadFile(fsys, m.name)
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
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version, name) VALUES (?, ?)`, m.version, m.name); err != nil {
		return err
	}
	return tx.Commit()
}

// ANCHOR_END: up

func appliedVersions(ctx context.Context, db *sql.DB) (map[int]bool, error) {
	rows, err := db.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("read schema_migrations: %w", err)
	}
	defer rows.Close()

	done := map[int]bool{}
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("read schema_migrations: %w", err)
		}
		done[v] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read schema_migrations: %w", err)
	}
	return done, nil
}

// load 找出所有 migration，依編號（不是依檔名）排序。
// 編號重複的話，後來的檔案會被當成「已套用」默默跳過，所以在動手之前就拒絕。
func load(fsys fs.FS) ([]migration, error) {
	names, err := fs.Glob(fsys, "*.sql")
	if err != nil {
		return nil, err
	}
	var ms []migration
	for _, name := range names {
		prefix, _, _ := strings.Cut(name, "_")
		v, err := strconv.Atoi(prefix)
		if err != nil {
			return nil, fmt.Errorf("migration file %q: name must start with a number and an underscore", name)
		}
		ms = append(ms, migration{version: v, name: name})
	}
	slices.SortFunc(ms, func(a, b migration) int { return cmp.Compare(a.version, b.version) })
	for i := 1; i < len(ms); i++ {
		if ms[i].version == ms[i-1].version {
			return nil, fmt.Errorf("migration files %q and %q have the same number", ms[i-1].name, ms[i].name)
		}
	}
	return ms, nil
}
