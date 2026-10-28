# migration

## 本集目標

寫得出一個最小的 migration 機制：編號的 SQL 檔、一張版本表、依序套用、重跑不會重複；並判斷 schema 該怎麼改，部署時還在跑的舊版程式才不會壞掉。

## 情境

你的服務用 SQLite，第一版有 `users` 與 `audit_log` 兩張表。第二版要讓使用者多一個 `name` 欄位。服務有開發、測試、正式三套環境，正式環境的資料庫已經有資料，而且是滾動更新：有一段時間新舊兩版程式同時在跑。

開連線與交易的機制，前作〈[`sql.Open` 與 SQLite driver](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter15/01_open_sqlite.html)〉與〈[交易](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter15/06_transaction.html)〉已經講過，`//go:embed` 在前作〈[`//go:embed`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter11/05_go_embed.html)〉。這集只談 schema 怎麼隨版本演進。

## 常見寫法

啟動時，每個功能各自確認需要的資料表存在，用 `CREATE TABLE IF NOT EXISTS`：

```go,ignore
{{#include ../../../examples/chapter4/06_migration/before/userdb/db.go:open}}
```

```go,ignore
{{#include ../../../examples/chapter4/06_migration/before/userdb/users.go:users}}
```

`audit_log` 的建表在 `audit.go`，長得一樣。第二版要加 `name`，就直接改 `users` 那句 `CREATE TABLE`：本機刪掉資料庫重跑，一切正常。完整程式碼在 `examples/chapter4/06_migration/before/`。

## 問題在哪

本機的資料庫每次都是新的，正式環境的不是。下面的 `Example` 模擬正式環境：資料庫是第一版建的，第二版照常啟動。`// Output:` 是實際輸出，由 `go test` 逐字比對：

```go,ignore
{{#include ../../../examples/chapter4/06_migration/before/userdb/example_test.go:example}}
```

三件事一次看到：

- **改過的 `CREATE TABLE` 對舊資料庫不起作用。** `IF NOT EXISTS` 看到表已經在，整句跳過。`Open` 沒有任何錯誤，直到第一次寫入 `name` 才失敗（`add:` 那行）。啟動成功的當下，沒有人知道這個資料庫缺欄位。
- **手動 `ALTER` 不能重跑。** 這句 `ALTER` 沒有「已經有了就跳過」的防護，第二次執行就失敗（最後一行）。誰跑過、哪個資料庫還沒跑，只存在人的記憶和聊天紀錄裡。
- **沒有版本紀錄。** 開發、測試、正式三套資料庫各自停在不同的版本，而你問不出它們現在是哪一版。想把 `ALTER` 寫進啟動流程也不行：得先知道這個資料庫跑過了沒有。

## 較好的做法

每個 schema 變更是一個編號的 SQL 檔。`003` 就是第二版要的變更：

```sql
{{#include ../../../examples/chapter4/06_migration/after/schema/003_add_user_name.sql}}
```

檔案內嵌進執行檔，程式與 SQL 一起發布，不會出現程式是新的、SQL 忘了部署：

```go,ignore
{{#include ../../../examples/chapter4/06_migration/after/schema/schema.go:embed}}
```

套用的是 `migrate.Up`：

```go,ignore
{{#include ../../../examples/chapter4/06_migration/after/migrate/migrate.go:up}}
```

幾個細節：

- **版本表 `schema_migrations` 記錄套用過的編號。** 它是整個機制裡唯一用 `IF NOT EXISTS` 建的表，因為它的形狀永遠不會變。
- **依編號排序，不是依檔名。** 依檔名排的話 `10_` 會排在 `2_` 前面；檔名補零（`001_`）只是讓列表好看。編號重複則在動手前就拒絕：否則後來的檔案會被當成「已套用」默默跳過。
- **每個檔案與它的紀錄在同一個交易。** 用的是前作教過的 `defer tx.Rollback()` 加上檢查 `Commit` 的錯誤。檔案裡第二句失敗，第一句也一起回滾，下次啟動會從這個檔案重來。

用同一個情境再跑一次。`release1` 把前兩個檔案放進一個 `fs.FS`，模擬第一版程式內嵌的內容：

```go,ignore
{{#include ../../../examples/chapter4/06_migration/after/schema/example_test.go:release1}}
```

```go,ignore
{{#include ../../../examples/chapter4/06_migration/after/schema/example_test.go:example}}
```

第二版啟動只套用 `003`；再啟動一次什麼也不做。before 第二次 `ALTER` 會失敗，這裡不會，因為版本表記得 3 已經套用過。正式環境是哪一版？`SELECT max(version) FROM schema_migrations` 就是答案。（`columns` 在同一個套件的 `schema_test.go`，查 `pragma_table_info` 印出欄位名稱。）

測試驗證空資料庫套用後的欄位、重跑是 no-op 且資料不動；`migrate_test.go` 另外用 `fstest.MapFS` 驗證回滾與排序。完整程式碼在 `examples/chapter4/06_migration/after/`。

## 取捨

這個最小版本有幾個明確的限制：

- **套用過的檔案不能再改。** 版本表只記編號，不記內容。改了舊資料庫不會重跑，新建的資料庫卻拿到新內容，兩邊的 schema 就分岔了。要改就新增一個編號。
- **沒有鎖。** 兩個程序同時啟動，都會先讀到「還沒套用」，然後搶著套用同一個檔案，至少有一個會出錯。副本不只一個時，改成部署流程裡先單獨跑一次 migration，再滾動更新服務，或用資料庫的鎖。
- **每個檔案包進交易，靠的是 SQLite 的 DDL 可以回滾。** MySQL 的 DDL 會隱式提交，PostgreSQL 的 `CREATE INDEX CONCURRENTLY` 不能放在交易裡，換了資料庫，「失敗就整個回滾」不一定還成立。
- **down migration 值不值得寫，要看你敢不敢跑它。** 寫了就得測，沒測過的 down 在出事時比沒有更危險；`DROP COLUMN` 之後資料也回不來。不少團隊出事時選擇再寫一個往前的 migration。

**零停機：先擴充，再收斂。** 滾動更新期間新舊兩版同時跑，所以每個 migration 都得讓舊版與新版都能用。`003` 只新增有預設值的欄位，舊版的 `INSERT INTO users (email)` 照常成功（`TestOldReleaseStillWorksAfterExpand`）。如果把 `email` 直接改名或刪掉，舊版的 SQL 立刻找不到欄位。要改名，拆成幾步：先新增 `login`（擴充），新版同時寫兩個欄位並補齊舊資料；等所有實例都換成新版、不會再回頭，下一次部署才刪掉 `email`（收斂）。

正式專案通常直接用現成工具，例如 goose、golang-migrate。骨架和這集相同（編號的檔案加一張版本表），另外處理鎖、down、命令列這些最小版本沒做的事。本書不示範，也不引入。migration 在測試裡怎麼套用，本章〈測試資料層：fake 與真實資料庫〉會談。

before 在這些情況下反而夠用：一次性腳本、原型，或資料庫每次都從頭建的場合（測試用的記憶體資料庫、可以丟掉重建的快取）。沒有「舊資料庫」要照顧，`IF NOT EXISTS` 就夠。一旦有一個丟不掉的資料庫、又需要改 schema，就是該換的時候；拖得越久，越難弄清楚每個環境現在到底長什麼樣。

## 重點整理

- `CREATE TABLE IF NOT EXISTS` 只管「有沒有表」，管不到表長得對不對；手動 `ALTER` 不能重跑，也沒有人知道哪個資料庫跑過。
- 最小的 migration：編號的 SQL 檔、`//go:embed`、一張版本表；依編號依序套用，每個檔案與它的紀錄在同一個交易，重跑是 no-op。
- 套用過的檔案不要改，要改就新增編號；編號重複要在動手前拒絕。
- 零停機要先擴充（新增有預設值的欄位），等新舊版都不再依賴舊結構，下一次部署才收斂（刪掉舊欄位）。
- 正式專案通常用 goose、golang-migrate 這類工具；一次性或可重建的資料庫，`IF NOT EXISTS` 就夠。
