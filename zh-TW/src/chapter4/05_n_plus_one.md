# N+1 與查詢組裝

## 本集目標

要列出一份清單、每一筆還要帶著自己的子資料時，能認出「迴圈裡逐筆查詢」就是 N+1，改成一次取回、在 Go 裡組裝，把查詢次數壓成常數；也能說出什麼情況下 N+1 其實可以留著。

## 情境

你在寫訂單服務的「最近訂單」：取最近的幾張訂單，每張都要帶著自己的品項。資料分在兩張資料表，一張訂單有零到多個品項：

```go,ignore
{{#include ../../../examples/chapter4/05_n_plus_one/before/setup_test.go:schema}}
```

`Query`、`Rows` 的用法，前作〈[`Query` 與 `Rows`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter15/04_query_rows.html)〉已經講過，`?` 佔位符見〈[`Exec`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter15/02_exec.html)〉。這集談的是查詢怎麼分配，不是 API。

`Store` 只依賴一個只有 `QueryContext` 的小介面（介面宣告在使用端，見第 3 章〈[在使用端定義介面](../chapter3/01_define_interface_at_consumer.md)〉）。`*sql.DB` 與 `*sql.Tx` 都符合，為什麼這樣設計，本章〈[交易邊界放在哪裡](02_transaction_boundary.md)〉談過；這集用它的另一個好處：包住它，就數得出查詢次數。

```go,ignore
{{#include ../../../examples/chapter4/05_n_plus_one/before/orders.go:querier}}
```

## 常見寫法

先查出清單，再對清單裡的每一張訂單，各查一次它的品項：

```go,ignore
{{#include ../../../examples/chapter4/05_n_plus_one/before/orders.go:recent}}
```

```go,ignore
{{#include ../../../examples/chapter4/05_n_plus_one/before/orders.go:itemsof}}
```

每個函式只做一件事，`rows` 有關閉、錯誤有處理，讀起來沒什麼好挑的。`RecentOrders` 先把清單讀完，才開始逐筆查；邊走訪 `rows` 邊查，會同時占住兩條連線，本章〈[連線池要設多大、設錯會怎樣](04_pool_size.md)〉談過。完整程式碼在 `examples/chapter4/05_n_plus_one/before/`。

## 問題在哪

測試只看結果，這段程式碼的結果完全正確，問題出在查詢次數。用一個包住 `Querier` 的計數器，每次 `QueryContext` 加一（`Store` 只用這個方法，所以每次查詢都數得到）：

```go,ignore
{{#include ../../../examples/chapter4/05_n_plus_one/before/example_test.go:counter}}
```

分別取 1、2、4 張訂單，數出查詢次數。`// Output:` 由 `go test` 逐字比對，計數不靠計時：

```go,ignore
{{#include ../../../examples/chapter4/05_n_plus_one/before/example_test.go:example}}
```

次數是 `1 + N`：一次清單，加上每張訂單一次。這就是名字的由來。SQLite 在同一個程式裡，多幾次查詢感覺不出來；資料庫換成另一台機器，每次查詢都是一個網路來回，N 是 100 就是 101 個來回，而且 N 會隨資料量和 `limit` 成長。

這種問題不會被測試抓到：結果對，次數錯。查詢還常常藏在 `itemsOf` 這樣的小函式後面，看呼叫端的程式碼，看不出迴圈裡有一次資料庫來回。

## 較好的做法

查詢次數改成兩次：一次清單，一次用 `IN` 取回這幾張訂單的全部品項，再用 `map` 依訂單編號分組：

```go,ignore
{{#include ../../../examples/chapter4/05_n_plus_one/after/orders.go:recent}}
```

```go,ignore
{{#include ../../../examples/chapter4/05_n_plus_one/after/orders.go:itemsbyorder}}
```

有幾個細節：

- **佔位符逐一產生。** `IN (?, ?, ?)` 的 `?` 依編號數量產生，編號本身放進 `args`。SQL 字串裡只有 `?`，沒有任何資料。不要用 `fmt.Sprintf` 把編號拼進去：這裡的編號是 `int`，拼起來看似無害，但同樣的寫法換成 email 或 SKU 就成了 SQL injection。完整說明，第 11 章〈SQL injection 與參數化查詢〉會談。
- **組裝靠 `map`。** `byOrder` 以訂單編號為鍵，最後依第一次查詢的順序把品項接回每張訂單。沒有品項的訂單不在 `map` 裡，取出來是 `nil`，訂單本身仍在結果中，所以不會因為沒有子資料而消失。
- **空清單提早回傳。** 沒有訂單就不用查品項，也避開空的 `IN ()`：它不是標準 SQL，SQLite 放行，別的資料庫未必。

同樣的 `Example`，只換成 `after` 的 `Store`：

```go,ignore
{{#include ../../../examples/chapter4/05_n_plus_one/after/example_test.go:example}}
```

不論取幾張，都是兩次。`orders_test.go` 鎖住四件事：組裝結果正確（包含沒有品項的訂單，以及品項的順序）、`limit` 以外的訂單品項不會混進來、查詢次數不隨訂單數成長（0 張訂單 1 次，1 張與 25 張都是 2 次）、`limit` 超出範圍時在查詢之前就回傳錯誤。完整程式碼在 `examples/chapter4/05_n_plus_one/after/`。

## 取捨

兩次查詢不是白拿的：

- **自己組裝，程式碼變長。** 分組、接回、空清單的處理，都是 before 不用寫的。
- **`IN` 的參數有上限。** SQLite 的預設上限，3.32.0 以後是 32766，更早是 999（見 [SQLite 的限制說明](https://www.sqlite.org/limits.html)）；別的資料庫各有各的數字。清單可能很長時，要分批（`slices.Chunk`），查詢次數變成「批數 + 1」，仍然遠少於 `1 + N`，只是不再是常數。這裡選擇不分批：`RecentOrders` 拒絕小於 1 或超過 `MaxRecentLimit`（1000，遠低於 32766）的 `limit`，「最多只查兩次」才站得住；代價是呼叫端想要更多筆時得自己分頁。
- **也可以用 `JOIN`。** `orders LEFT JOIN order_items` 一次就取回，連第二次查詢都省了。代價是每個品項一列，訂單的欄位在每一列重複，欄位寬或品項多時傳輸量變大；沒有品項的訂單要用 `LEFT JOIN` 才不會消失，它的品項欄位是 `NULL`，要用 `sql.Null*` 或指標去掃（前作〈[`NULL` 處理](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter15/07_null.html)〉）；而且 `LIMIT` 限制的是 JOIN 之後的「列」，不是訂單數，要取最近 10 張訂單得改寫成子查詢。

N+1 在這些情況下反而是合理的：

- **N 很小又固定。** 例如一頁最多三筆、或一次請求只查一張訂單，逐筆查最短也最好讀，多出來的來回有限。
- **一次取回的資料太大。** 每筆子資料很重（例如整份文件），一次全取回會占住大量記憶體，逐筆或分批處理反而穩。
- **資料庫在同一台機器上、而且查詢很便宜。** SQLite 就是這樣，但換到正式環境的資料庫要重新評估，別拿本機的感覺當依據。

判斷的方法是把次數寫進測試，像這集的 `Example` 一樣數出來，而不是憑感覺。上線後怎麼觀察查詢次數，第 9 章〈[分散式追蹤與 OpenTelemetry](../chapter9/04_tracing_opentelemetry.md)〉談過。

## 重點整理

- N+1 是「一次清單加 N 次逐筆查詢」：結果對、次數錯，測試看不出來，要把查詢次數數出來。
- 計數不靠計時：包住只有 `QueryContext` 的小介面，每次呼叫加一，輸出次數固定。
- 修法是一次取回、在 Go 裡組裝：`IN` 的 `?` 逐一產生，值走參數，不拼字串；用 `map` 依父資料分組，沒有子資料的父資料仍要留在結果裡。
- `IN` 的參數有上限，大清單要分批；`JOIN` 會重複父資料的欄位，`LIMIT` 的對象也變成列。
- N 很小、或子資料很大時，N+1 可能更簡單；先數出次數，再決定。
