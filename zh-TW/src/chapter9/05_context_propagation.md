# 在 `context` 裡傳遞 trace

## 本集目標

開 goroutine 或對外發請求時，判斷 trace 怎麼接起來：同一個行程裡靠 `context` 一路傳，跨行程靠 propagator 在標頭注入與取出；日誌帶上 trace ID；背景工作則在「延續 trace」與「開新的 trace 再連回去」之間選。

## 情境

出貨追蹤服務：gateway 收到 `GET /track?id=A1`，向承運商（carrier）服務查狀態，回覆之後在背景留一筆稽核紀錄。兩個服務是兩個行程，各自有自己的 `TracerProvider`。

span 是什麼、`TracerProvider` 為什麼用注入的，本章〈[分散式追蹤與 OpenTelemetry](04_tracing_opentelemetry.md)〉談過。`context` 的傳遞慣例前作教過：〈[context 傳遞慣例](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter10/07_context_conventions.html)〉；取消怎麼傳進 goroutine，見第 6 章〈[取消與 `context` 的傳遞](../chapter6/04_cancellation_context.md)〉。這集談的是同一個 `ctx` 還帶著 trace。測試用 `httptest`：〈[`httptest`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter14/09_httptest.html)〉。

## 常見寫法

`ctx` 傳進 `lookup`，也傳給 `NewRequestWithContext`；背景工作怕被請求結束取消，改用 `context.Background()`：

```go,ignore
{{#include ../../../examples/chapter9/05_context_propagation/before/gateway.go:handler}}
```

承運商那邊的 handler 也寫對了：它會從請求標頭取出上游的 trace，再開自己的 span。完整程式碼在 `examples/chapter9/05_context_propagation/before/`。

## 問題在哪

用 `SpanRecorder` 把一次請求產生的 span 全部收起來，依名稱印出每個 span 的父 span，以及它屬於哪一條 trace：

```go,ignore
{{#include ../../../examples/chapter9/05_context_propagation/before/example_test.go:example}}
```

四個 span 分在三條 trace。`audit` 是新的根：`Start` 把 span 放進它回傳的 `ctx`，`context.Background()` 沒有這個 span，所以背景工作的 span 無處可接。`carrier.lookup` 也是新的根：`ctx` 只活在一個行程裡，`NewRequestWithContext` 只拿它管取消與逾時，不會把 span 寫進標頭，承運商的 `Extract` 找不到 `traceparent`。

後果是在 trace 後端看到三段短短的 trace，查不出哪個 `audit` 屬於哪一次請求，也看不到承運商那一段是誰叫的。

## 較好的做法

三處修正：對外請求注入標頭、背景工作用 `context.WithoutCancel(ctx)`、日誌帶 trace ID：

```go,ignore
{{#include ../../../examples/chapter9/05_context_propagation/after/gateway.go:handler}}
```

`propagation.TraceContext{}` 實作 W3C Trace Context：`Inject` 把 `ctx` 裡目前的 span（這裡是 `carrier.request`）寫成 `traceparent`，`Extract` 在另一端還原。`WithoutCancel` 回傳的 `ctx` 不會被取消、也沒有 deadline，但保留原本的值，span 還在，所以 `audit` 接在 `GET /track` 底下。`Wait` 與 `audit` 回呼是為了測試：背景工作在回覆之後才跑，測試呼叫 `Wait` 等它結束、用 `audit` 決定背景做什麼，不必靠 `time.Sleep` 猜時間。日誌用下面這個函式把 trace ID 放進欄位，鍵名沿用本章〈[日誌層級與欄位設計](02_log_levels_and_fields.md)〉的約定（snake_case、錯誤用 `error`、定成常數），`slog` 的用法見〈[`log/slog`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter12/10_slog.html)〉：

```go,ignore
{{#include ../../../examples/chapter9/05_context_propagation/after/gateway.go:withtrace}}
```

承運商一端用 `Extract` 把上游的 trace 接進自己的 `ctx`，再 `Start`：

```go,ignore
{{#include ../../../examples/chapter9/05_context_propagation/after/carrier.go:carrier}}
```

測試用兩個 provider、兩個 recorder，模擬兩個行程，斷言四個 span 同一條 trace、父子關係正確，而且 `carrier.lookup` 的父 span 是遠端的：

```go,ignore
{{#include ../../../examples/chapter9/05_context_propagation/after/track_test.go:sametrace}}
```

另外兩個測試鎖住其他行為：承運商掛掉時，gateway 回 502，錯誤日誌帶同一個 trace ID；請求的 `ctx` 取消之後，背景的 `ctx.Err()` 仍是 `nil`，用 channel 排定先後，沒有 `time.Sleep`。

`Example` 用固定的 ID 產生器，讓 `traceparent` 與日誌裡的 trace ID 都能逐字驗證：

```go,ignore
{{#include ../../../examples/chapter9/05_context_propagation/after/example_test.go:ids}}
```

```go,ignore
{{#include ../../../examples/chapter9/05_context_propagation/after/example_test.go:example}}
```

`traceparent` 的格式是版本、trace ID、parent span ID、旗標，以 `-` 相連。parent span ID 是 `…0002`，是 `carrier.request` 的 span ID（第二個產生的），不是 gateway 的 `GET /track`，所以承運商的 span 是 client span 的子 span。

## 取捨

背景工作有兩種接法。延續 trace（上面的做法）能在同一條 trace 看到稽核，代價是子 span 可以比父 span 晚結束；工作很久或會重試，這條 trace 就被拖得很長。工作是獨立的（排程、批次、多個請求觸發同一件事），改開新的根，再用 link 指回原請求：

```go,ignore
{{#include ../../../examples/chapter9/05_context_propagation/after/track_test.go:link}}
```

請求快的、稽核算請求的一部分，延續；背景工作自成一件事，用 link。

`WithoutCancel` 拿掉的不只取消，還有 deadline：背景工作若卡住，沒有人會叫它停，要自己再包一層逾時，第 10 章〈[逾時預算](../chapter10/01_timeout_budget.md)〉談過。它保留所有的值，不只 span。

`Inject` 與 `Extract` 要在每個邊界各寫一次。contrib 的 `otelhttp` 把它們包進 `Transport` 與 `Handler`；程式進入點也可以用 `otel.SetTextMapPropagator` 設全域的 propagator。這一集不碰全域設定，測試才能各用各的 provider 平行跑。對外部的第三方服務，注入 `traceparent` 等於把內部的 trace ID 送出去，要不要注入由邊界決定。

日誌帶 trace ID 也可以自訂 `slog.Handler`，在 `Handle` 讀 `ctx` 自動加欄位。要轉發 `WithAttrs` 與 `WithGroup`，呼叫端也得用 `InfoContext`，才有 `ctx` 可讀；呼叫點不多時，上面的函式比較簡單。

Baggage 是另一個 propagator：`propagation.Baggage` 讓 key-value 跟著請求傳給每個下游。傳遞是自動的，下游全都收得到，標頭會變大；不要放個人資料與秘密。

before 在單一行程、沒有跨服務呼叫，背景工作本來就獨立時，沒有問題。指標與追蹤在整套服務裡怎麼配合，第 16 章〈可觀測性與韌性〉會談。

## 重點整理

- `ctx` 只在行程內帶著 span；`context.Background()` 開的工作會變成新的根，`NewRequestWithContext` 不會寫標頭。
- 跨服務要在 client 用 `propagation.TraceContext` 的 `Inject` 寫入 `traceparent`，在 server 用 `Extract` 還原，trace 才連得起來。
- 日誌帶目前 span 的 trace ID，就能從一行日誌找到整條 trace。
- 背景工作用 `context.WithoutCancel` 延續 trace 又不被請求取消；獨立的長工作用新的根加 span link。
- 測試用 `SpanRecorder` 斷言 trace ID 與父子關係；要印 trace ID 時用固定的 ID 產生器。
