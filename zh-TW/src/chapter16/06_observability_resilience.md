# 可觀測性與韌性

## 本集目標

告警從使用者看得到的結果出發（提醒有沒有在預定時間後 1 小時內送出），每個告警連到 runbook；並判斷指標、日誌與追蹤各回答什麼問題、韌性的工具哪些要加、哪些不加，以及關機時怎麼讓最後一段指標與追蹤送得出去。

## 情境

第 15 章事故的行動項目 2 是 goroutine 與 heap 的告警，行動項目 6 是 runbook（第 15 章〈[行動項目與追蹤](../chapter15/05_action_items.md)〉、〈[runbook](../chapter15/06_runbook.md)〉）。提醒服務拆出來之後，上線前要有它自己的告警，每個都連到 runbook。服務的指標、日誌與追蹤已經在 `examples/chapter16/reminder/` 裡，指標的設計見第 9 章〈[指標（metrics）設計與高基數問題](../chapter9/03_metrics_cardinality.md)〉；這集決定看什麼、什麼時候叫醒值班工程師，以及這個服務要加哪些韌性的工具（平台與數字都是教材範例）。

## 常見寫法

先照報名服務現成的兩條告警，條件寫的是機器看得到的症狀，這正是第 15 章那次事故的樣子（記憶體上升、報名 API 回 5xx）：

```text
告警：記憶體使用率高（通知值班工程師）
  條件：容器記憶體使用率 > 80%，持續 5 分鐘
  runbook：（沒有）
告警：錯誤率高（通知值班工程師）
  條件：HTTP 5xx 的比例 > 1%，持續 5 分鐘
  runbook：（沒有）
```

## 問題在哪

提醒服務做了行動項目 1（逾時與並行上限）之後，事故的症狀不會再出現，它現在的失敗是安靜的。下面讓供應商整個掛掉，每個請求回 503：一場活動 200 位報名者（兩萬人的 1/100），預定時間設在 58 分鐘前，掃描間隔縮成 10 秒（預設 5 分鐘），所以約 2 分鐘後就過了預定時間後 1 小時；服務啟動後約 6 秒與 135 秒各查一次。平台是 `uname -sm` 的 `Darwin arm64`，`go version` 是 `go version go1.27.0 darwin/arm64`；重現腳本在 repo 外，指令照列。資料庫先用 `sqlite3` 套 `demoreg/schema.sql`、灌一場活動與 200 位報名者；假供應商是只回固定狀態碼、只聽 `127.0.0.1` 的小程式：

```bash
go build -o reminder ./examples/chapter16/reminder/cmd/reminder
REMINDER_PROVIDER_URL=http://127.0.0.1:9000 REMINDER_SEND_ENABLED=true REMINDER_DB_PATH=reminder.db \
REMINDER_API_ADDR=127.0.0.1:8080 REMINDER_SCAN_INTERVAL=10s ./reminder
curl -s http://127.0.0.1:6060/debug/metrics | grep -E 'result="(sent|expired)"|overdue|goroutines|heap'
curl -s -o /dev/null -w 'api %{http_code}\n' http://127.0.0.1:8080/v1/events/1/reminder-counts
```

執行結果（某一次）：快照省略 `retrying`（累計的次數，第二份比第一份多，是有些提醒到了重試的時間、再失敗一次）、`failed`、`missed`、`skipped`、`errored`（後四項兩次都是 0）與 `reminder.scan.duration`；`--- t=` 是重現腳本印的標頭，單位是秒：

```text
--- t=6s
reminder.deliveries{result="expired"} 0
reminder.deliveries{result="sent"} 0
reminder.overdue.unsent 200
runtime.goroutines 32
runtime.heap.bytes 2703016
api 200
--- t=135s
reminder.deliveries{result="expired"} 200
reminder.deliveries{result="sent"} 0
reminder.overdue.unsent 0
runtime.goroutines 32
runtime.heap.bytes 2759016
api 200
```

- goroutine 與 heap 兩次相近，沒有隨失敗的提醒增加：並行上限與逾時讓它們有上限（本章〈[並行](05_concurrency.md)〉），「記憶體使用率高」不會響。
- 內部 API 兩次都回 200：服務唯一的 HTTP 介面是唯讀 API，供應商回 503 不是任何人收到的 HTTP 錯誤，「錯誤率高」也不會響。
- 第一份快照，200 則已過預定時間仍未寄出；第二份，200 則都記為 `expired`，`sent` 從頭到尾是 0。使用者的提醒一則都沒寄出，兩條告警從頭到尾沒響；就算響了，也沒有 runbook 告訴值班工程師看什麼、能不能重啟。

## 較好的做法

### 告警

SLI 從使用者看得到的結果出發，來自 `reminder.deliveries`。告警分兩層：一層在期限內、還來得及處理的時候響，一層看錯誤預算。消耗速度與多窗口，第 9 章〈[SLI 與 SLO](../chapter9/08_sli_slo.md)〉只提了概念，這裡實際套用：

```text
SLI：預定時間後 1 小時內送出的比例 = sent / (sent + failed + expired + missed)；retrying 還不是終點，不進分母
SLO：99.5%，30 天（錯誤預算 0.5%）；消耗速度 = (1 - SLI) / 0.005
告警 1：逾期未寄（叫醒值班工程師）
  條件：reminder.overdue.unsent > 100，持續 20 分鐘
  runbook：https://runbooks.example.test/reminder/overdue-unsent
告警 2：錯誤預算快速消耗（叫醒值班工程師）
  條件：消耗速度 > 14.4，1 小時與 15 分鐘兩個窗口同時成立
  runbook：https://runbooks.example.test/reminder/budget-burn
告警 3：goroutine 數量（叫醒值班工程師）
  條件：runtime.goroutines > 200，持續 10 分鐘
  runbook：https://runbooks.example.test/reminder/goroutines-heap
告警 4：heap（叫醒值班工程師）
  條件：runtime.heap.bytes > 容器記憶體上限的 60%，持續 15 分鐘
  runbook：https://runbooks.example.test/reminder/goroutines-heap
```

- **99.5% 是這份範例訂的**，實際看歷史資料與產品負責人在意的程度。告警 1 的 100 則是一場 2 萬人的活動乘以 0.5%：比它少的逾期，多半是個別提醒在退避、等下一次重試，不值得叫人。20 分鐘的估法：最壞等 5 分鐘到下一次掃描，再以每秒 50 次送完 2 萬則（400 秒，約 7 分鐘），約 12 分鐘就歸零；超過 20 分鐘還沒歸零，不是在排隊，是卡住了，離 1 小時的期限還有 40 分鐘可以處理。
- **告警 2 是多窗口**：長窗口確認不是一時的雜訊，短窗口確認問題還在。短窗口取 15 分鐘，不是長窗口的 1/12（5 分鐘）：掃描每 5 分鐘一次，計數在每批掃完才加，5 分鐘的窗口可能剛好一批都沒有。慢速消耗開 ticket 的規則（第 9 章的做法，窗口更長）形狀相同，省略。
- **告警 3、4 用服務輸出的兩個 runtime 指標**，讀法見第 8 章〈[`runtime/metrics`](../chapter8/06_runtime_metrics.md)〉。200 約是上面快照的 6 倍，都是起始值，壓測之後再調，本章〈壓測與調校〉會談。
- **runbook 的連結寫在告警裡**，值班工程師從告警一步點到。怎麼寫見第 15 章〈[runbook](../chapter15/06_runbook.md)〉；新服務要改的地方（診斷位址、開關名稱），本章〈上線檢查表〉會談。

### 指標、日誌與追蹤

三者回答不同的問題，所以放的東西也不同：

| | 回答 | 這個服務的內容 | 一律不放 |
| --- | --- | --- | --- |
| 指標 | 現在整體有多糟 | `reminder.deliveries`、`reminder.scan.duration`、`reminder.overdue.unsent`、goroutine 與 heap | 活動、報名者、裝置 token（高基數） |
| 日誌 | 那一次發生了什麼 | 每批的 `scan finished` 計數；失敗的那一則帶活動與報名者編號、錯誤 | 裝置 token |
| 追蹤 | 一批的時間花在哪 | 每批一個 `reminder.scan`，每次送出一個 `reminder.send` 子 span，屬性是活動編號與結果 | 裝置 token、錯誤訊息 |

`reminder.deliveries` 只有 `result` 一個標籤、七種固定的值，不管這一批有多少人（`TestScanMetricsHaveOnlyTheResultLabel` 鎖住）。三個指標都在 `instrument` 建立：批次時間的 bucket 以秒為單位、一路到 3600 秒，所以一批超過 300 秒的掃描間隔（送不完）看得出來；逾期未寄用非同步的量測值，收集的當下才查資料庫，掃描卡住的時候它仍然是準的，查詢失敗只記日誌，不讓整份快照失敗：

```go,ignore
{{#include ../../../examples/chapter16/reminder/internal/dispatch/dispatch.go:instrument}}
```

追蹤的 span 包在 `Sender` 外面，不寫進 `deliver`：限流與寫紀錄的程式不必知道追蹤，等限流而被取消、沒有送出的提醒也不會留下 span。活動編號可以放 span 的屬性、不能放指標的標籤（第 9 章〈[分散式追蹤與 OpenTelemetry](../chapter9/04_tracing_opentelemetry.md)〉），裝置 token 兩邊都不放：

```go,ignore
{{#include ../../../examples/chapter16/reminder/internal/dispatch/dispatch.go:tracedsender}}
```

日誌也不記裝置 token（第 11 章〈[日誌裡的敏感資料](../chapter11/08_sensitive_data_in_logs.md)〉），`TestServiceLogsNeverContainDeviceTokens` 讓失敗、重試、放棄都走一遍，在整份日誌裡找 token。告警 1 響了之後，三者的順序是：指標說有多少則逾期，日誌找出是哪幾則、錯在哪，追蹤看這一批的時間是不是花在供應商那一段。

### 韌性的選擇

第 10 章的工具不是全部都加，依這個服務失敗的樣子選：

| 工具 | 這個服務 | 理由 |
| --- | --- | --- |
| 重試、退避與 jitter | 有 | 只重試暫時失敗，退避上限從 5 分鐘起每次加倍、最多 30 分鐘，只發生在掃描的時間點，期限是預定時間後 1 小時（第 10 章〈[重試、退避與 jitter](../chapter10/02_retry_backoff_jitter.md)〉）；`Retry-After` 本章〈[錯誤與資料層](03_errors_and_data.md)〉談過 |
| 限流 | 有 | 擋每秒送出幾個，供應商的上限是 50（第 10 章〈[`golang.org/x/time/rate` 限流](../chapter10/04_rate_limit.md)〉、本章〈[並行](05_concurrency.md)〉） |
| 逾時 | 有 | 每個請求與每個查詢各自逾時，關機期限另外算（第 10 章〈[逾時預算](../chapter10/01_timeout_budget.md)〉） |
| bulkhead | 拆成獨立行程 | 提醒 worker 與報名 API 共用行程時，供應商一慢，記憶體把兩邊一起拖垮；獨立行程各有自己的記憶體上限與 goroutine，就是第 10 章〈[bulkhead](../chapter10/08_bulkhead.md)〉依下游分開資源的做法，決定在 ADR-0019（本章〈[設計文件](01_design_doc.md)〉）。隔不開的是資料庫：檢視在報名服務那邊，慢查詢靠查詢逾時擋 |
| 斷路器 | 不加 | 見下 |

不加斷路器有三個理由。斷路器（第 10 章〈[斷路器](../chapter10/05_circuit_breaker.md)〉）要擋的是下游掛了之後，每個請求都等滿逾時、佔著 handler 與連線；這裡同時在路上的請求最多 20 個、各最多等 5 秒，行程裡沒有別的工作要保護，重試又只發生在掃描與退避之後，對供應商的壓力不會超過每秒 50 次。其次，加了要付代價：被擋下的那一則，結果是「沒有送出」，不是 `ErrTransient`（可能已經到了供應商），也不是 `ErrPermanent`；掃描迴圈要多一種結果，而且不能算一次嘗試，否則退避被白白拉長。第三，門檻與冷卻要從供應商實際的故障與恢復時間估，上線前沒有這份資料。會改變這個決定的是：加上 email 管道（第二個下游，範例省略，各要自己的斷路器），或供應商不回應時，一批被逾時拖得超過掃描間隔，`reminder.scan.duration` 看得到，本章〈壓測與調校〉會談怎麼量。

### 關機時先送出最後一段

追蹤的批次處理器預設每 5 秒才把緩衝裡的 span 送出一次，收到結束訊號的那一刻，最後一段還在記憶體裡，行程一結束就沒了。指標也一樣：範例用 `ManualReader`，讀的時候才收集，沒有東西要推；換成 `PeriodicReader` 加 exporter（預設每 60 秒送一次）就有同樣的問題。所以 `telemetry.Shutdown` 要在行程結束之前送完。順序是本章〈[並行](05_concurrency.md)〉的第 4、5 步：先等進行中的送出做完、關 HTTP 伺服器，它們的 span 與計數才完整，再 `Shutdown`，最後關資料庫，因為逾期未寄的指標在收集時要查資料庫。測試把前後兩種狀態都釘住：`Shutdown` 之前 exporter 一個 span 也沒收到，之後收到了緩衝裡的那一個。整個服務的版本是 `TestServiceSendsTheLastSpansOnShutdown`，用本章〈[測試](04_testing.md)〉的端對端組裝：

```go,ignore
{{#include ../../../examples/chapter16/reminder/internal/telemetry/telemetry_test.go:flush}}
```

## 取捨

- **告警 1 與告警 2 是接力，中間有一段看不到。** 超過期限的提醒被記為失敗（本章〈[錯誤與資料層](03_errors_and_data.md)〉），`reminder.overdue.unsent` 就歸零（上面第二份快照）；供應商一直暫時失敗時，提醒要到預定時間後 1 小時才記為 `expired`，計數又在一批掃完才加，告警 2 最早那時才響，預算已經燒掉了。所以告警 1 在期限內先響，告警 2 接手。代價是告警 1 的 100 與 20 分鐘上線前只是估計，值班工程師被吵醒的次數是調整的依據，本章〈上線檢查表〉會談上線後的第一週。
- **開關關閉時，告警 1 沒有資料。** `reminder.overdue.unsent` 只在送出的開關打開時才存在：上線第 1 步開關關閉，舊 worker 還在送，檢視裡「到期而沒有紀錄」的不是沒寄，回報會讓告警誤響。規則要把沒有資料當成正常；服務掛掉的靜默，靠平台的 readiness 探針發現（第 9 章〈[健康檢查](../chapter9/07_health_check.md)〉）。
- **`failed` 也包含供應商拒絕的裝置。** 裝置 token 失效不是服務的錯，卻吃預算，所以 SLO 不訂得更高。要排除，得新增一個 `result` 的值、細分供應商的 4xx，等第一週的比例再決定。
- **最需要最後一段資料的時候，期限已經沒了。** 關機的每一步共用一個 `ShutdownTimeout`；等進行中的送出等到期限，`Shutdown` 拿到的是過期的 context，一拿到就回傳錯誤、不等匯出完成，匯出留在背景，行程一結束就來不及。正常的關機有 5 秒的餘裕（`shutdownMargin`）。給送出另一段期限可以多留一些資料，代價是總時間可能超過平台的寬限期而被 `SIGKILL`（第 10 章〈[關機時該等什麼、等多久（graceful shutdown）](../chapter10/06_graceful_shutdown.md)〉）；這個服務選單一期限，那一次靠 `shutdown deadline passed` 的日誌與回傳的錯誤留下記錄。
- **before 的兩條告警不必拿掉。** 它們是告警 3、4 的原型，API 的錯誤率仍是客服與產品負責人看到的症狀。問題是只有它們；剛上線的內部小工具、沒有 SLO 與業務指標時，這兩條就夠用。

## 重點整理

- 告警從提醒有沒有在預定時間後 1 小時內送出出發：SLI 是 `sent / (sent + failed + expired + missed)`，錯誤預算用多窗口的消耗速度告警，每個告警連到 runbook。
- 「預定時間已過、仍未寄出」補的是 SLI 晚到的 1 小時：期限內先響，期限後由錯誤預算接手；goroutine 與 heap 的告警看服務自己輸出的兩個指標。
- 指標回答現在多糟（標籤只有 `result`），日誌回答那一次發生了什麼，追蹤回答時間花在哪；裝置 token 不進日誌與追蹤，活動與報名者不進指標的標籤。
- 韌性的工具依失敗的樣子選：重試、限流、逾時都有，拆成獨立行程就是 bulkhead；斷路器要擋的事已經被擋住、又要多一種結果，所以不加，條件改變再加。
- 關機時 `Shutdown` 在關伺服器之後、關資料庫之前送出緩衝裡的追蹤與指標；全部共用一個期限，期限到了的那一次仍會丟掉最後一段。
