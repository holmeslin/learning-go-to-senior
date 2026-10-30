# 逾時預算

## 本集目標

一條呼叫鏈有好幾層、每層都要設逾時時，判斷逾時怎麼分配：由最外層定下總預算、沿著 `ctx` 往內傳，每一步扣掉後續需要的時間再交給下游，時間不夠就提早回錯誤，而不是每一層各設一個固定值。

## 情境

運費試算 API。呼叫端（結帳頁的後端）最多等 3 秒。handler 先查庫存、再查運費，兩個都是下游。每一層的作者都設了「合理」的逾時：handler 5 秒，每個下游 client 5 秒。

`WithTimeout` 與 `WithDeadline` 前作教過：〈[`WithTimeout` 與 `WithDeadline`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter10/03_with_timeout.html)〉。單一查詢的逾時該由哪一層設，第 4 章〈[查詢逾時該設在哪一層](../chapter4/03_query_timeout_layer.md)〉談過；這集談的是好幾層疊在一起時，時間怎麼分。重試時剩下的時間怎麼用，本章〈重試、退避與 jitter〉會談。

## 常見寫法

每個下游的 client 帶著自己的逾時，handler 在請求的 `ctx` 上再加一個：

```go,ignore
{{#include ../../../examples/chapter10/01_timeout_budget/before/budget.go:layers}}
```

`Client` 不收呼叫端的 `ctx`，逾時是它自己的固定值，就像 `http.Client` 的 `Timeout` 搭配沒帶 context 的 request。handler 的想法和 `http.TimeoutHandler`（前作〈[逾時設定](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter14/07_timeouts.html)〉）一樣：`ctx` 結束就先回 504，不再等 `Service`。每一層單看都有逾時，完整程式碼在 `examples/chapter10/01_timeout_budget/before/`。

## 問題在哪

兩個下游各要 4 秒，都在自己的 5 秒之內，沒有一層會逾時。呼叫端 3 秒後放棄：

```go,ignore
{{#include ../../../examples/chapter10/01_timeout_budget/before/budget_test.go:test}}
```

時間用 `testing/synctest`（用法見第 5 章〈[測時間與並行：用 `testing/synctest`，還是自己注入時鐘](../chapter5/08_synctest_or_clock.md)〉）。bubble 裡的時間是假的，經過多久是確定的，所以可以斷言精確的 3 秒與 8 秒。`recorder` 與 `backend` 把每次下游呼叫的開始、結束與當時的 deadline 記下來：

```go,ignore
{{#include ../../../examples/chapter10/01_timeout_budget/before/helpers_test.go:helpers}}
```

handler 在第 3 秒回了 504，下層卻沒有停：庫存做到第 4 秒才結束，運費在第 4 秒才開始，那時呼叫端已經走了 1 秒，運費一路做到第 8 秒。每個下游拿到的都是新的 5 秒（`deadline in 5s`），沒有人知道呼叫端只剩多少時間。逾時的上限加起來是 10 秒，呼叫端只等 3 秒；這 5 秒是白做的，下游仍在替沒有人要的結果工作。

## 較好的做法

預算只在最外層定一次：handler 在請求的 `ctx` 上加 `Budget`，與呼叫端已經帶來的 deadline 取較早的那個，之後整條鏈只傳這個 `ctx`：

```go,ignore
{{#include ../../../examples/chapter10/01_timeout_budget/after/budget.go:handler}}
```

每一步用 `within` 從剩下的時間裡扣掉自己後續需要的，再交給下游。剩下的不到最低需求，就不呼叫，直接回 `ErrNoBudget`（匯出它，是為了讓呼叫 `Service` 的程式能用 `errors.Is` 分辨「沒時間」與下游自己的逾時）：

```go,ignore
{{#include ../../../examples/chapter10/01_timeout_budget/after/budget.go:within}}
```

`WithDeadline` 的結果不會晚於父 `ctx`，所以往前挪只會更緊。`Service` 決定每一步要留多少：庫存之後還要留運費與回應的時間，運費只需要留回應的。這幾個常數是範例的假設，實際的值來自下游延遲的量測（延遲的分布怎麼收，見第 9 章〈[SLI 與 SLO](../chapter9/08_sli_slo.md)〉）：

```go,ignore
{{#include ../../../examples/chapter10/01_timeout_budget/after/budget.go:service}}
```

測試的兩個 helper：`serve` 發請求並量 handler 花的假時間，`newHandler` 組出預算 3 秒的 handler；`recorder` 與 before 的相同。庫存 500 毫秒、運費 1 秒時，每一步看到的 deadline 是扣完後面需求的剩餘（庫存 3 − 1 − 0.2 = 1.8 秒，運費 3 − 0.5 − 0.2 = 2.3 秒）：

```go,ignore
{{#include ../../../examples/chapter10/01_timeout_budget/after/budget_test.go:serve}}
```

```go,ignore
{{#include ../../../examples/chapter10/01_timeout_budget/after/budget_test.go:within}}
```

同樣兩個各要 4 秒的下游，這次庫存在 1.8 秒被中止，運費根本沒被呼叫，handler 在預算內回 504：

```go,ignore
{{#include ../../../examples/chapter10/01_timeout_budget/after/budget_test.go:slow}}
```

呼叫端自己只剩 1 秒時，庫存之後要留的 1.2 秒已經不夠，連庫存也不必呼叫，錯誤在 0 秒就回來：

```go,ignore
{{#include ../../../examples/chapter10/01_timeout_budget/after/budget_test.go:nobudget}}
```

另一個測試 `TestCallerLeavingStopsTheChain` 讓呼叫端 1 秒後斷線：庫存在第 1 秒就被中止，運費沒有被呼叫，也沒有對已離開的呼叫端寫任何回應。

## 取捨

**預算太緊會誤殺。** 庫存只拿到 1.8 秒，平常偶爾要 2 秒的話，本來會成功的請求就被殺掉。`need` 與 `reserve` 太保守，不如不分配；太寬鬆，又回到 before。

**deadline 要有人設，下游要肯看。** `within` 遇到沒有 deadline 的 `ctx` 原樣放行，所以每個邊界（handler、背景工作）都要自己設。`net/http` 不會把呼叫端的 deadline 傳進來，`Budget` 要和呼叫端約定（並扣掉網路往返）；呼叫端斷線時 `r.Context()` 會取消。deadline 只是 `ctx` 上的值，下游不看就沒用：HTTP 請求要用 `NewRequestWithContext`，它與 `Client.Timeout` 同時存在時先到的生效（前作〈[HTTP client](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter14/06_http_client.html)〉）。

**背景工作要自己設逾時。** `context.WithoutCancel` 拿掉的不只取消，還有 deadline（第 9 章〈[在 `context` 裡傳遞 trace](../chapter9/05_context_propagation.md)〉談過）。請求結束後還要做完的工作，寫成 `context.WithTimeout(context.WithoutCancel(ctx), limit)`：上限是這件工作自己的，不是請求剩下的時間。

**和 HTTP server 的逾時是兩回事。** `ReadHeaderTimeout`、`WriteTimeout` 這些（前作〈[逾時設定](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter14/07_timeouts.html)〉）保護的是連線，不讓慢的呼叫端佔住伺服器；預算保護的是下游，不讓慢的下游拖住整條鏈。兩者不能互相取代，但有順序：`WriteTimeout` 從讀完標頭起算，`Budget` 要比它短，否則 504 來不及寫出去。

**before 什麼時候夠用？** 只有一層、一個下游時，固定逾時就是預算，不需要分配。離線的批次工作沒有人在等，也不必扣呼叫端的時間。

## 重點整理

- 每一層各設固定逾時，下層不知道呼叫端還剩多少時間：每個都沒逾時，加起來卻超出預算，呼叫端走了之後下層還在做。
- 預算在最外層定一次，沿著 `ctx` 往內傳；每一步扣掉後續需要的時間，剩下的不夠就回 `ErrNoBudget`，不呼叫下游。
- 前面的步驟再慢也吃不掉後面的預算；呼叫端放棄時整條鏈一起停。
- deadline 要有人設、下游要看 `ctx`；`WithoutCancel` 之後的背景工作要自己包逾時。
- 測試用 `synctest` 的假時間，可以斷言精確的經過時間與每個下游拿到的 deadline，不需要 `time.Sleep`。
