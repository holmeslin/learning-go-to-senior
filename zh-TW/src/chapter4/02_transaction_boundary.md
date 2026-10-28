# 交易邊界放在哪裡

## 本集目標

一個用例要改好幾筆資料時，判斷交易該由誰開：不是每個 store 方法各開各的，而是由用例決定範圍、把同一個交易交給 store 方法；也能說出交易該多大。

## 情境

你在寫轉帳：從 `ann` 的帳戶扣 300 元，加到 `bob` 的帳戶。資料層有一個 `Store`，提供 `Debit` 與 `Credit`。每次餘額異動，除了改 `accounts` 的餘額，還要在 `entries` 記一筆帳本，兩件事得一起成功，所以每個方法都開交易。轉帳這個用例（`Service.Transfer`）依序呼叫這兩個方法。

`BeginTx`、`Commit`、`Rollback` 與 `defer tx.Rollback()` 的寫法，前作〈[交易](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter15/06_transaction.html)〉已經講過；`database/sql` 怎麼接 SQLite，前作〈[`sql.Open` 與 SQLite driver](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter15/01_open_sqlite.html)〉也講過。這集談的是：交易該開在哪一層。

範例用 SQLite 的 in-memory 資料庫。`:memory:` 每條連線各是一個獨立的資料庫，連線池開出第二條連線就看不到資料表，所以 DSN 用具名的共享快取（`file:…?mode=memory&cache=shared`），每個 `Example` 與測試各用一個名字。

## 常見寫法

每個 store 方法自己開交易，做完就提交。方法內部是完整的：改餘額與記帳本一起成功：

```go,ignore
{{#include ../../../examples/chapter4/02_transaction_boundary/before/bank.go:apply}}
```

`Debit` 以 `-amount`、`Credit` 以 `+amount` 呼叫它。用例照順序呼叫：

```go,ignore
{{#include ../../../examples/chapter4/02_transaction_boundary/before/bank.go:transfer}}
```

每個方法單獨看都沒有錯，用起來也直覺：誰呼叫都不必操心交易。完整程式碼在 `examples/chapter4/02_transaction_boundary/before/`。

## 問題在哪

把錢轉給一個不存在的帳戶 `nobody`，看看會剩下什麼。`newDB` 開一個放著 `ann=500`、`bob=100` 的資料庫，`readState` 讀出餘額與帳本，兩者都在同目錄的 `example_test.go`。`// Output:` 由 `go test` 逐字比對：

```go,ignore
{{#include ../../../examples/chapter4/02_transaction_boundary/before/example_test.go:example}}
```

`Transfer` 回傳了錯誤，呼叫端會以為「轉帳失敗，什麼都沒發生」，資料庫卻不是這樣：`ann` 少了 300 元，帳本記著這筆扣款，`bob` 一毛沒收到。`Debit` 的交易在 `Credit` 開始前就提交了，之後沒有東西可以收回它。第一步就失敗時沒有這個問題；只有第二步失敗才會留下一半。程式在兩步之間當掉，或別的連線在兩步之間讀資料，看到的也是這個少了 300 元的狀態。

原因是每個方法的交易只保護自己：「改餘額與記帳本」是一起的，「扣款與入帳」卻不是。知道這兩步要一起成功的是用例，交易卻是 store 方法開的，用例手上沒有可以決定的東西。

你可能想在 `Transfer` 外面再包一層交易，但行不通：`apply` 裡的 `s.db.BeginTx` 開的是另一個交易，不在你的交易裡，`database/sql` 也沒有巢狀交易。要讓用例決定，store 方法就不能自己開交易。

## 較好的做法

交易由用例開，store 的方法在「呼叫端給的東西」上執行 SQL。`*sql.DB` 與 `*sql.Tx` 都有 `ExecContext`，所以 `Store` 只宣告它用到的這一個方法，建構函式收這個小介面、回傳具體型別（第 3 章〈[在使用端定義介面](../chapter3/01_define_interface_at_consumer.md)〉與〈[接受介面、回傳 struct](../chapter3/02_accept_interfaces_return_structs.md)〉談過）：

```go,ignore
{{#include ../../../examples/chapter4/02_transaction_boundary/after/bank.go:store}}
```

`apply` 的 SQL 一個字都沒變，只是不再開交易，改用 `s.q` 執行：

```go,ignore
{{#include ../../../examples/chapter4/02_transaction_boundary/after/bank.go:apply}}
```

用例開交易，再把它交給 `Store`：

```go,ignore
{{#include ../../../examples/chapter4/02_transaction_boundary/after/bank.go:transfer}}
```

`NewStore(tx)` 讓 `Debit` 與 `Credit` 的 SQL 都在 `tx` 裡執行。任何一個 `return` 都會觸發 `defer tx.Rollback()`，成功才 `Commit`，`Commit` 的錯誤也要檢查，因為提交本身也可能失敗。同一組方法在交易外照樣能用：`NewStore(db)` 讓每個方法直接在連線池上執行（`TestStoreWorksWithoutTransaction`）。

同樣的轉帳，這次的 `Example`（`newDB` 與 `readState` 的作用同 before，放在 `bank_test.go`）：

```go,ignore
{{#include ../../../examples/chapter4/02_transaction_boundary/after/example_test.go:example}}
```

第一個和 before 是同一個呼叫，錯誤一樣，但餘額沒變、帳本是空的；第二個是成功的轉帳，兩筆帳本都寫入。`bank_test.go` 另有表格測試，涵蓋入帳失敗與扣款失敗（餘額不足，被資料表的 `CHECK (balance >= 0)` 擋下）兩種，失敗後餘額不變、帳本是空的。測試都用 `t.Parallel()` 與各自的資料庫，`go test -race` 通過。完整程式碼在 `examples/chapter4/02_transaction_boundary/after/`。

## 取捨

交易不是越大越安全：

- **交易越大，鎖得越久。** 交易開著的期間，它改過的資料不會放開，別的交易要改同一份資料就得等（PostgreSQL、MySQL 的 InnoDB 鎖的是被改到的列；SQLite 一次只允許一個寫入交易，整個資料庫都在排隊）。它也占著連線池裡的一條連線，直到 `Commit` 或 `Rollback`。`Transfer` 的交易只含兩個 `UPDATE` 與兩個 `INSERT`，很短；連線被占住的後果，本章〈連線池要設多大、設錯會怎樣〉會談。
- **交易內不要做網路呼叫，也不要等使用者。** 例如扣款之後，在同一個交易裡呼叫外部的風控服務，或等使用者按確認：對方慢一秒，這兩個帳戶就多鎖一秒，要動這兩個帳戶的請求都排在後面，連線一條條被占滿。需要的資料在 `BeginTx` 之前查好，通知之類的事放到 `Commit` 之後（通知失敗怎麼補救，是另一個問題）。交易自己的 deadline，本章〈查詢逾時該設在哪一層〉會談。
- **用例要自己寫開交易、回滾、提交。** 用例一多，這一段會重複，可以抽成一個 `inTx(ctx, db, func(*Store) error) error` 輔助函式，把三步收在一處，用例只寫中間的部分。

before 在這些情況下反而是對的：

- 用例只呼叫一個 store 方法：方法自己開的交易就是整個用例的範圍，沒有「兩步」可以只做一半。
- 兩步本來就可以各自成功：例如主要操作成功後，寫一筆盡力而為的稽核紀錄，失敗只該記 log，不該讓主要操作回滾。
- 兩步不在同一個資料庫：一步寫資料庫、一步呼叫別的服務，單一個資料庫交易包不住它們，得靠補償（撤銷前一步）之類的做法，本書不展開。

判斷的方法：問自己，這幾筆寫入「全部成功或全部不算」是不是業務規則？是，交易就由知道這條規則的用例開；不是，各自提交就好。

## 重點整理

- 交易邊界跟著用例走：知道哪幾步要一起成功的是用例，不是 store 方法。
- store 方法各開各的交易，每個方法單獨都對，組起來卻會只成功一半，也包不起來（`database/sql` 沒有巢狀交易）。
- `Store` 收 `*sql.DB` 與 `*sql.Tx` 都符合的小介面，同一組方法在交易內外都能用；用例 `BeginTx`、`defer tx.Rollback()`、成功才 `Commit` 並檢查錯誤。
- 交易越大鎖越久、占著連線；交易內不做網路呼叫、不等使用者。
- 用例只做一次寫入，或兩步可以各自成功時，store 方法自己提交就夠了。
