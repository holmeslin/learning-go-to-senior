package schema_test

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log"
	"testing/fstest"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter4/06_migration/after/migrate"
	"github.com/holmeslin/learning-go-to-senior/examples/chapter4/06_migration/after/schema"
)

// ANCHOR: release1

// release1 是第一版程式內嵌的檔案：只有編號 1、2。
func release1() fs.FS {
	files := fstest.MapFS{}
	for _, name := range []string{"001_create_users.sql", "002_create_audit_log.sql"} {
		data, err := schema.FS.ReadFile(name)
		if err != nil {
			log.Fatal(err)
		}
		files[name] = &fstest.MapFile{Data: data}
	}
	return files
}

// ANCHOR_END: release1

// ANCHOR: example

func Example() {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:schema_example?mode=memory&cache=shared")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// 第一版上線：資料庫是空的，兩個檔案都要套用。
	applied, err := migrate.Up(ctx, db, release1())
	fmt.Println("release 1:", applied, err)

	// 第二版上線：資料庫已經在第 2 版，只有新檔案會套用。
	applied, err = migrate.Up(ctx, db, schema.FS)
	fmt.Println("release 2:", applied, err)

	// 重新啟動：沒有新檔案，什麼都不做。
	applied, err = migrate.Up(ctx, db, schema.FS)
	fmt.Println("restart:", applied, err)

	var version int
	if err := db.QueryRowContext(ctx, `SELECT max(version) FROM schema_migrations`).Scan(&version); err != nil {
		log.Fatal(err)
	}
	cols, err := columns(ctx, db, "users")
	fmt.Println("version:", version)
	fmt.Println("users:", cols, err)
	// Output:
	// release 1: [001_create_users.sql 002_create_audit_log.sql] <nil>
	// release 2: [003_add_user_name.sql] <nil>
	// restart: [] <nil>
	// version: 3
	// users: [id email name] <nil>
}

// ANCHOR_END: example
