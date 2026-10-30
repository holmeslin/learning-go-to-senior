# 日誌層級與欄位設計

## 本集目標

寫日誌時判斷四件事：錯誤在哪一層記、層級怎麼選、鍵名誰說了算、層級怎麼在不重新部署的情況下調整。

## 情境

一個付款服務：`POST /invoices/{id}/pay` 依發票扣款，分成 store、service、handler 三層；另有一個批次匯入發票的函式。資料庫連不上時，值班的人要從日誌看出「哪個使用者、哪張發票、為什麼失敗」。

`log/slog` 的基本用法前作〈[`log/slog`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter12/10_slog.html)〉講過；handler 與欄位的寫法，本章〈[結構化日誌：`log/slog` 的欄位與 handler 怎麼選](01_structured_logging.md)〉談過。這集談的是判斷，不是 API。

## 常見寫法

每一層看到錯誤就記一筆，鍵名各寫各的：

```go,ignore
{{#include ../../../examples/chapter9/02_log_levels_and_fields/before/billing.go:store}}
```

```go,ignore
{{#include ../../../examples/chapter9/02_log_levels_and_fields/before/billing.go:service}}
```

```go,ignore
{{#include ../../../examples/chapter9/02_log_levels_and_fields/before/billing.go:handler}}
```

批次匯入則是每一筆都記一行 Info：

```go,ignore
{{#include ../../../examples/chapter9/02_log_levels_and_fields/before/billing.go:import}}
```

## 問題在哪

資料庫連不上時送一個請求，再匯入四筆資料（其中一筆是壞的）。`Example` 的 `// Output:` 就是日誌的實際輸出：

```go,ignore
{{#include ../../../examples/chapter9/02_log_levels_and_fields/before/example_test.go:request}}
```

```go,ignore
{{#include ../../../examples/chapter9/02_log_levels_and_fields/before/example_test.go:importlog}}
```

- 一個失敗三行 Error。兩個請求失敗就是六行，看日誌的人得自己數。這是第 2 章〈[在哪一層處理、哪一層只回傳](../chapter2/03_handle_at_boundary.md)〉談過的問題，換成結構化日誌也一樣。
- 使用者代號有三種寫法：`userId`、`user_id`、`uid`。查 `user_id=u-42` 只找得到其中一行，要把三種都列進查詢，換一個服務又是另一套。
- 匯入一萬筆就是一萬行 Info，真正值得看的那一行（有一筆資料列壞了）埋在裡面。

## 較好的做法

**錯誤只在處理的那一層記。** store 與 service 只加上下文就回傳，邊界（這裡是 handler）記一次 Error。批次函式自己是邊界，所以它記摘要。

**層級依「誰需要採取行動」選：**

- Error：需要有人處理。請求因為本服務或下游失敗了。
- Warn：服務自己撐過去了，但值得有人看一眼，例如匯入跳過了壞掉的資料列。
- Info：業務事件，一個請求或一個批次一行，不是一筆資料一行。
- Debug：除錯用的細節，例如每一筆資料列，預設不開。

**鍵名集中定義**成常數，耗時的單位固定在鍵名裡：

```go,ignore
{{#include ../../../examples/chapter9/02_log_levels_and_fields/after/keys.go:keys}}
```

**請求範圍的欄位用 `With` 帶入。** handler 建一次帶著請求代號、使用者代號、發票代號的 logger，往下傳；下層只寫自己的訊息，欄位自動跟著：

```go,ignore
{{#include ../../../examples/chapter9/02_log_levels_and_fields/after/billing.go:handler}}
```

```go,ignore
{{#include ../../../examples/chapter9/02_log_levels_and_fields/after/billing.go:service}}
```

批次函式每一筆只記 Debug，結束時記一行摘要，有跳過就升成 Warn：

```go,ignore
{{#include ../../../examples/chapter9/02_log_levels_and_fields/after/billing.go:import}}
```

同樣的失敗請求與匯入，這次的輸出（`Example` 用 `ReplaceAttr` 拿掉每次都不同的 `time` 與耗時）：

```go,ignore
{{#include ../../../examples/chapter9/02_log_levels_and_fields/after/example_test.go:drop}}
```

```go,ignore
{{#include ../../../examples/chapter9/02_log_levels_and_fields/after/example_test.go:request}}
```

```go,ignore
{{#include ../../../examples/chapter9/02_log_levels_and_fields/after/example_test.go:importlog}}
```

**`slog.LevelVar` 讓層級在執行期間可以調整。** `LevelVar` 的零值是 Info，傳給 `HandlerOptions.Level`，之後 `Set` 一次，下一筆日誌就生效。下面第一個請求沒有 Debug，切換之後第二個請求才有，而且 Debug 那行沒有自己寫欄位，請求代號是 `With` 帶進來的：

```go,ignore
{{#include ../../../examples/chapter9/02_log_levels_and_fields/after/example_test.go:levelvar}}
```

測試用 `JSONHandler` 解析輸出，斷言每個失敗的請求恰好一筆 Error：

```go,ignore
{{#include ../../../examples/chapter9/02_log_levels_and_fields/after/billing_test.go:test}}
```

完整程式碼在 `examples/chapter9/02_log_levels_and_fields/after/`，還有 `LevelVar` 切換、鍵名集合與匯入層級的測試。`Set` 由什麼觸發（訊號、設定檔、管理用的端點）由你決定；端點的安全問題，本章〈`/debug/pprof` 與診斷端點〉會談。

## 取捨

- **鍵名約定要有人維護。** 常數放進共用套件，每個服務都得依賴它；改一個鍵名，下游的查詢、儀表板與告警規則一起壞。`slog` 不擋你寫 `"uid"`，`go vet` 只檢查鍵值是否成對。要靠工具強制，可以用 `sloglint` 這類 linter。單位寫進鍵名（`duration_ms`）很囉唆，但比查日誌時猜單位便宜。
- **Debug 不是免費的。** 關著的時候，被過濾的日誌不會輸出，但呼叫端傳進去的參數照樣會算；參數本身很貴時（序列化大物件），要先用 `logger.Enabled` 判斷。打開之後日誌量與儲存成本跟著上升，所以 `LevelVar` 是拿來暫時打開、排查完調回去的。它調的是整個 logger，不是單一請求；想只看某個使用者，要另外在 handler 做篩選。
- **日誌太多時才考慮取樣：** 同一種重複的日誌只留一部分。取樣會丟資訊，Error 通常不取樣；`slog` 沒有內建取樣，要在 handler 或收集端做。這集只談概念。
- **before 在小地方不一定錯。** 單一函式的小工具、沒有呼叫端的背景 worker，就地記錄是對的。需要在中間層記，是它自己把錯誤吞掉的時候（重試成功、改走備援），這時它就是處理的那一層。
- 日誌欄位與錯誤訊息裡可能帶著不該出現的資料，第 11 章〈日誌裡的敏感資料〉會談。

## 重點整理

- 錯誤只在處理的那一層記：中間層加上下文就回傳，邊界記一次。
- 層級看誰需要採取行動：Error 要人處理，Warn 自己恢復但值得看，Info 是業務事件，Debug 是預設不開的細節。
- 迴圈裡每筆記 Debug，結束時用一行摘要回報筆數，有異常就升成 Warn。
- 鍵名與單位集中定義成常數；請求範圍的欄位用 `With` 帶入，下層不用知道。
- `slog.LevelVar` 讓層級執行期間可調，是暫時工具，Debug 的量與成本要自己負責。
