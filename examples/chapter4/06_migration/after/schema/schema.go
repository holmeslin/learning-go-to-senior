// Package schema 是這個應用程式的資料庫 schema：編號的 SQL 檔，隨執行檔一起發布。
package schema

import "embed"

// ANCHOR: embed

// FS 內嵌所有編號的 SQL 檔，交給 migrate.Up 套用。
//
//go:embed *.sql
var FS embed.FS

// ANCHOR_END: embed
