# 查詢逾時該設在哪一層

## 本集目標

要替查詢設逾時時，判斷該由誰設：資料層自己用 `context.Background()` 加一個固定值，還是由邊界（handler、背景工作）設 deadline、沿著 `ctx` 往下傳；也能說出資料層什麼時候才需要自己縮短。

## 情境

你有一個訂單服務，後台儀表板每次載入都呼叫 `GET /pending`，取得待處理的訂單數。資料庫偶爾會忙，查詢會卡住。你要兩件事：卡住的查詢不能無限等下去；使用者關掉頁面之後，沒人要的查詢也別繼續占著資料庫。

`QueryRowContext` 這族方法與 context 的逾時，前作〈[context 與連線池](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter15/08_context_pool.html)〉與〈[`WithTimeout` 與 `WithDeadline`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter10/03_with_timeout.html)〉已經講過，這集只談 deadline 該由哪一層設。連線池該設多大，本章〈連線池要設多大、設錯會怎樣〉會談；範例用「一個沒結束的交易占住唯一的連線」讓資料庫確定性地忙碌，只是個手段，不依賴查詢跑多久。

## 常見寫法

為了不讓查詢無限等，store 的每個方法自己設一個固定的逾時：

```go,ignore
{{#include ../../../examples/chapter4/03_query_timeout_layer/before/store.go:store}}
```

handler 沒有做錯什麼，它老實地把請求的 context 交了下去：

```go,ignore
{{#include ../../../examples/chapter4/03_query_timeout_layer/before/handler.go:handler}}
```

完整程式碼在 `examples/chapter4/03_query_timeout_layer/before/`。

## 問題在哪

請求的 `ctx` 進了 `PendingCount`，第一行就被 `context.Background()` 蓋掉：`ctx, cancel :=` 在函式最外層重新指定了參數，編譯器與 `go vet` 都不會抱怨。呼叫端的取消與 deadline，查詢一概不知道。下面的 `// Output:` 是實際輸出（由 `go test` 逐字比對；為了讓輸出固定，先取消、再呼叫）：

```go,ignore
{{#include ../../../examples/chapter4/03_query_timeout_layer/before/example_test.go:example}}
```

- 第一行：`ctx` 已經取消，查詢照跑，還回傳 3。使用者早就走了，這次查詢是白做的。
- 第二行：資料庫忙碌時，取消的 `ctx` 一樣不起作用，查詢撐到 store 自己的上限才放棄。錯誤是 store 自己的 `context deadline exceeded`，不是呼叫端的 `context canceled`，handler 分不出「使用者走了」和「資料庫真的太慢」。
- 上限是固定的。範例為了跑得快設 100 毫秒，實際服務可能是 5 秒：請求只剩 1 秒預算時，查詢仍然可以等 5 秒；呼叫端想等得更久，也沒有辦法。多次查詢的上限還會一個接一個疊上去，這類預算的算法，第 10 章〈[逾時預算](../chapter10/01_timeout_budget.md)〉談過。

更常見的版本連這個固定逾時都沒有：用 `db.QueryRow`，文件寫明它內部用 `context.Background`，永遠不會取消。

## 較好的做法

store 只用收到的 `ctx`，自己不設任何逾時：

```go,ignore
{{#include ../../../examples/chapter4/03_query_timeout_layer/after/store.go:store}}
```

deadline 由邊界決定。handler 在 `r.Context()` 上加 deadline，一起往下傳；`r.Context()` 在使用者斷線時就會取消。錯誤也在這裡處理，一次就好（第 2 章〈[在哪一層處理、哪一層只回傳](../chapter2/03_handle_at_boundary.md)〉）：

```go,ignore
{{#include ../../../examples/chapter4/03_query_timeout_layer/after/handler.go:handler}}
```

三種結果各有各的處理：使用者已經走了，不必回應；deadline 到了回 504；其他錯誤記錄下來回 500。能這樣分，是因為錯誤保留了 `context.Canceled` 與 `context.DeadlineExceeded` 的差別；before 的 handler 只看得到 store 自己的逾時。同樣的兩個情況，這次的 `// Output:`：

```go,ignore
{{#include ../../../examples/chapter4/03_query_timeout_layer/after/example_test.go:example}}
```

取消的 `ctx` 讓查詢立刻以 `context canceled` 結束；呼叫端的 deadline 一到，查詢就放棄，不必等任何固定上限。測試放在外部套件，用表格把這兩種情況與預期錯誤寫在一起：

```go,ignore
{{#include ../../../examples/chapter4/03_query_timeout_layer/after/store_test.go:test}}
```

資料層偶爾需要比呼叫端更緊的上限，例如備註欄沒有索引、會掃過整張表的搜尋。這時在收到的 `ctx` 上縮短，而不是另起 `Background`：

```go,ignore
{{#include ../../../examples/chapter4/03_query_timeout_layer/after/store.go:search}}
```

`WithTimeout` 取較早的那個：呼叫端的 deadline 更早就聽呼叫端的，呼叫端取消，縮短過的 context 也跟著取消（文件寫明 `WithDeadline` 的結果不會晚於父 context）。`store_test.go` 裡針對 `SearchCount` 的三個測試分別驗證：呼叫端沒有 deadline 時 store 的上限先到、呼叫端的 deadline 更早時不會被延長、取消照樣傳得進去。

交易也沒有自己的逾時。本章〈[交易邊界放在哪裡](02_transaction_boundary.md)〉的用例用收到的 `ctx` 呼叫 `BeginTx`；文件寫明這個 context 管到交易結束，取消時 `database/sql` 會回滾交易（`TestBeginTxCancelRollsBack`：取消之後，交易裡的寫入沒有留下來，`Commit` 回傳 `sql.ErrTxDone`）。所以邊界的 deadline 同時是整個交易的上限，不必另外設。要讓某個交易更短，就像 `SearchCount` 那樣在 `ctx` 上縮短，再交給 `BeginTx`。

完整程式碼在 `examples/chapter4/03_query_timeout_layer/after/`。

## 取捨

背景工作沒有請求可以繼承，沒有人替它設 deadline，只能自己設。它就是邊界，所以從整個程式的 context（關機時取消）往下加：

```go,ignore
{{#include ../../../examples/chapter4/03_query_timeout_layer/after/job.go:job}}
```

代價與限制：

- 每個 store 方法都多一個 `ctx` 參數，每個呼叫端都得交一個；既有的程式要一路改上去。
- store 不設預設上限後，邊界忘了設 deadline，查詢就沒有任何上限。每個邊界都該檢查：handler 有沒有設、背景工作有沒有設。
- 縮短只在特定查詢確實需要時才加；每個方法都縮短，就是把固定逾時換個樣子搬回來。

before 在這些情況下反而可以接受：

- 一次性的腳本或命令列工具，沒有人會取消、也沒有請求，固定逾時加上 `Background` 最直接。
- 有些查詢就是要跑完，即使呼叫端離開了，例如請求結束前要寫下的稽核紀錄。這是刻意的決定，該用 `context.WithoutCancel(ctx)`，保留 `ctx` 帶的值、只拿掉取消，前作〈[context 傳遞慣例](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter10/07_context_conventions.html)〉講過；不是悄悄把 `ctx` 蓋掉。

## 重點整理

- store 在 `context.Background()` 上自己設逾時，呼叫端的取消與 deadline 就傳不進查詢：查詢照跑，錯誤也分不出是誰放棄。
- deadline 由邊界設：handler 在 `r.Context()` 上加，背景工作從程式的 context 往下加；store 只用收到的 `ctx`。
- 資料層需要更緊的上限時，在收到的 `ctx` 上縮短；`WithTimeout` 只會更早，不會延長，取消也照樣傳得進去。
- 背景工作沒有請求可以繼承，要自己設 deadline；跨服務的逾時預算，第 10 章〈[逾時預算](../chapter10/01_timeout_budget.md)〉談過。
- 查詢要在呼叫端離開後仍然跑完，用 `context.WithoutCancel` 明說，不要靠蓋掉 `ctx`。
