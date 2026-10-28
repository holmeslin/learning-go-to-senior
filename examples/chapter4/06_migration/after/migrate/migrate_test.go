package migrate_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	_ "modernc.org/sqlite"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter4/06_migration/after/migrate"
)

// openDB 在測試專屬的暫存目錄建一個檔案資料庫，每個測試各自一份。
func openDB(t *testing.T) *sql.DB {
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
	return db
}

// query 回傳單一欄位的查詢結果。
func query(t *testing.T, db *sql.DB, q string) []string {
	t.Helper()
	rows, err := db.Query(q)
	if err != nil {
		t.Fatalf("query %q: %v", q, err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

const tables = `SELECT name FROM sqlite_master WHERE type = 'table' ORDER BY name`

func sqlFile(s string) *fstest.MapFile { return &fstest.MapFile{Data: []byte(s)} }

func TestUpAppliesByNumberNotByName(t *testing.T) {
	db := openDB(t)
	// 依檔名排序 10 會排在 2 前面；2 要用到 1 建的表，10 要用到 2 加的欄位。
	fsys := fstest.MapFS{
		"1_create_a.sql": sqlFile(`CREATE TABLE a (id INTEGER);`),
		"2_add_x.sql":    sqlFile(`ALTER TABLE a ADD COLUMN x TEXT;`),
		"10_use_x.sql":   sqlFile(`INSERT INTO a (id, x) VALUES (1, 'ok');`),
	}

	applied, err := migrate.Up(context.Background(), db, fsys)
	if err != nil {
		t.Fatalf("Up() error = %v", err)
	}
	if want := []string{"1_create_a.sql", "2_add_x.sql", "10_use_x.sql"}; !slices.Equal(applied, want) {
		t.Errorf("applied = %v, want %v", applied, want)
	}
}

func TestUpAppliesOnlyNewFiles(t *testing.T) {
	ctx := context.Background()
	db := openDB(t)
	fsys := fstest.MapFS{"1_create_a.sql": sqlFile(`CREATE TABLE a (id INTEGER);`)}
	if _, err := migrate.Up(ctx, db, fsys); err != nil {
		t.Fatalf("first Up() error = %v", err)
	}

	// 下一個版本多了一個檔案；舊檔案的 CREATE TABLE 若重跑會失敗。
	fsys["2_create_b.sql"] = sqlFile(`CREATE TABLE b (id INTEGER);`)
	applied, err := migrate.Up(ctx, db, fsys)
	if err != nil {
		t.Fatalf("second Up() error = %v", err)
	}
	if want := []string{"2_create_b.sql"}; !slices.Equal(applied, want) {
		t.Errorf("applied = %v, want %v", applied, want)
	}

	applied, err = migrate.Up(ctx, db, fsys)
	if err != nil || len(applied) != 0 {
		t.Errorf("third Up() = %v, %v; want nothing applied", applied, err)
	}
	if got, want := query(t, db, `SELECT version FROM schema_migrations ORDER BY version`), []string{"1", "2"}; !slices.Equal(got, want) {
		t.Errorf("recorded versions = %v, want %v", got, want)
	}
}

func TestFailedMigrationLeavesNothingBehind(t *testing.T) {
	ctx := context.Background()
	db := openDB(t)
	fsys := fstest.MapFS{
		"1_create_a.sql": sqlFile(`CREATE TABLE a (id INTEGER);`),
		// 第一句會成功，第二句失敗：整個檔案都要回滾。
		"2_broken.sql":   sqlFile(`CREATE TABLE b (id INTEGER); INSERT INTO missing VALUES (1);`),
		"3_create_c.sql": sqlFile(`CREATE TABLE c (id INTEGER);`),
	}

	applied, err := migrate.Up(ctx, db, fsys)
	if err == nil || !strings.Contains(err.Error(), "2_broken.sql") {
		t.Fatalf("Up() error = %v, want an error naming 2_broken.sql", err)
	}
	if want := []string{"1_create_a.sql"}; !slices.Equal(applied, want) {
		t.Errorf("applied = %v, want %v (earlier files stay applied)", applied, want)
	}
	if got, want := query(t, db, tables), []string{"a", "schema_migrations"}; !slices.Equal(got, want) {
		t.Errorf("tables = %v, want %v (no b from the broken file, no c after it)", got, want)
	}

	// 修好之後重跑，只補上沒套用的。
	fsys["2_broken.sql"] = sqlFile(`CREATE TABLE b (id INTEGER);`)
	applied, err = migrate.Up(ctx, db, fsys)
	if err != nil {
		t.Fatalf("Up() after fix error = %v", err)
	}
	if want := []string{"2_broken.sql", "3_create_c.sql"}; !slices.Equal(applied, want) {
		t.Errorf("applied after fix = %v, want %v", applied, want)
	}
}

func TestUpRejectsAmbiguousFileSets(t *testing.T) {
	tests := []struct {
		name string
		fsys fstest.MapFS
	}{
		{"duplicate number", fstest.MapFS{
			"4_add_x.sql": sqlFile(`CREATE TABLE x (id INTEGER);`),
			"4_add_y.sql": sqlFile(`CREATE TABLE y (id INTEGER);`),
		}},
		{"no number", fstest.MapFS{
			"create_users.sql": sqlFile(`CREATE TABLE users (id INTEGER);`),
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := openDB(t)

			applied, err := migrate.Up(context.Background(), db, tt.fsys)

			if err == nil {
				t.Fatal("Up() error = nil, want an error")
			}
			if len(applied) != 0 {
				t.Errorf("applied = %v, want nothing", applied)
			}
			if got := query(t, db, tables); len(got) != 0 {
				t.Errorf("tables = %v, want none (nothing runs before the file set is accepted)", got)
			}
		})
	}
}
