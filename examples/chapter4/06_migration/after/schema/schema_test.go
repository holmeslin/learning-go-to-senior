package schema_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"slices"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter4/06_migration/after/migrate"
	"github.com/holmeslin/learning-go-to-senior/examples/chapter4/06_migration/after/schema"
)

// columns 依欄位順序回傳 table 的欄位名稱。
func columns(ctx context.Context, db *sql.DB, table string) ([]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT name FROM pragma_table_info(?) ORDER BY cid`, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

// openMigrated 在暫存目錄建一個檔案資料庫，並套用內嵌的 migration。
func openMigrated(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close db: %v", err)
		}
	})
	if _, err := migrate.Up(context.Background(), db, schema.FS); err != nil {
		t.Fatalf("Up() error = %v", err)
	}
	return db
}

func TestEmptyDatabaseGetsCurrentSchema(t *testing.T) {
	ctx := context.Background()
	db := openMigrated(t)

	for table, want := range map[string][]string{
		"users":             {"id", "email", "name"},
		"audit_log":         {"id", "action"},
		"schema_migrations": {"version", "name"},
	} {
		got, err := columns(ctx, db, table)
		if err != nil {
			t.Fatalf("columns(%s) error = %v", table, err)
		}
		if !slices.Equal(got, want) {
			t.Errorf("%s columns = %v, want %v", table, got, want)
		}
	}
}

func TestRerunIsNoop(t *testing.T) {
	ctx := context.Background()
	db := openMigrated(t)
	if _, err := db.ExecContext(ctx, `INSERT INTO users (email, name) VALUES ('ann@example.com', 'Ann')`); err != nil {
		t.Fatal(err)
	}

	applied, err := migrate.Up(ctx, db, schema.FS)

	if err != nil || len(applied) != 0 {
		t.Fatalf("second Up() = %v, %v; want nothing applied", applied, err)
	}
	var n int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM users`).Scan(&n); err != nil || n != 1 {
		t.Errorf("users rows = %d, %v; want the existing row untouched", n, err)
	}
}

// 003 只新增一個有預設值的欄位：滾動更新時還在跑的舊版程式，INSERT 不帶 name 也照常成功。
func TestOldReleaseStillWorksAfterExpand(t *testing.T) {
	ctx := context.Background()
	db := openMigrated(t)

	if _, err := db.ExecContext(ctx, `INSERT INTO users (email) VALUES ('old@example.com')`); err != nil {
		t.Fatalf("insert as release 1: %v", err)
	}

	var name string
	if err := db.QueryRowContext(ctx, `SELECT name FROM users WHERE email = 'old@example.com'`).Scan(&name); err != nil || name != "" {
		t.Errorf("name = %q, %v; want the empty default", name, err)
	}
}
