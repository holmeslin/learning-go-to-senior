# 負載卸載與降級

## 本集目標

流量超出服務的處理能力時，判斷該讓所有請求一起排隊，還是在入口提早拒絕一部分：用同時處理數當門檻，依重要性決定先拒絕誰，非核心的功能改回便宜的回應，並明說這是降級。

## 情境

商品服務有兩支 API：`/checkout/quote` 是結帳報價，失敗就是掉單；`/recommendations` 是推薦，慢一點或舊一點都沒人在意。活動開跑，流量瞬間超過平常的兩倍。

真正做事的容量是固定的：CPU 核心、資料庫連線池。請求再多，同時能做的事也不會變多，goroutine 比 P 多並不會讓 CPU 工作更快，第 8 章〈[排程器（GMP）怎麼看](../chapter8/01_scheduler_gmp.md)〉談過。第 6 章〈[背壓](../chapter6/08_backpressure.md)〉處理的是一個佇列滿了怎麼辦；這集看的是整個服務的入口。HTTP handler 與 `httptest` 前作教過：〈[`httptest`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter14/09_httptest.html)〉。

## 常見寫法

照單全收：每個請求都進 handler，在容量前排隊，輪到了才處理。

```go,ignore
{{#include ../../../examples/chapter10/07_load_shedding/before/handler.go:handler}}
```

流量在容量以內，這個寫法最簡單，也沒有任何請求被誤拒。完整程式碼在 `examples/chapter10/07_load_shedding/before/`。

## 問題在哪

容量是 2、每個請求做 1 秒，所以每秒做完 2 個。測試讓請求每秒進來 4 個，持續 10 秒。時間用 `testing/synctest` 的假時間（用法見第 5 章〈[測時間與並行：用 `testing/synctest`，還是自己注入時鐘](../chapter5/08_synctest_or_clock.md)〉），它是確定的，所以延遲可以精確斷言：

```go,ignore
{{#include ../../../examples/chapter10/07_load_shedding/before/backlog_test.go:backlog}}
```

40 個請求全部回 200，沒有人被拒絕，但最慢的等了 11 秒。排隊的深度每秒多 2 個，每個新請求都排在更後面，延遲一路往上。呼叫端只願意等 3 秒，前 10 個趕得上，之後每一個都來不及。

更糟的是，伺服器還是把 40 個都做完，其中 30 個是替早就放棄的呼叫端做白工，白工又讓後面的請求等更久。排隊的請求各占著 goroutine 與連線，核心的報價和不重要的推薦排在同一條隊伍裡，一起變慢，一起逾時。

## 較好的做法

進來的時候就決定收不收，不排隊。`Shedder` 記著目前的同時處理數，狀態在自己的值裡：

```go,ignore
{{#include ../../../examples/chapter10/07_load_shedding/after/shedder.go:shedder}}
```

`limit` 是總上限，`reserve` 個名額只留給 `Critical`：`Optional` 最多用到 `limit-reserve`，所以推薦把自己的份額用完時，報價仍有位子。`Handler` 包住真正的 handler：

```go,ignore
{{#include ../../../examples/chapter10/07_load_shedding/after/shedder.go:handler}}
```

沒有名額時有兩條路。沒有 `fallback`，回 503 加 `Retry-After`，呼叫端知道多久之後再來。有 `fallback`，改回便宜的內容（快取、部分結果），並由 `Handler` 自己加上 `X-Degraded: overload`，標示不會因為某個 `fallback` 忘了寫而漏掉。`fallback` 不占名額，降級本身不會再製造負載。

測試用的 helper 在下面：`gate` 是阻塞的 handler，進來就算一個處理中的請求，直到放行才回應；`fire` 同時送出一批請求。

```go,ignore
{{#include ../../../examples/chapter10/07_load_shedding/after/shedder_test.go:helpers}}
```

並行的測試不用等任何時間：`synctest.Wait` 等到所有 goroutine 都卡住（放得進去的卡在 `gate`，被拒的已經回應），再數結果。總上限 4、保留 2：5 個 `Optional` 只進 2 個，其餘 3 個拿到 503 與 `Retry-After`；接著 3 個 `Critical` 進 2 個（用到保留的名額），第 3 個超過總上限被拒：

```go,ignore
{{#include ../../../examples/chapter10/07_load_shedding/after/shedder_test.go:shed}}
```

這個測試證明了並行與保留名額：把 `reserve` 忽略，`Optional` 會進 4 個；讓 `next` 一個接一個執行，同時處理數只會是 1，兩種改法測試都會失敗。降級的測試確認回應帶標示，而且沒有呼叫真正的 handler：

```go,ignore
{{#include ../../../examples/chapter10/07_load_shedding/after/shedder_test.go:degrade}}
```

同樣的流量（每秒 4 個，持續 10 秒），這次容量 2、每個請求 800 ms：20 個被接受、20 個被拒絕，最慢的回應就是 800 ms，被拒絕的 0 秒就回來了：

```go,ignore
{{#include ../../../examples/chapter10/07_load_shedding/after/shedder_test.go:waves}}
```

回應長什麼樣子，用 `Example` 看：

```go,ignore
{{#include ../../../examples/chapter10/07_load_shedding/after/example_test.go:example}}
```

## 取捨

- **門檻從量測來，不是猜的。** 壓測找出同時處理數多到什麼程度，延遲會超過 SLO 的門檻（第 9 章〈[SLI 與 SLO](../chapter9/08_sli_slo.md)〉）。同時處理數大約是每秒完成數乘以延遲：例如每秒能完成 100 個、延遲目標 300 ms，起點是 30 個左右（假設的數字，要用壓測校正）。太低會在正常的尖峰誤拒合法請求，太高就跟沒有門檻一樣。
- **降級要讓呼叫端看得到。** 推薦用舊資料沒關係；價格與庫存不能降級成舊的，這種請求只有 503 一條路。呼叫端要能處理 `X-Degraded`（前端少顯示一塊、快取要有時效），否則降級只是悄悄給錯的資料。範例沒有放計數器；實際使用時，拒絕與降級的次數要記成指標，屬性用路由與重要性，不用使用者代號，見第 9 章〈[指標（metrics）設計與高基數問題](../chapter9/03_metrics_cardinality.md)〉。
- **卸載和限流不同。** 限流（本章〈[`golang.org/x/time/rate` 限流](04_rate_limit.md)〉談過）看每個 client 每秒送了幾個，不管伺服器現在忙不忙，超過回 429；卸載看伺服器此刻同時在做幾件事，不分是誰送的，超過回 503。限流擋住單一 client 的超額，卸載保護整體，兩者通常一起設。
- **重要性由伺服器決定。** 這裡依路由決定。讓 client 用標頭自報優先級，任何人都能把自己標成 `Critical`；服務之間要傳，也得在邊界驗證。
- **被拒絕的 client 要尊重 `Retry-After`。** 立刻重試只會讓入口更擠，退避與 jitter 本章〈[重試、退避與 jitter](02_retry_backoff_jitter.md)〉談過。卸載也不該讓 readiness 失敗，否則所有實例同時退出流量，健康檢查的做法見第 9 章〈[健康檢查](../chapter9/07_health_check.md)〉。
- **只看同時處理數，看不到排隊的代價。** 這裡完全不排隊；短暫的突波可以另外留一個很短的佇列，長度仍要由延遲目標決定。一個慢的下游把名額占光、連不相關的功能也被拒絕，本章〈bulkhead〉會談。
- **before 什麼時候夠用？** 內部的批次工具、client 數量固定而且自己會控制速度，或容量遠大於峰值的服務，排隊的壞處不會出現。

## 重點整理

- 容量固定時，多收的請求只會排隊：每個都變慢，最後超過呼叫端的期限，伺服器還替放棄的人做白工。
- 在入口依同時處理數提早拒絕，回 503 加 `Retry-After`；保留名額讓核心請求在非核心塞滿時仍進得來。
- 非核心功能改回快取或部分結果，並在回應標示（`X-Degraded`），降級才不是悄悄發生。
- 門檻從壓測與 SLO 的延遲目標來；重要性由伺服器決定，不信 client 自報。
- 卸載看伺服器此刻的負載，限流看單一 client 的速率，兩者一起用。
- 測試用阻塞的 handler 加 `synctest` 數同時處理數，用假時間斷言延遲，不依賴真實時間。
