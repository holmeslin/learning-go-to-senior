# SQL injection 與參數化查詢

## 本集目標

寫會帶使用者輸入的查詢時，能分辨哪些東西可以走 `?` 佔位符、哪些不行（排序欄位、表名、`IN` 的個數），並對後者用白名單與產生佔位符處理；也能說出為什麼換成 ORM 或 query builder 仍然要檢查。

## 情境

你在寫圖書館的借閱查詢：會員查自己借了什麼書，排序方式由網址的 `order` 參數決定；還有一支「這幾本書我借過沒有」，一次帶好幾個書名。資料表只有一張：

```go,ignore
{{#include ../../../examples/chapter11/02_sql_injection/before/setup_test.go:schema}}
```

```go,ignore
{{#include ../../../examples/chapter11/02_sql_injection/before/setup_test.go:seed}}
```

`?` 佔位符與 SQL injection 是什麼，前作〈[`Exec`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter15/02_exec.html)〉與〈[prepared statement](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter15/05_prepared_statement.html)〉已經講過，不再重複。第 4 章〈[N+1 與查詢組裝](../chapter4/05_n_plus_one.md)〉的 `IN` 已經逐一產生佔位符，並預告了這一集。這集談的是判斷：值走參數之外，查詢裡還有哪些部分由輸入決定。

## 常見寫法

用 `fmt.Sprintf` 把輸入放進 SQL，單引號把值框起來：

```go,ignore
{{#include ../../../examples/chapter11/02_sql_injection/before/loans.go:bymember}}
```

`IN` 的版本把每個書名加上引號再接起來：

```go,ignore
{{#include ../../../examples/chapter11/02_sql_injection/before/loans.go:bytitles}}
```

這兩個函式讀起來沒什麼問題，正常的輸入也都查得到。`before/` 的套件文件註解寫明這是刻意示範的漏洞，不要照用；完整程式碼在 `examples/chapter11/02_sql_injection/before/`。

## 問題在哪

先用一個包住 `Querier` 的記錄器，看實際送進資料庫的 SQL（`Querier` 與第 4 章同一個做法）：

```go,ignore
{{#include ../../../examples/chapter11/02_sql_injection/before/example_test.go:recorder}}
```

用 `ann` 查，與用一段特別構造的字串查：

```go,ignore
{{#include ../../../examples/chapter11/02_sql_injection/before/example_test.go:example}}
```

第二個輸入的單引號提早結束了字串，`OR '1'='1'` 讓條件永遠成立，結果多出 `bob` 與 `o'neil` 的借閱。輸入從「值」變成了「SQL 的一部分」。同樣的手法可以接上 `UNION` 讀別張表；這裡只示範到證明漏洞存在為止。

- **`IN` 也一樣。** 書名 `x') OR ('1'='1` 跳出引號，讓所有人的借閱都出來（`loans_test.go` 的 `TestByTitlesInjection`）。
- **排序欄位更糟。** `order` 不是字串值，是一段會被執行的 SQL，連引號都不必跳出（`TestByMemberOrderIsExecuted`）。
- **不必有惡意也會壞。** 名字裡本來就有單引號的 `o'neil`，正常查詢就是語法錯誤（`TestByMemberQuoteBreaksQuery`）。
- **自己跳脫引號只是半套。** 每個資料庫的跳脫規則不同，漏掉一處就破功，而且對 `order` 這種不是值的部分完全沒用。

## 較好的做法

值走佔位符，SQL 字串裡只剩 `?`。排序欄位不能走佔位符：`ORDER BY ?` 收到 `"title"`，那是一個常數字串、不是欄位，結果等於沒排序（`TestPlaceholderIsNotAColumn` 實際驗證）。表名與欄位名也一樣，佔位符只能代表值。所以排序用白名單：

```go,ignore
{{#include ../../../examples/chapter11/02_sql_injection/after/loans.go:bymember}}
```

呼叫端給的 `order` 只是白名單的鍵，拼進 SQL 的是程式裡寫死的片段；不在白名單裡就直接回錯誤，連查詢都不送。`IN` 的個數是動態的，做法是依個數產生 `?`，書名放進參數：

```go,ignore
{{#include ../../../examples/chapter11/02_sql_injection/after/loans.go:bytitles}}
```

同樣的記錄器（這次也記參數），看輸入與送出的 SQL：

```go,ignore
{{#include ../../../examples/chapter11/02_sql_injection/after/example_test.go:recorder}}
```

```go,ignore
{{#include ../../../examples/chapter11/02_sql_injection/after/example_test.go:example}}
```

攻擊字串現在只出現在參數那一行，查不到任何會員；SQL 的形狀不隨輸入改變。`Querier` 在 after 也是匯出的，理由是它讓測試能包住它、看見實際送出的 SQL 與參數；它同時是 `*sql.DB` 與 `*sql.Tx` 的共同介面，不是只為測試而存在。三個測試放在外部套件，分別鎖住三件事。同樣的輸入只被當成資料：

```go,ignore
{{#include ../../../examples/chapter11/02_sql_injection/after/loans_test.go:test}}
```

非法的排序被拒絕，而且沒有送出任何查詢：

```go,ignore
{{#include ../../../examples/chapter11/02_sql_injection/after/loans_test.go:orders}}
```

`IN` 的佔位符個數等於書名個數，書名不會出現在 SQL 裡：

```go,ignore
{{#include ../../../examples/chapter11/02_sql_injection/after/loans_test.go:in}}
```

把 `ByMember` 改回 `Sprintf`、拿掉白名單、或讓佔位符多一個，這些測試都會失敗。完整程式碼在 `examples/chapter11/02_sql_injection/after/`。

## 取捨

白名單與產生佔位符不是白拿的：

- **白名單要維護。** 新增一種排序要改 `orderClauses`，也要補測試；鍵是對外的名稱，不是資料庫的欄位名，欄位改名不必動 API，內部的欄位名也不會外洩。動態條件越多（多欄位排序、可選的篩選），組出來的 SQL 越多變化，每個組合都要有白名單與測試。
- **`IN` 的個數有上限。** 參數的上限與分批，第 4 章〈[N+1 與查詢組裝](../chapter4/05_n_plus_one.md)〉談過；`LIKE` 的萬用字元 `%` 與 `_` 也不會被參數化處理，要照字面比對時得另外跳脫。
- **ORM 與 query builder 不是免死金牌。** 它們幫忙產生的 `WHERE` 值通常都走參數，但多半也提供收一段原始字串的 `Order`、`Where`、`Raw`。把輸入交給那些參數，就是換了寫法的 `Sprintf`。判斷標準沒變：輸入有沒有變成 SQL 的一部分。code review 看到 SQL 附近有 `fmt.Sprintf` 或字串相加，就停下來問「這一段從哪來」。
- **輸入驗證擋不住它。** 本章〈[輸入驗證](01_input_validation.md)〉談過的長度、格式、範圍檢查，能擋掉明顯不合理的輸入，但 `o'neil` 是合法的名字；驗證是另一道防線，不能代替參數化。
- **資料庫權限是另一道防線。** 查詢用的帳號只給它需要的資料表與權限（讀取用的服務就不給寫入與改結構），萬一哪裡漏了，傷害有限。SQLite 沒有帳號的概念，這集的測試無法展示；它補的是程式碼擋不住的那一層，不是不必參數化的理由。

before 的拼接在 SQL 完全由程式碼決定時沒有風險，例如只有常數的片段，或值來自程式自己的列舉。這時換成白名單或佔位符只是多一層保險；一旦有一部分來自請求，就不再安全，所以預設一律走參數。

## 重點整理

- 用 `Sprintf` 或字串相加把輸入放進 SQL，輸入就可能變成 SQL 的一部分：讀出別人的資料，也會讓名字裡有單引號的正常輸入壞掉。
- 值走 `?` 佔位符，輸入只是資料；佔位符只能代表值，不能代表欄位名、表名或排序方向。
- 不能參數化的部分用白名單：呼叫端給名稱，程式裡寫死的片段才拼進 SQL，不在白名單就拒絕。
- `IN` 的個數依輸入產生 `?`，值放進參數；空清單不查詢。
- ORM、query builder 的原始字串 API 一樣危險；輸入驗證與資料庫權限是另外兩道防線，不是替代品。
