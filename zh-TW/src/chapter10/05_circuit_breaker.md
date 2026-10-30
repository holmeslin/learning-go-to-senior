# 斷路器

## 本集目標

下游已經掛了，判斷什麼時候該讓請求不再等它：斷路器（circuit breaker）的三個狀態、怎麼決定打開與恢復的時機，以及它和重試、降級怎麼配合。

## 情境

運費報價服務，`GET /quote` 把請求轉給承運商的 API，每次呼叫最多等 3 秒（這個逾時怎麼從呼叫端的 deadline 分配，本章〈逾時預算〉談過）。某天承運商開始完全不回應。

`WithTimeout`、HTTP 逾時設定與 `httptest` 前作教過：〈[`WithTimeout` 與 `WithDeadline`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter10/03_with_timeout.html)〉、〈[逾時設定](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter14/07_timeouts.html)〉、〈[`httptest`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter14/09_httptest.html)〉。這集的測試用 `testing/synctest` 的假時間，用法見第 5 章〈[測時間與並行：用 `testing/synctest`，還是自己注入時鐘](../chapter5/08_synctest_or_clock.md)〉。

## 常見寫法

有設逾時，錯誤也有對應的狀態碼，每個請求都照樣呼叫下游：

```go,ignore
{{#include ../../../examples/chapter10/05_circuit_breaker/before/quote.go:handler}}
```

承運商正常，或只是偶爾慢，這個寫法沒有任何毛病。完整程式碼在 `examples/chapter10/05_circuit_breaker/before/`。

## 問題在哪

讓承運商完全不回應，連續送 10 個請求：

```go,ignore
{{#include ../../../examples/chapter10/05_circuit_breaker/before/quote_test.go:test}}
```

每個請求都等滿 3 秒才收到 504，10 個請求花了 30 秒，下游被呼叫 10 次，沒有一次有機會成功。`synctest` 的假時間是確定的，所以這裡可以直接斷言 30 秒。請求在這 3 秒裡佔著 handler 的 goroutine 與對承運商的連線；請求進來的速度比逾時快，等待中的請求就一直累積。逾時只保護單一請求，它不記得上一個請求剛剛才失敗。

## 較好的做法

讓斷路器記住下游的狀況，有三個狀態：

- **關閉**：請求正常通過，記錄連續失敗的次數。
- **打開**：連續失敗到達門檻之後，請求立刻被擋下，不呼叫下游。
- **半開**：冷卻時間過了，放一個試探請求過去；成功就關上，失敗就重新打開。

```go,ignore
{{#include ../../../examples/chapter10/05_circuit_breaker/after/breaker/breaker.go:types}}
```

`State` 與 `State()` 是為了測試才匯出的：測試要斷言狀態轉換；正式環境也會把狀態放進指標與日誌。冷卻用注入的 `Now` 量，測試不用等。

```go,ignore
{{#include ../../../examples/chapter10/05_circuit_breaker/after/breaker/breaker.go:do}}
```

半開只會由試探請求進入、由它結束離開，所以半開的時候，途中一定只有一個請求，其餘的直接回 `ErrOpen`。`admit` 回傳的 `probe` 讓 `record` 分得出試探與一般請求：斷路器打開之前就出發的請求，晚到的失敗不重新計算冷卻。`fn` 如果 panic，`ok` 還是 `false`，視為失敗，斷路器不會卡在半開。

測試用的時鐘只在 `Advance` 時前進：

```go,ignore
{{#include ../../../examples/chapter10/05_circuit_breaker/after/breaker/breaker_test.go:helpers}}
```

狀態轉換一步一步斷言：成功會讓計數歸零、打開時不呼叫下游、冷卻差一秒還是擋下、試探失敗重新計冷卻、試探成功才關上：

```go,ignore
{{#include ../../../examples/chapter10/05_circuit_breaker/after/breaker/breaker_test.go:transitions}}
```

半開只放一個試探，要有並行的測試。試探卡在途中時，其餘 5 個請求都不能碰到下游（把 `admit` 的半開分支改成放行，這個測試就會失敗）：

```go,ignore
{{#include ../../../examples/chapter10/05_circuit_breaker/after/breaker/breaker_test.go:probe}}
```

整段過程用 `Example` 看一次：

```go,ignore
{{#include ../../../examples/chapter10/05_circuit_breaker/after/breaker/example_test.go:example}}
```

接到 handler：斷路器擋下就回 503 加 `Retry-After`，逾時與其他錯誤維持原本的狀態碼。`Timeout` 設在 `Do` 裡面，每次真的呼叫下游才計時。

```go,ignore
{{#include ../../../examples/chapter10/05_circuit_breaker/after/quote/quote.go:handler}}
```

同樣的 10 個請求，承運商一樣不回應：

```go,ignore
{{#include ../../../examples/chapter10/05_circuit_breaker/after/quote/quote_test.go:failfast}}
```

下游只被呼叫 3 次，花 9 秒（3 次逾時）；其餘 7 個請求立刻回 503，不佔下游的連線。冷卻之後承運商恢復，試探成功，斷路器關上。

## 取捨

**門檻與冷卻怎麼選。** 門檻太低，偶發的失敗就誤打開；太高，下游掛了很久才反應，中間每個請求還是等滿逾時。冷卻太短，試探會一直打到還沒恢復的下游，每次試探都要等一次逾時；太長，下游早就好了還是被擋著。兩個數字都該從下游實際的故障與恢復時間來估，並用指標觀察，不是一次設好。

**失敗率門檻。** 上面只數連續失敗，流量低時最直觀。流量大、錯誤零星混在成功之間時，改成「最近一段時間的失敗率超過門檻」，要有滑動視窗與最少請求數，否則流量低時一次失敗就是 100%。

**哪些錯誤算失敗。** `Do` 把 `fn` 回傳的每個非 nil 錯誤都算失敗。呼叫端自己取消、下游回 4xx 這類不是下游壞掉的錯誤，要在 `fn` 裡轉成 nil，否則一批壞請求就能把斷路器打開。

**和重試的順序。** 重試在外、斷路器在內：每一次嘗試都經過斷路器，每次失敗都算數；斷路器打開之後，重試要把 `ErrOpen` 當作不可重試的錯誤，立刻放棄，不是退避等待（可重試的條件本章〈重試、退避與 jitter〉談過）。反過來，斷路器在外、重試在內，一整串重試只算一次失敗，下游在斷路器反應之前先被放大好幾倍的流量。

**打開時回什麼。** 上面回 503 與 `Retry-After`（冷卻時間是上限，實際剩下的可能更短）。也可以回快取的運費或固定費率，並在回應裡標示是降級的結果，本章〈負載卸載與降級〉會談。

**範圍。** 一個下游一個斷路器，不要幾個下游共用；共用的話，一個壞了會擋掉其他下游。每個副本各有自己的斷路器，狀態不共享，各自偵測、各自恢復，這通常夠用。狀態只有三種，放進指標沒有高基數問題，設計見第 9 章〈[指標（metrics）設計與高基數問題](../chapter9/03_metrics_cardinality.md)〉。

**自己寫還是用套件。** 上面是示範判斷用的最小版本。需要失敗率統計、半開放行多個試探、狀態變化的回呼時，用成熟的套件，例如 `sony/gobreaker`；服務網格也能在網路層做類似的事。資源隔離（一個慢的下游不拖垮其他功能）本章〈bulkhead〉會談。

**before 什麼時候夠用？** 下游穩定、只偶爾逾時，或流量小到等待中的請求累積不起來時，逾時加重試就夠了，不必多一個要調參數的元件。

## 重點整理

- 逾時只保護單一請求；下游已經掛了，每個請求還是等滿逾時，斷路器讓後面的請求立刻失敗。
- 三個狀態：關閉、打開、半開；半開只放一個試探，成功才關上，失敗就重新打開並重新計冷卻。
- 冷卻用注入的時鐘或 `synctest` 測，狀態轉換、打開時不呼叫下游、半開只有一個試探，都是確定的斷言。
- 門檻與冷卻要從下游的故障與恢復時間估；不是下游壞掉的錯誤不該計入。
- 重試在外、斷路器在內，`ErrOpen` 不重試；打開時回 503 與 `Retry-After`，或明確標示的降級回應。
