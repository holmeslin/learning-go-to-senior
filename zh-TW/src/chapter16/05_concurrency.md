# 並行

## 本集目標

收到結束訊號時，判斷提醒服務裡每個 goroutine 歸誰管、各部分依什麼順序停，並讓每個請求的逾時與關機期限加起來對得上；同時分清並行上限與每秒 50 次的限流各擋什麼。

## 情境

提醒服務只跑一個副本，發布新版本時先停舊、再起新（本章〈[設計文件](01_design_doc.md)〉）。平台送出 `SIGTERM` 的那一刻，預設最多有 20 則送出在路上（`SendConcurrency`），每一則都已經寫了「寄送中」的紀錄，之後還要寫結果。

服務本體的 `Service.Run` 在 `examples/chapter16/reminder/`。before 與 after 另放在 `examples/chapter16/05_concurrency/`，是不 import 服務本體的縮小版（服務的套件在 `internal/`，也 import 不到）：`Ledger` 在記憶體裡代替寄送紀錄表，只保留關機看得到的兩個性質，關閉之後寫不進去，ctx 已經結束的寫入會失敗（同 `ExecContext`）。這集只談 goroutine 的擁有關係、取消、逾時與限流組在一起時的順序與期限；個別的做法，下面各處連回對應的章。

## 常見寫法

`Run` 開一個 goroutine 掃描，ctx 結束就關資料庫、回傳；ctx 一路傳進送出與寫入：

```go,ignore
{{#include ../../../examples/chapter16/05_concurrency/before/shutdown.go:run}}
```

```go,ignore
{{#include ../../../examples/chapter16/05_concurrency/before/shutdown.go:deliver}}
```

## 問題在哪

測試放進 5 則提醒、並行上限 2、每個請求要 3 秒，在第 1 秒送出結束訊號（`synctest` 的假時間，用法見第 5 章〈[測時間與並行：用 `testing/synctest`，還是自己注入時鐘](../chapter5/08_synctest_or_clock.md)〉）。它鎖住問題本身，所以在 before 通過：

```go,ignore
{{#include ../../../examples/chapter16/05_concurrency/before/shutdown_test.go:cut}}
```

三件事同時發生：

- **沒有人等。** `Run` 在訊號當下回傳，`main` 接著結束行程；兩則送出做到一半，沒有人知道它們做完沒有（第 6 章〈[goroutine 的生命週期歸誰管](../chapter6/01_goroutine_ownership.md)〉）。
- **取消切斷了送出。** 兩個請求被放棄；供應商可能已經收到，這一邊只看到 `context canceled`。
- **結果寫不進去。** 紀錄簿先關了，ctx 也結束了，兩則都停在 `sending`。這個狀態不會被再挑出（本章〈[錯誤與資料層](03_errors_and_data.md)〉），預定時間後 1 小時由逾期的檢查記為失敗：推播可能已經送達，紀錄卻寫著失敗。

反方向也有錯法：有人等，卻沒有期限。送出沒有各自的逾時、供應商不回應，關機就一直等，等到平台用 `SIGKILL` 結束行程，結果和 before 一樣。

## 較好的做法

先決定誰擁有什麼，再決定順序：

| goroutine | 誰開 | 誰等 |
| --- | --- | --- |
| 掃描迴圈 | `Service.Run` | `Service.Run`，關機時等它回傳 |
| 每一則送出 | `Scan` 裡的 `errgroup` | `Scan`，進行中的都做完才回傳 |
| HTTP 伺服器（內部 API、診斷端點） | `Service.Run` | `Service.Run`，關機時 `Shutdown` |

收到 `SIGTERM`（ctx 結束）之後的順序如下，期限 `ShutdownTimeout` 從這一刻起算：

1. readiness 先變成未就緒，讓負載平衡器開始把新的流量導開。
2. 停止新的掃描：ctx 結束，`dispatch` 不再拿新的提醒。
3. 等進行中的送出做完。
4. 關 HTTP 伺服器，送出最後的指標與追蹤（本章〈可觀測性與韌性〉會談）。
5. 關資料庫。

資料庫在最後：第 3 步的送出要寫結果，第 4 步要等進行中的 API 請求做完，它們還要讀紀錄。縮小版只有第 2、3、5 步：

```go,ignore
{{#include ../../../examples/chapter16/05_concurrency/after/shutdown.go:run}}
```

期限用 `WithoutCancel` 另外算，否則 ctx 已經結束，期限一開始就到了（第 10 章〈[關機時該等什麼、等多久（graceful shutdown）](../chapter10/06_graceful_shutdown.md)〉）；等的時候同時等期限，到了就回傳錯誤，不無限等下去。`deliver` 負責另一半：

```go,ignore
{{#include ../../../examples/chapter16/05_concurrency/after/shutdown.go:deliver}}
```

取消只決定「要不要開始這一則」，開始之後，送出與寫入都切開取消。切開時 deadline 也一起拿掉（第 10 章〈[逾時預算](../chapter10/01_timeout_budget.md)〉、本章〈[錯誤與資料層](03_errors_and_data.md)〉談過），所以請求要有自己的逾時；記結果用沒有逾時的 ctx，請求逾時之後仍寫得進去。三個測試各鎖一個性質：進行中的做完且記為已寄出，`Run` 在訊號後 2 秒回傳；供應商不回應時，最多等一個請求逾時；期限比請求逾時短時，到期就回傳錯誤。`synctest.Test` 同時保證沒有殘留的 goroutine（第 6 章〈[goroutine 洩漏怎麼發生、怎麼在測試裡抓到](../chapter6/09_goroutine_leak.md)〉）。

```go,ignore
{{#include ../../../examples/chapter16/05_concurrency/after/shutdown_test.go:finish}}
```

```go,ignore
{{#include ../../../examples/chapter16/05_concurrency/after/shutdown_test.go:deadline}}
```

服務本體是同樣的形狀。下面是從「等進行中的送出」到「關資料庫」的那一段，期限同樣從沒有被取消的 context 另外算；整個服務的端對端測試（本章〈[測試](04_testing.md)〉）用 `TestServiceStopsWithinShutdownTimeout` 斷言它：

```go,ignore
{{#include ../../../examples/chapter16/reminder/service.go:shutdown}}
```

期限怎麼算，看設定的預設值與 `shutdownMargin`：

```go,ignore
{{#include ../../../examples/chapter16/reminder/internal/config/config.go:defaults}}
```

進行中的一則在 `deliver` 裡，順序是限流、寫紀錄、送出、記結果，切開取消之後每一步各有自己的逾時：

```go,ignore
{{#include ../../../examples/chapter16/reminder/internal/dispatch/dispatch.go:deliver}}
```

最壞要等三段：寫紀錄的查詢逾時、請求逾時、記結果的查詢逾時，預設都是 5 秒，共 15 秒；`ShutdownTimeout` 預設 30 秒，還要留給第 4、5 步。

並行上限與限流是兩件事，各管一個地方（第 6 章〈[`errgroup` 的並行上限](../chapter6/03_errgroup_limit.md)〉、第 10 章〈[`golang.org/x/time/rate` 限流](../chapter10/04_rate_limit.md)〉談過）：

| | 並行上限（預設 20） | 限流（預設每秒 50） |
| --- | --- | --- |
| 擋的是 | 同時在路上的請求數 | 每秒送出的請求數 |
| 為了誰 | 自己：goroutine、連線與記憶體 | 供應商：合約是每秒 50 次 |
| 成為瓶頸的時候 | 供應商慢 | 供應商快 |
| 做法 | `Scan` 裡 `errgroup` 的 `SetLimit` | `deliver` 裡 `rate.Limiter` 的 `Wait`，burst 為 1 |

`TestScanRespectsTheRateLimit` 讓呼叫不占時間，並行上限 20 沒有用上，100 則仍要約 2 秒才送完；`TestScanLimitsConcurrency` 每則占 1 秒，12 則、上限 5，要 3 輪。

## 取捨

這集的取捨都來自前面各章的判斷湊在一起：

- **取消不切斷進行中的送出，關機就要等。** 最壞的 15 秒是兩個查詢逾時加一個請求逾時，設定檢查（`shutdownMargin`）卻只要求 `ShutdownTimeout` 比請求逾時多 5 秒：調到最小值 10 秒，碰上資料庫也卡住就等不完。它還要比部署平台給的寬限期短（第 10 章〈[關機時該等什麼、等多久（graceful shutdown）](../chapter10/06_graceful_shutdown.md)〉談過），預設的 30 秒要和平台的設定一起訂。單一實例先停舊、再起新，關機最久的 30 秒也算在空窗裡，相對預定時間後的 1 小時用得起。
- **限流的等待放在寫紀錄之前。** `deliver` 先 `Wait`、再 `Begin`。寫紀錄之後的等待若被取消，這一則就停在 `sending`，連試都沒試過卻被記成失敗；放在前面，取消丟掉的只是還沒寫紀錄的，重啟後的第一輪會再挑到。代價是通過限流之後 ctx 就得切開：寫紀錄到送出之間，不能有會被取消的等待。
- **並行上限太小，批次反而趕不上 1 小時。** 送出的速度是每秒的限流與「並行上限 ÷ 每則耗時」兩者中小的那個。供應商每則在 0.4 秒內，20 ÷ 0.4 不小於 50，限流是瓶頸，兩萬人要 400 秒；每則都拖到請求逾時的 5 秒，上限 20 只剩每秒 4 則，兩萬人要超過 80 分鐘，超過了預定時間後 1 小時的期限。上限是為了保護自己（第 15 章〈[用 `pprof` 與 trace 在現場診斷](../chapter15/03_pprof_trace_live.md)〉的事故），但不是越小越保險，怎麼依證據調整，本章〈壓測與調校〉會談。

before 在這些情況下反而夠用：每一輪都從來源重新挑、重複送出無害（供應商認冪等鍵，第 10 章〈[冪等性](../chapter10/03_idempotency.md)〉），也沒有「寄送中」這種不再被挑出的中間狀態，直接結束只是多做一次。這個服務為了「寧可不寄，也不重複」先寫紀錄、不補寄停在寄送中的，所以關機要把進行中的每一則做完。

## 重點整理

- goroutine 的擁有關係與關機順序一起設計：掃描迴圈歸 `Run`、每一則送出歸 `Scan`，資料庫最後關。
- 取消只決定要不要開始下一則；開始之後切開取消，請求有自己的逾時，期限從沒有被取消的 context 另外算。
- 關機期限要大於進行中那一則的最壞耗時加餘裕，最壞耗時由每個請求與每個查詢的逾時決定，期限也要小於平台的寬限期。
- 並行上限擋同時幾個、限流擋每秒幾個，速度取決於比較小的那個；限流的等待放在寫紀錄之前。
