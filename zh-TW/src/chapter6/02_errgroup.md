# `errgroup`

## 本集目標

同時做好幾件事、任何一件失敗就整批失敗時，判斷該用 `errgroup`：第一個錯誤會取消其他工作，`Wait` 回傳的就是那個錯誤。也要知道它不做什麼：只給第一個錯誤、不 recover panic。

## 情境

一個同步服務啟動前要抓好幾個來源的資料，只要有一個抓不到就不用啟動了。抓取函式 `Fetch` 收一個 `context.Context` 與來源名稱，`ctx` 取消時會盡快回傳。`FetchAll` 要同時抓所有來源，回傳第一個錯誤。

`WaitGroup`、`wg.Go`、`sync.Mutex` 與 `context` 前作都教過：〈[`sync.WaitGroup` 與 `wg.Go`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter9/02_waitgroup.html)〉、〈[`sync.Mutex`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter9/04_mutex.html)〉、〈[`WithCancel`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter10/02_with_cancel.html)〉。`errgroup` 不在前作裡，這集教它的 `Go`、`Wait`、`WithContext`；並行上限 `SetLimit` 本章〈`errgroup` 的並行上限〉會談。

## 常見寫法

`WaitGroup` 負責等，mutex 保護存錯誤的變數，變數叫 `firstErr`：

```go,ignore
{{#include ../../../examples/chapter6/02_errgroup/before/fetch.go:fetchall}}
```

沒有 data race，每個來源也都抓了。完整程式碼在 `examples/chapter6/02_errgroup/before/`。

## 問題在哪

有兩個問題。第一，`firstErr` 其實是「最後一個」：每個失敗的 goroutine 都無條件覆蓋它。下面的測試讓 a 在 10 毫秒失敗、b 在 20 毫秒失敗（用 `synctest` 的假時鐘，不真的等，用法見第 5 章〈[測時間與並行：用 `testing/synctest`，還是自己注入時鐘](../chapter5/08_synctest_or_clock.md)〉），測試卻期待 `FetchAll` 回傳後失敗的 b。這是在鎖住錯誤的行為，好讓問題看得見：

```go,ignore
{{#include ../../../examples/chapter6/02_errgroup/before/fetch_test.go:overwrite}}
```

補一個 `if firstErr == nil` 可以修掉覆蓋，但「第一個」會變成「第一個搶到鎖的」，而且還有第二個問題：出錯之後，其他來源照樣抓到底。下面的 `Example` 用 channel 讓 b 先出錯，其他四個來源等 b 失敗後才完成：

```go,ignore
{{#include ../../../examples/chapter6/02_errgroup/before/example_test.go:example}}
```

整批已經注定失敗，另外四個來源的工作卻全做完了。`fetch` 明明看得懂 `ctx`，但沒人在出錯時取消它。要補上，得自己 `WithCancel`、在鎖裡呼叫 `cancel`、記得 `defer cancel()`，又是一堆容易寫漏的細節。

## 較好的做法

`errgroup.WithContext` 把這些都包好了：

```go,ignore
{{#include ../../../examples/chapter6/02_errgroup/after/fetch.go:fetchall}}
```

- `g.Go` 開一個 goroutine 跑函式，函式回傳 `error`。
- 第一個回傳非 nil 錯誤的函式，會記下它的錯誤並取消 `WithContext` 回傳的 `ctx`；之後的錯誤不會蓋掉它。
- `g.Wait` 等所有函式都結束，回傳那個第一個錯誤。

注意閉包裡用的是 `WithContext` 回傳的新 `ctx`，傳給 `fetch` 的才是會被取消的那一個。同一個 `Example` 的情境，這次其他來源只有被取消才會結束：

```go,ignore
{{#include ../../../examples/chapter6/02_errgroup/after/example_test.go:example}}
```

`Wait` 回傳 b 的錯誤，不是其他來源收到取消後回的 `context canceled`。測試把時間與先後也鎖住：

```go,ignore
{{#include ../../../examples/chapter6/02_errgroup/after/fetch_test.go:firsterror}}
```

a 在 10 毫秒失敗，`FetchAll` 就在 10 毫秒回傳：b 本來要在 20 毫秒失敗、c 要等一小時，都在 a 失敗時被取消了。把這個測試拿去測 before 的 `FetchAll`，輸出如下（省略了耗時；`took 1h0m0s` 是 `synctest` 的假時鐘，不是真的等了一小時）。

執行結果：

```text
--- FAIL: TestFetchAllReturnsFirstError
    fetch_test.go:50: FetchAll() = b failed, want a failed
    fetch_test.go:53: FetchAll() took 1h0m0s, want 10ms
    fetch_test.go:56: canceled sources = 0, want 2
FAIL
```

## 取捨

`errgroup` 只給第一個錯誤，其他來源因取消而回的錯誤都丟掉了。需要全部錯誤（例如要一次告訴使用者哪幾個來源壞了），用 `wg.Go` 加 `errors.Join`，每個 goroutine 把錯誤寫進自己的 slice 索引，不需要 mutex，順序也固定。什麼時候該合併、什麼時候只回第一個，第 2 章〈[多個錯誤要合併回傳，還是只回第一個](../chapter2/05_join_or_first_error.md)〉談過：

```go,ignore
{{#include ../../../examples/chapter6/02_errgroup/after/join.go:join}}
```

```go,ignore
{{#include ../../../examples/chapter6/02_errgroup/after/example_test.go:joinexample}}
```

幾件事要記得：

- 沒有錯誤要傳（函式不會失敗，只是要等大家做完），`wg.Go` 就夠，不必引入 `errgroup`。
- `Wait` 回傳時 `ctx` 也會被取消，之後不要再拿它做事；`Wait` 之後還要用的話，用外面傳進來的 `ctx`。
- 取消是合作式的：不看 `ctx` 的函式不會被打斷，`Wait` 還是會等它做完。怎麼讓 goroutine 看得懂取消，本章〈取消與 `context` 的傳遞〉會談。
- `errgroup` 不會 recover panic。`Go` 的原始碼註解寫明刻意不把 panic 傳回呼叫 `Wait` 的 goroutine，所以任何一個函式 panic，整個程式照樣結束。
- 零值 `errgroup.Group` 也能用，但不會在出錯時取消；要取消就用 `WithContext`。一個 `Group` 只用在一批工作，不要重複使用。

before 並不是一無是處：單純要「等全部做完」、或像 `FetchAllJoin` 這樣要保留全部錯誤時，`WaitGroup` 本來就是對的工具。

## 重點整理

- 手刻「第一個錯誤」常見兩個 bug：後來的錯誤蓋掉先到的，以及出錯後其他 goroutine 照跑到底。
- `errgroup.WithContext` 的第一個錯誤會取消共用的 `ctx`，`Wait` 等所有 goroutine 結束後回傳那個錯誤。
- 傳給 `fetch` 的要是 `WithContext` 回傳的 `ctx`，而且函式要看 `ctx.Done()`，取消才有用。
- 只要第一個錯誤用 `errgroup`；要全部錯誤用 `wg.Go` 加 `errors.Join`；沒有錯誤要傳用 `wg.Go`。
- `errgroup` 不 recover panic，也不限制並行數量（本章〈`errgroup` 的並行上限〉會談）。
