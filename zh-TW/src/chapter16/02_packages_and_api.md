# 套件結構與 API

## 本集目標

判斷一個服務的套件要依什麼切、依賴往哪個方向、`internal` 該圈住多少、介面該放在哪一側；以及一支唯讀的內部 API 要先決定哪幾件事：路徑、回應、驗證、錯誤對應，還有為什麼不提供寫入。

## 情境

提醒服務除了掃描與送出，還要回答客服與產品負責人的一個問題：這場活動的提醒送到哪了？答案是各狀態的提醒數量，由一支內部 HTTP API 提供。資料歸屬與上線順序本章〈[設計文件](01_design_doc.md)〉決定過；寄送紀錄表與推播的錯誤分類，本章〈錯誤與資料層〉會談；各套件怎麼測，本章〈測試〉會談。這集只看套件怎麼擺、誰依賴誰，以及這支 API 的契約。

## 套件怎麼切

切法依職責，不依種類（第 1 章〈[套件切分的依據（依職責，不依型別）](../chapter1/01_package_by_responsibility.md)〉）：每個套件各有一個「為了誰、因為什麼而改」的理由，沒有 `models`、`handlers`。

```text
reminder/
├── service.go          package reminder：Service，組裝零件、決定關機順序
├── cmd/reminder/       package main：讀環境變數、接 signal
└── internal/
    ├── dispatch/       掃描與送出：並行上限、限流、重試
    ├── store/          寄送紀錄表、migration、狀態轉換
    ├── push/           推播客戶端、錯誤分類
    ├── api/            內部 HTTP API
    ├── diag/           診斷端點：healthz、readyz、pprof、指標快照
    ├── telemetry/      指標與追蹤：設定、快照、關機時送出
    ├── config/         環境變數與啟動時驗證
    └── demoreg/        報名服務那一側的示範 schema 與唯讀檢視
```

依賴方向，箭頭是 `import`，只列本專案內的套件：

```text
cmd/reminder → reminder、config
reminder     → dispatch、api、diag、telemetry、store、push、config
dispatch     → store、push
api          → store（只用型別與 ErrNotFound）
store、push、config、diag、telemetry、demoreg → 沒有
```

- 箭頭只從組裝的一側指向零件，排得出先後、沒有環（第 1 章〈[循環相依與依賴方向](../chapter1/07_import_cycle.md)〉）。第三方套件也有方向：`dispatch` 只依賴 OpenTelemetry 的 API，SDK 只出現在 `telemetry` 與 `reminder`。
- `dispatch` 與 `api` 互不認識，只透過 `store` 碰同一張表。這是刻意的：寫紀錄的只有掃描迴圈，`api` 只讀；兩邊若互相 import，誰是寫入者就說不清楚。
- `demoreg` 沒有人指向它：服務本身不 import，只有測試與示範資料用到它。

## `cmd` 只組裝，`Service` 才是整個服務

```go,ignore
{{#include ../../../examples/chapter16/reminder/cmd/reminder/main.go:main}}
```

`main` 只讀設定、接 signal、呼叫 `Service.Run`；signal 只在這裡處理，`Service` 看到的只有一個會被取消的 `context`。組裝與關機順序放在 `reminder` 套件：`package main` 不能被 import，而端對端測試要在行程內組起整個服務、又不啟動 `main`，所以需要一個測試呼叫得到的 `Service.Run`。

```go,ignore
{{#include ../../../examples/chapter16/reminder/service.go:service}}
```

欄位裡有三個接縫是為測試而匯出的：

- `Transport`：端對端測試跑在 `synctest` 的 bubble 裡，bubble 裡 socket I/O 不算 durably blocked，假時間不會前進，所以要能把網路換成假的 `RoundTripper`。
- `SpanExporter`：測試用記憶體的 exporter 檢查關機時送出了什麼；上線時在 `main` 接真的，`nil` 代表不送出。
- `Config` 的兩個位址可以是空字串：代表不啟動那個 HTTP 伺服器。`config.Load` 不會產生空字串，只有程式碼能設，原因同上，bubble 裡不能有真的 socket。

## `internal` 圈住多少

`reminder/internal/` 底下的套件，只有 `reminder/` 以下的程式碼 import 得到，也就是 `Service`、`cmd/reminder` 與測試（機制見第 1 章〈[什麼時候用 `internal`，什麼時候該公開](../chapter1/03_internal_or_public.md)〉）。圈住的理由是資料歸屬：本章〈[設計文件](01_design_doc.md)〉決定寄送紀錄表的 schema、migration 與寫入都歸提醒服務。`store` 若公開，日後任何一行 `import` 都能呼叫 `Begin` 寫這張表，繞過掃描迴圈的順序；`internal` 讓「只有掃描迴圈寫紀錄」由編譯器把關，不靠約定。對外的契約只剩 HTTP API、環境變數，和報名服務擁有的唯讀檢視。

代價在這本書裡長這樣：本章〈並行〉的 `before`、`after` 放在 `examples/chapter16/05_concurrency/`，在 `reminder/` 之外，import 不到 `dispatch`，只好自成不依賴服務的縮小版。

## 介面只有三個，都在使用端

```go,ignore
{{#include ../../../examples/chapter16/reminder/internal/dispatch/dispatch.go:sender}}
```

```go,ignore
{{#include ../../../examples/chapter16/reminder/internal/api/api.go:counter}}
```

`dispatch.Sender`、`api.Counter`、`diag.Snapshotter` 各由使用它的套件宣告，只列一個方法（第 3 章〈[在使用端定義介面](../chapter3/01_define_interface_at_consumer.md)〉）；`push.Client`、`*store.Store`、`*telemetry.Telemetry` 不知道它們存在，隱式實作自動滿足。判斷標準是第 3 章〈[介面何時是多餘的](../chapter3/03_redundant_interface.md)〉：跨越 I/O 邊界，而且測試要製造真實作做不出來的狀況。`Sender` 要能回暫時與永久的失敗，也要能占用一段（假）時間，才測得出並行上限；`Counter` 要能回逾時與不明的錯誤，還要斷言「輸入不合法時沒有去查資料庫」。

`dispatch` 拿的是 `*store.Store`，不是介面：資料層的測試用真的 in-memory sqlite，狀態轉換靠資料庫的唯一限制，替身得把那套規則重寫一遍，成為第二份要維護的實作。`Counter` 的假實作只是一個函式，`CountByState` 本身另有真 sqlite 的測試（`TestCountByState`）。

## 內部 API 的契約

| 請求 | 回應 |
| --- | --- |
| `GET /v1/events/{event_id}/reminder-counts` | `200`：活動編號與四個狀態的數量，四個狀態都在，沒有的是 0 |
| `event_id` 不是正整數的標準寫法 | `400`，不查資料庫 |
| 檢視裡沒有這場活動，也沒有任何紀錄 | `404` |
| 查詢逾時 | `503`，可以稍後重試 |
| 其他錯誤 | `500`，本文只有 `internal error`，原因寫進日誌 |
| 改變資料的方法 | `405`，`Allow: GET, HEAD` |

回應格式與錯誤用 `Example` 鎖住，輸出由 `go test` 逐字比對；`fakeCounter` 是 API 測試裡假的 `Counter`：

```go,ignore
{{#include ../../../examples/chapter16/reminder/internal/api/api_test.go:fake}}
```

```go,ignore
{{#include ../../../examples/chapter16/reminder/internal/api/example_test.go:example}}
```

```go,ignore
{{#include ../../../examples/chapter16/reminder/internal/api/api.go:handler}}
```

- **回應的內容由用途決定。** 客服要的是「送到哪了」，所以只有數量，沒有報名者與裝置 token。這支 API 沒有驗證（套件說明有 `shortcut:`，只靠內部網路，預設監聽 `:8080`），能回的東西就得是無害的。
- **404 與 0 要分得開。** 活動「存在」是檢視裡有它的報名者，或已有紀錄；存在但還沒到預定時間，回 `200`、四個數字都是 0，客服因此分得出「活動編號打錯」與「還沒開始寄」。
- **驗證只收標準寫法。** `+5`、`007` 用 `strconv.ParseInt` 讀得懂，但同一場活動會有好幾個網址，日誌與快取被打散，所以一律 `400`，訊息固定、不回顯輸入（`TestHandlerRejectsInvalidEventIDs`）。通過驗證的是 `int64`，`Counter` 只收這個型別，往下不必再檢查。第 11 章〈[輸入驗證](../chapter11/01_input_validation.md)〉談的是請求本文，順序是先正規化再檢查；路徑上的編號是資源的身分，所以不正規化，直接拒絕。
- **錯誤用 `errors.Is` 對應，不看訊息。** `Counter` 的文件註解寫明「沒有這場活動時，回傳的錯誤要包著 `store.ErrNotFound`」，這是 `store` 與 `api` 之間的錯誤契約（第 2 章〈[錯誤是 API 契約的一部分](../chapter2/07_error_contract.md)〉），`TestHandlerMapsErrorsToStatusCodes` 用多包一層的錯誤鎖住；狀態碼與本文則是 `api` 對外的另一份契約，500 不帶磁碟路徑這類內部訊息。503 是 handler 自己設的 `QueryTimeout` 到期，`store` 不設逾時。
- **只讀，而且 405 不是自己寫的。** `ServeMux` 的 pattern 只註冊了 `GET`，其他方法自動得到 `405` 與 `Allow`（`TestHandlerOffersNoWriteOperations`）。
- **API 與診斷端點是兩個 handler、兩個 listener。** pprof 不能和業務路由共用（第 9 章〈[`/debug/pprof` 與診斷端點](../chapter9/06_debug_pprof.md)〉）：診斷端點聽 `127.0.0.1:6060`，API 的 handler 對 `/debug/pprof/` 與 `/healthz` 回 `404`（`TestHandlerSetsAQueryDeadlineAndServesOnlyItsRoute`）。

## 哪裡會衝突

- **接縫是匯出的，承諾卻不大。** 第 1 章〈[匯出名稱與 API 穩定性](../chapter1/04_exported_api_stability.md)〉說匯出就是承諾，而上面三個接縫是 `reminder` 的匯出欄位。`reminder` 沒有放進 `internal`，別的 module 也 import 得到它，`Config` 的型別雖在 `internal`，外部程式碼仍可逐欄設值。但它的呼叫端只有 `cmd/reminder` 與端對端測試，同一個 repo、一次就能一起改，正是第 1 章〈[什麼時候用 `internal`，什麼時候該公開](../chapter1/03_internal_or_public.md)〉說公開反而合理的情況：服務的契約是 HTTP、環境變數與檢視，不是 Go 的識別字。要把接縫也藏起來，就把 `reminder` 一起移進 `internal`，代價是 `reminder/` 底下只剩 `cmd` 與 `internal`。
- **介面縮小了替身，沒有切斷 import。** `Sender` 的簽名用 `push.Notification`、`Counter` 的簽名用 `store.State`，所以 `dispatch` 仍 import `push`、`api` 仍 import `store`。想切斷就得在使用端另訂一份型別、再翻譯一次，這個翻譯層要測、要維護，換來的只是少一條 import；這裡不做。只有 `diag.Snapshotter` 的簽名純用標準庫，`diag` 才真的不依賴指標的實作。
- **只讀是為了守住唯一的寫入者。** 本章〈[設計文件](01_design_doc.md)〉說唯一限制擋得住重複、擋不住互搶；API 若能「重送」或「改狀態」，就成了第二個寄送者，同一件事再來一次。要不要補寄超過期限的提醒，是第 15 章〈[行動項目與追蹤](../chapter15/05_action_items.md)〉裡由產品負責人決定的行動項目 5，這支 API 不預設答案。只讀也是沒有驗證還撐得住的前提：先有寫入的需求，才輪到驗證與授權。

## 取捨

- `internal` 底下八個套件，換來每一層有自己的改動原因與測試方式（真的 sqlite、假的 `RoundTripper`、`httptest`），代價是讀一則提醒的路徑要跨好幾個套件。服務若只做「掃描、送出」一件事，一個 `main` 套件加幾個檔案也夠；這裡同時有資料層、對外 API 與診斷端點，各自的曝險與測試不同，才切開。
- `internal` 擋掉所有人：別的服務想重用 `push` 就得複製。真有第二個服務需要它時，再抽成獨立的 module 公開。
- 只有數量的 API 回答不了「某個人有沒有收到」。那類問題要看個人資料，得先決定誰能看、怎麼驗證，是另一支 API，不是在這支加欄位。
- `Counter` 的假實作只驗證翻譯（輸入驗證、錯誤到狀態碼）；「存在」的定義這類兩邊接縫上的行為，假實作看不到。`TestServiceServesHTTPAndTurnsUnreadyBeforeWaitingForInFlightSends` 在 bubble 外起真的伺服器、接真的 `store`，補上這一段。

## 重點整理

- 套件依職責切，依賴只從組裝的一側指向零件；`cmd` 只讀設定與接 signal，組裝放在測試呼叫得到的 `Service`。
- `internal` 把「只有掃描迴圈寫紀錄」交給編譯器把關；代價是同一本書裡的其他範例 import 不到服務。
- 介面定義在使用端，只放用到的一個方法；只有跨越 I/O 邊界、測試需要製造狀況時才值得有。介面縮小替身，不一定切斷 import。
- 內部 API 先決定用途、路徑、回應、驗證、錯誤對應；驗證只收標準寫法，錯誤用 `errors.Is` 對應，500 不帶內部訊息。
- 只讀守住唯一的寫入者；有沒有驗證與是否提供寫入，要一起決定。
