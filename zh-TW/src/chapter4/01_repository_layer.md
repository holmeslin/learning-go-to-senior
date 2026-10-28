# 要不要有 repository 這一層

## 本集目標

看到 handler 裡的 SQL 時，判斷要不要把它搬進專屬的 store；要搬的話，能說出它的方法怎麼命名、回傳什麼、`sql.ErrNoRows` 怎麼處理。

## 情境

你維護一個訂單服務，資料放在 SQLite 的 `orders` 資料表，有三個端點：查單筆訂單、列出某使用者還沒出貨的訂單、關閉帳號（使用者還有未出貨的訂單就拒絕）。訂單的狀態是 `pending`、`paid`、`shipped`，「還沒出貨」指前兩種。

`sql.Open`、`QueryRow`、`Query` 與 `Rows` 怎麼用，前作〈[`sql.Open` 與 SQLite driver](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter15/01_open_sqlite.html)〉、〈[`QueryRow`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter15/03_query_row.html)〉與〈[`Query` 與 `Rows`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter15/04_query_rows.html)〉已經講過。這集只談這些 SQL 該寫在哪裡；範例的資料表由測試建立，schema 怎麼管理，本章〈migration〉會談。

## 常見寫法

每個 handler 自己寫 SQL，欄位直接 `Scan` 進區域變數，判斷也寫在旁邊。列出未出貨訂單的端點，和關閉帳號的檢查：

```go,ignore
{{#include ../../../examples/chapter4/01_repository_layer/before/shop.go:list}}
```

```go,ignore
{{#include ../../../examples/chapter4/01_repository_layer/before/shop.go:close}}
```

兩個端點各自跑得動，錯誤也都處理了。完整程式碼在 `examples/chapter4/01_repository_layer/before/`。

## 問題在哪

假設 `paid` 是後來才加的狀態，產品說已付款、還沒出貨的訂單也算「未完成」。有人改了列表端點的 SQL，關閉帳號那一份是別人寫的，沒有人想到它。使用者只有一筆已付款的訂單：

```go,ignore
{{#include ../../../examples/chapter4/01_repository_layer/before/example_test.go:example}}
```

列表說有一筆未完成的訂單，關閉帳號卻回了 `204`。「未完成」這個意圖分散在兩段 SQL 裡，各自演變，因為它沒有名字，編譯器和測試都不知道這兩處該一致。

查單筆訂單的 handler 還有第二個問題：

```go,ignore
{{#include ../../../examples/chapter4/01_repository_layer/before/shop.go:get}}
```

HTTP 層要決定回 404，就得知道 `database/sql` 用 `sql.ErrNoRows` 表示「沒有這一筆」。哪天排程或命令列也要查訂單，這段判斷得再寫一次。資料表改欄位時，每個 handler 裡的 SQL 也都得找出來改。

另一個極端也解決不了：有人看到 SQL 散著，就包一層什麼都能存的 `Repository[T]`，方法只有 `Get`、`List`、`Save`、`Delete`。光看方法的形狀就知道，它說不出「某使用者還沒出貨的訂單」：呼叫端只剩兩條路，`List` 全撈回來自己篩，或把 `WHERE` 的片段傳進去，SQL 又回到呼叫端。查詢的意圖還是沒有名字。

## 較好的做法

SQL 搬進領域套件 `order` 的具體型別 `Store`；HTTP 另成一個套件，只認得 `order`：

```text
after/
├── order/      Order、ErrNotFound、Store（所有 SQL 都在這裡）
└── shophttp/   把 Store 透過 HTTP 提供出去，不 import database/sql
```

`Store` 放在 `order` 而不是另開一個 `repository` 套件，理由和第 1 章〈[套件切分的依據（依職責，不依型別）](../chapter1/01_package_by_responsibility.md)〉一樣：它為訂單而改。它是具體型別，沒有先配一個介面，第 3 章〈[介面何時是多餘的](../chapter3/03_redundant_interface.md)〉談過原因。它有三個特點：

- 方法依用途命名，不是通用的 `Query(where string)`，查詢的意圖留在簽名裡。
- 回傳領域型別 `Order`，不回傳 `*sql.Rows`。
- 把 `sql.ErrNoRows` 翻成自己的 `ErrNotFound`。

```go,ignore
{{#include ../../../examples/chapter4/01_repository_layer/after/order/store.go:open}}
```

`OpenOrdersFor` 是「未完成」唯一的定義：`defer rows.Close()`，迴圈後檢查 `rows.Err()`，沒有訂單回傳空結果，不是錯誤。已出貨與別人的訂單都不會出現：

```go,ignore
{{#include ../../../examples/chapter4/01_repository_layer/after/order/example_test.go:open}}
```

找單筆訂單的方法：

```go,ignore
{{#include ../../../examples/chapter4/01_repository_layer/after/order/store.go:find}}
```

只有 `sql.ErrNoRows` 才翻譯成 `ErrNotFound`，用 `%w` 包住的是**自己的**錯誤，原因與做法見第 2 章〈[包裝錯誤：`%w` 該不該暴露給呼叫端](../chapter2/02_wrap_or_expose.md)〉。其他錯誤則刻意原樣包裝，因為 `context.Canceled` 與逾時是呼叫端需要判斷的（本章〈查詢逾時該設在哪一層〉會談），文件註解也寫明了。實際長相：

```go,ignore
{{#include ../../../examples/chapter4/01_repository_layer/after/order/example_test.go:notfound}}
```

測試放在外部套件，驗證翻譯後自己的錯誤成立、`sql.ErrNoRows` 不再透出：

```go,ignore
{{#include ../../../examples/chapter4/01_repository_layer/after/order/store_test.go:test}}
```

這些測試直接對真的 SQLite 跑。`:memory:` 在每條連線上都是各自獨立的資料庫，連線池多開一條連線就會看到空的資料庫，所以測試用具名的共享快取 `file:<名稱>?mode=memory&cache=shared`，每個測試各用一個名稱（本章後面用記憶體資料庫的範例都這樣做，不再重複說明）。為什麼要對真的資料庫測，本章〈測試資料層：fake 與真實資料庫〉會談。

現在 handler 不碰 SQL，關閉帳號只問 `OpenOrdersFor`，同樣的情境結果就對了：

```go,ignore
{{#include ../../../examples/chapter4/01_repository_layer/after/shophttp/handler.go:close}}
```

```go,ignore
{{#include ../../../examples/chapter4/01_repository_layer/after/shophttp/example_test.go:example}}
```

查單筆訂單的 handler 改成判斷 `order.ErrNotFound`，不再認得 `database/sql`：

```go,ignore
{{#include ../../../examples/chapter4/01_repository_layer/after/shophttp/handler.go:get}}
```

完整程式碼在 `examples/chapter4/01_repository_layer/after/`。

## 取捨

多這一層有代價：

- 每個查詢都要先取名字、寫成方法，再從呼叫端用。只在一個地方用到的短查詢，也得走這一圈。
- store 會隨需求變大。方法越多，越要留意兩個方法的 SQL 是不是各自重複了同一個定義（例如又寫了一份「未出貨」）。
- 它不保證你能隨時換資料庫。SQL 還是寫給特定資料庫的，只是要改的地方集中在一個套件。

直接在 handler 寫 SQL 在這些情況下反而是對的：

- 小工具、一次性的腳本，只有一兩個查詢：多一個套件、一個型別、一組錯誤翻譯，換不到什麼。
- 查詢只在一個地方用，也沒有要統一的意圖：抽出來只是把 SQL 搬到另一個檔案。
- 資料層還在摸索，欄位與查詢天天在變。等形狀穩了再抽，改起來也不遲。

判斷的方法：同一個意圖有了第二個呼叫端，或除了 HTTP 層以外也要判斷「找不到」的時候，就是該有 store 的時候。

## 重點整理

- SQL 散在 handler 裡，「未完成」這類意圖會在多處各自演變，沒有名字就沒有一致性。
- store 放在領域套件裡，是具體型別，方法依用途命名、回傳領域型別；通用 CRUD 的 `Repository[T]` 表達不出查詢的意圖。
- 在 store 這一層把 `sql.ErrNoRows` 翻成自己的 `ErrNotFound`，HTTP 層就不必認得 `database/sql`。
- after 的測試對真的 SQLite 跑，驗證「找不到」是領域錯誤，也驗證資料庫其他失敗不會被誤判成找不到。
- 小工具與只用一次的少數查詢，直接寫 SQL 比較簡單；有第二個呼叫端時再抽出 store。
