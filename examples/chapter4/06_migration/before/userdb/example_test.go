package userdb_test

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter4/06_migration/before/userdb"
)

// ANCHOR: example

func ExampleOpen_existingDatabase() {
	ctx := context.Background()
	dsn := "file:before_existing?mode=memory&cache=shared"

	// 正式環境的資料庫是第一版建的：users 還沒有 name。
	prod, err := sql.Open("sqlite", dsn)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer prod.Close()
	if _, err := prod.ExecContext(ctx, `CREATE TABLE users (id INTEGER PRIMARY KEY, email TEXT NOT NULL UNIQUE)`); err != nil {
		fmt.Println(err)
		return
	}

	// 第二版上線，照常啟動：IF NOT EXISTS 看到 users 已經存在，整句跳過，沒有任何錯誤。
	db, err := userdb.Open(ctx, dsn)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer db.Close()
	fmt.Println("open: ok")

	// 第一次寫入 name 才發現。
	fmt.Println("add:", userdb.AddUser(ctx, db, "ann@example.com", "Ann"))

	// 有人登入正式環境手動補欄位；另一個人不知道已經補過，又跑一次。
	for range 2 {
		_, err := db.ExecContext(ctx, `ALTER TABLE users ADD COLUMN name TEXT NOT NULL DEFAULT ''`)
		fmt.Println("alter:", err)
	}
	// Output:
	// open: ok
	// add: SQL logic error: table users has no column named name (1)
	// alter: <nil>
	// alter: SQL logic error: duplicate column name: name (1)
}

// ANCHOR_END: example
