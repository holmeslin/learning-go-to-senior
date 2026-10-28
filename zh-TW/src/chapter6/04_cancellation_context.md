# 取消與 `context` 的傳遞

## 本集目標

寫會開 goroutine 的函式時，判斷 `ctx` 要傳到哪裡：迴圈的每一輪、每個會阻塞的地方，都要有人看它；也知道取消是合作式的，CPU 密集的迴圈得自己檢查，檢查本身有成本。

## 情境

連結預覽服務：使用者貼上一批網址，`Pages` 依序抓取，每抓完一筆就送進 channel，讓呼叫端邊收邊顯示。使用者關掉頁面時，請求的 `ctx` 會取消。`context` 與 `select` 前作教過：〈[為什麼需要 context](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter10/01_why_context.html)〉、〈[`WithCancel`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter10/02_with_cancel.html)〉、〈[`select`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter9/11_select.html)〉；這集只談取消怎麼傳進 goroutine。

## 常見寫法

函式照慣例把 `ctx` 放在第一個參數，也傳給 `fetch`：

```go,ignore
{{#include ../../../examples/chapter6/04_cancellation_context/before/pages.go:pages}}
```

完整程式碼在 `examples/chapter6/04_cancellation_context/before/`。簽章符合〈[context 傳遞慣例](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter10/07_context_conventions.html)〉，編譯器與 `go vet` 都沒有意見，但 goroutine 裡沒有任何一行在看 `ctx.Done()`。

## 問題在哪

取消之後，剩下的工作照做。下面的 `Example` 讓 `fetch` 在處理第二筆時呼叫 `cancel`，模擬使用者在抓取途中離開；呼叫端把 channel 讀到關閉為止，數 `fetch` 被呼叫了幾次。計數只由抓取的 goroutine 寫入，channel 關閉之後才讀，所以沒有 data race，輸出也不依賴排程：

```go,ignore
{{#include ../../../examples/chapter6/04_cancellation_context/before/example_test.go:example}}
```

五筆全抓完，後三筆沒有人要。傳給 `fetch` 的 `ctx` 也不會讓迴圈停下來：就算 `fetch` 一看到取消就立刻返回，迴圈還是會用同一個已取消的 `ctx` 呼叫剩下的每一筆。

每一筆都是一次沒人要的下游請求：名單愈長、使用者離開得愈早，白做的就愈多。

另一個問題在送出那一行。`out` 沒有緩衝，呼叫端取消後若不再接收，`out <- fetch(ctx, u)` 永遠等不到人，goroutine 就卡在那裡。怎麼在測試裡抓到這種洩漏，本章〈goroutine 洩漏怎麼發生、怎麼在測試裡抓到〉會談。

## 較好的做法

每個會阻塞的地方都要能被 `ctx` 叫醒。每一輪開始前看一眼 `ctx`，送出時和 `ctx.Done()` 一起 `select`：

```go,ignore
{{#include ../../../examples/chapter6/04_cancellation_context/after/pages.go:pages}}
```

同樣的 `Example`，這次取消之後只抓到第二筆：

```go,ignore
{{#include ../../../examples/chapter6/04_cancellation_context/after/example_test.go:example}}
```

第二筆正在處理時發生的取消，無法收回已經開始的那一筆；但第三筆不會再開始。取消後不再接收的情況，用 `synctest` 確定地證明，不必等任何時間：第一筆收下後取消，`synctest.Wait` 等到 goroutine 都結束或卡住，再檢查 channel 已經關閉。若送出那一行少了 `select`，goroutine 會卡在送出，bubble 結束時它還沒結束，`synctest.Test` 會讓測試失敗。測試的基本規則見第 5 章〈[測時間與並行：用 `testing/synctest`，還是自己注入時鐘](../chapter5/08_synctest_or_clock.md)〉：

```go,ignore
{{#include ../../../examples/chapter6/04_cancellation_context/after/pages_test.go:abandoned}}
```

## 取捨

取消是合作式的。Go 沒有從外面停止 goroutine 的辦法，`cancel()` 只是把 `Done` 關起來，要不要停，由 goroutine 自己決定。所以 `fetch` 若是不看 `ctx` 的阻塞呼叫，取消得等它返回；對外的呼叫要把 `ctx` 傳進去。

CPU 密集的迴圈不會阻塞，也就沒有 `select` 可以放，只能自己定期看 `ctx.Err()`：

```go,ignore
{{#include ../../../examples/chapter6/04_cancellation_context/after/pages.go:sum}}
```

檢查有成本。一次檢查是沿著 `context` 鏈呼叫 `Err`，最底層的 `cancelCtx` 是一次 atomic 讀取，單次很便宜，但每個元素都檢查，次數就跟元素數一樣多。每 `CheckEvery` 個元素檢查一次，換來的是取消的延遲：最多多算 `CheckEvery-1` 個元素。測試用一個數 `Err` 呼叫次數的 `context` 讓這件事可以驗證：一萬個元素只檢查 10 次。粒度要多大，用 `b.Loop` 量過再決定，量測的方法第 7 章〈[`b.Loop` 與 `benchstat`：怎麼比較前後結果才可信](../chapter7/02_b_loop_benchstat.md)〉談過。

```go,ignore
{{#include ../../../examples/chapter6/04_cancellation_context/after/pages_test.go:cpu}}
```

迴圈每一輪本來就要等 I/O 或做很重的事時，每輪檢查一次就夠，成本可以忽略；before 在「做不做完都無所謂」的短工作上也不算錯，例如要抓的只有兩三筆、每筆毫秒內完成。

deadline 不是這裡設的：函式只收 `ctx`，不自己加逾時。逾時由邊界（handler、背景工作）設定、往下傳，見第 4 章〈[查詢逾時該設在哪一層](../chapter4/03_query_timeout_layer.md)〉。

## 重點整理

- 簽章收了 `ctx` 不等於有人在看；goroutine 裡沒有 `ctx.Done()` 或 `ctx.Err()`，取消就沒有效果。
- 迴圈每一輪開始前檢查 `ctx`；每個會阻塞的 channel 操作都和 `ctx.Done()` 一起 `select`，送出也不例外。
- 取消是合作式的：外部呼叫要傳 `ctx`，CPU 密集的迴圈要自己定期呼叫 `ctx.Err()`。
- 檢查的間隔是取捨：愈密成本愈高，愈疏取消後多做的工作愈多。
- 函式只傳遞 `ctx`，deadline 由邊界設定；取消是否有效，用 `synctest` 或 channel 訊號就能確定地測，不需要 `time.Sleep`。
