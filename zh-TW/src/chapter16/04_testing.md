# 測試

## 本集目標

判斷提醒服務的測試要放在哪幾層、各層用什麼替身、證明什麼；並寫出一個在行程內跑完整服務的端對端測試，知道它斷言哪些外部結果、刻意不斷言什麼。

## 情境

維護者每次提交都要知道三件事：到期的提醒有沒有恰好送出一次、供應商一直壞時 1 小時後有沒有放棄、收到結束訊號後進行中的送出有沒有做完。CI 沒有供應商，也不起容器。本章的規則照用：測試不連網路、不用固定 port、不用 `time.Sleep`（`synctest` 的假時間除外）、不啟動 `main`。服務由幾個套件組成，本章〈[套件結構與 API](02_packages_and_api.md)〉談過；這集看它們各用什麼測。

## 測試分幾層

每一層只證明自己這一層的事，替身只換掉這一層以外的東西：

| 層 | 證明什麼 | 替身 | 為什麼 |
| --- | --- | --- | --- |
| `store` | 唯一限制、狀態轉換、逾期記為失敗 | 沒有：每個測試一個真的 sqlite（記憶體） | 約束與 SQL 是資料庫的行為，fake 證明不了（第 4 章〈[測試資料層：fake 與真實資料庫](../chapter4/07_fake_or_real_database.md)〉）；`now` 是參數，不必假時間 |
| `push` | 錯誤分類、冪等鍵標頭、每個請求的逾時 | 假的 `http.RoundTripper`（一個函式） | 分類只看回應，不需要 socket；沒有 socket 才放得進 `synctest` |
| `dispatch` | 恰好一次、並行上限、限流、重試與放棄的時間點、取消 | 假的 `Sender`、真的 `store`、`synctest` | 要證明的是分類之後怎麼辦，回一個包著 `push.ErrTransient` 的錯誤就夠，不必造 503 |
| `api`、`diag` | 輸入驗證、錯誤對應的狀態碼、唯讀、readiness | 假的 `Counter` 與 `Snapshotter`，`httptest.NewRecorder` 跑組好的 handler | 路由與方法的對應（405、404）只有組好的 handler 測得到（第 5 章〈[要啟動真的 server，還是只測 handler](../chapter5/09_server_or_handler.md)〉） |
| `config` | 驗證表、錯誤訊息 | `Load` 收一個函式，測試傳 map 當環境變數 | 不碰全域環境，可以 `t.Parallel`；用 `t.Setenv` 就不行（第 12 章〈[設定與環境變數](../chapter12/08_config_env.md)〉） |
| 整個服務 | 供應商收到什麼、紀錄寫了什麼、`Run` 何時回傳 | 換掉網路，也不啟動 HTTP 伺服器 | 見下一段 |

時間有兩種角色，第 5 章〈[測時間與並行：用 `testing/synctest`，還是自己注入時鐘](../chapter5/08_synctest_or_clock.md)〉分別給了做法，這個服務兩種都用。`store` 的時間是資料（`due_at`、`next_attempt_at`），每個方法都收 `now`，資料層測試直接指定那一刻，不必進 bubble；`push`、`dispatch` 與整個服務的時間是「等」，用 `synctest`。

替身放在用它的那一層的測試檔裡，不另開 `pushtest` 這類套件：每個替身只有一個使用端，這是第 5 章〈[測試替身放在哪裡](../chapter5/04_where_test_doubles_live.md)〉說的、不必讓提供者附上 fake 與契約的情況。`Sender` 是定義在使用端的介面（第 3 章〈[在使用端定義介面](../chapter3/01_define_interface_at_consumer.md)〉），假的 `Sender` 就寫在 `dispatch` 自己的測試裡：

```go,ignore
{{#include ../../../examples/chapter16/reminder/internal/dispatch/dispatch_test.go:sender}}
```

`push` 的假 `RoundTripper` 與 `Example`，本章〈[錯誤與資料層](03_errors_and_data.md)〉貼過，這裡不重貼。同一個供應商在三處有三種替身：`push` 的測試造 HTTP 回應，`dispatch` 的測試直接回錯誤，端對端測試擺在最外面的網路邊界，讓 `push` 與 `dispatch` 都真的跑。

## 端對端測試

整個服務在行程內跑：真的 `config.Load`、真的 sqlite、真的 `Service.Run`，只有網路換成假供應商。它同時是 `http.RoundTripper`，記下每次呼叫的冪等鍵與假時間；`arrived` 與 `gate` 給不在 bubble 裡的那個測試用：

```go,ignore
{{#include ../../../examples/chapter16/reminder/service_test.go:provider}}
```

`harness` 把服務組起來，測試只從三個地方觀察：供應商收到的呼叫、寄送紀錄、`Run` 的回傳：

```go,ignore
{{#include ../../../examples/chapter16/reminder/service_test.go:harness}}
```

設定走 `config.Load`：預設值若通不過驗證，端對端測試先失敗。`run` 的 `ctx` 不從 `t.Context()` 衍生：第 6 章〈[goroutine 洩漏怎麼發生、怎麼在測試裡抓到](../chapter6/09_goroutine_leak.md)〉提過，`t.Context()` 在 bubble 結束前就會被取消，從它衍生的 `ctx` 會讓忘了停服務的測試照樣通過；改用 `context.Background()`，沒有人停的服務會讓 `synctest.Test` 報出殘留的 goroutine。

第一個測試斷言「恰好一次」。`time.Sleep(2 * time.Hour)` 在 bubble 裡是假時間，之後的每一次掃描都不再送已寄出的：

```go,ignore
{{#include ../../../examples/chapter16/reminder/service_test.go:once}}
```

第二個測試斷言取消之後的結果。本章〈並行〉會談的關機期限，在服務層就用它檢查：進行中的兩則做完並記為已寄出，`Run` 約 2 秒後、在關機期限內回傳，沒開始的沒有紀錄。

```go,ignore
{{#include ../../../examples/chapter16/reminder/service_test.go:shutdown}}
```

另外幾個測試：`TestServiceGivesUpAfterOneHour` 讓供應商持續回 503，59 分鐘仍待重試、61 分鐘記為失敗、之後請求數不變；`TestServiceDoesNotSendWhenSwitchIsOff` 斷言供應商 0 個請求、紀錄表 0 筆；`TestServiceRecordsAsFailedWhatItMissedWhileDown` 讓預定時間在 90 分鐘前，第一次掃描記為失敗、次數 0、不送。

**不斷言的東西**：

- 重試的每個時間點：`dispatch` 層用注入的 `Jitter` 斷言（`TestRunRetriesWithBackoffThenGivesUp`），端對端只看 59 與 61 分鐘兩個點。
- 並行上限與限流的數字：留給 `dispatch` 層，本章〈並行〉會談。
- goroutine 數量：不數，由 `synctest.Test` 在結束時檢查有沒有殘留。
- 真實時間、記憶體、壓測的數字：永遠不斷言。
- 日誌、指標、span 的內容與 HTTP 端點：另有測試，本章〈可觀測性與韌性〉會談。

## 為了測試而匯出的 API

- `Service.Transport`：假供應商要換掉網路。bubble 裡 socket I/O 不算 durably blocked，假時間不會前進，所以供應商不能是 `httptest.NewServer`。正式環境留空，用 `http.DefaultTransport`。
- `Config.APIAddr`、`DiagAddr` 為空字串代表不啟動那個伺服器，理由相同：bubble 裡不能有真的 listener。`Load` 不會產生空字串（空字串視為沒設），只有程式碼設得出來，`loadConfig` 在 `Load` 之後清掉；使用者設定的路徑走不到這個分支。
- `Service.SpanExporter`：測試用記憶體的 exporter 檢查關機時送出什麼，本章〈可觀測性與韌性〉會談。
- `dispatch.Options.Jitter`：注入亂數來源，才斷言得出重試的時間點，是第 10 章〈[重試、退避與 jitter](../chapter10/02_retry_backoff_jitter.md)〉的同一個接縫。
- `dispatch.Options.RetryBase`、`RetryMax`：匯出是為了讓測試把重試間隔縮短，不必等真實的退避；正式設定用預設值（5 分鐘與 30 分鐘），環境變數不提供這兩項。
- `store.Get` 與 `demoreg`：測試用來讀寄送紀錄、灌報名資料，`demoreg` 也給本章〈壓測與調校〉的灌資料工具；目前沒有正式程式碼呼叫 `Get`。除了 `Service` 的兩個欄位，這些都在 `internal/` 底下，不會變成別人依賴的 API。

## 不測什麼、怎麼避免不穩定

**不測的東西，要知道它們沒被證明：**

- 真的供應商。假供應商是對供應商行為的假設（503、429、很慢），不是供應商本身；供應商認不認 `Idempotency-Key`（本章〈[錯誤與資料層](03_errors_and_data.md)〉的假設），沒有東西能在這裡驗證；契約測試要真的實作也跑同一組（第 5 章〈[測試替身放在哪裡](../chapter5/04_where_test_doubles_live.md)〉），供應商不是自己的，做不到。
- 檔案資料庫。所有測試都用記憶體資料庫，`OpenDB` 設的 WAL 對它沒有作用（`journal_mode` 是 `memory`）；`dispatch` 的並行寫入走的是共享快取的鎖，不是檔案的鎖。
- `main` 與 signal。測試用 `cancel()` 代替 `SIGTERM`；`main` 只在 repo 外的重現腳本裡執行，本章〈壓測與調校〉會談。

**不穩定的來源，各擋一個**（第 5 章〈[不穩定（flaky）測試](../chapter5/10_flaky_tests.md)〉）：

- 共用狀態：每個測試自己一個資料庫，名稱取自測試名稱。`cache=shared` 不能省，連線池有多條連線，沒有它每條連線各開一個空的記憶體資料庫；測試結束就關，`-count=3` 時下一輪從空的開始。

```go,ignore
{{#include ../../../examples/chapter16/reminder/internal/store/store_test.go:newstore}}
```

- 時間：用假時間，不用 `time.Sleep`，否則結果取決於機器的快慢；要等某件事就 `synctest.Wait`，或用 channel。
- 網路與 port：沒有 socket。唯一用真 listener 的測試，位址是 `127.0.0.1:0`，實際位址從日誌讀出來。
- goroutine：`synctest.Test` 檢查，建置流程對本章再跑 `-race -count=3`。

## 取捨

- **沒有 `integration` tag，因為全部都在行程內。** sqlite 是純 Go、不必安裝，供應商是函式，沒有 port；第 5 章〈[整合測試與 build tags](../chapter5/05_integration_tests_build_tags.md)〉的判準是成本與環境，兩項都不成立，每個人的 `go test ./...` 都跑得到端對端測試。加了 tag，`go vet` 與編輯器預設看不到它；放在一般測試裡，`synctest.Test` 檢查殘留 goroutine（第 6 章〈[goroutine 洩漏怎麼發生、怎麼在測試裡抓到](../chapter6/09_goroutine_leak.md)〉）也每次 `go test` 都做。
- **記憶體資料庫夠用，是因為服務自己選了 sqlite。** 第 4 章〈[測試資料層：fake 與真實資料庫](../chapter4/07_fake_or_real_database.md)〉的「SQLite 不是正式環境」，在這裡縮成「記憶體不是檔案」。換成需要外部資料庫，`store` 的測試就得對正式用的那種資料庫跑，那一組才需要 tag。
- **`synctest` 與真的 server 湊在一起，換來兩個只為測試的接縫。** 假時間要求拿掉網路與 HTTP 伺服器，端對端測試因此看不到真的連線與伺服器關機的順序。這兩件事交給一個在 bubble 外的測試（`TestServiceServesHTTPAndTurnsUnreadyBeforeWaitingForInFlightSends`）：真的 socket、真的時間，用 channel 與日誌同步。它是端對端測試裡唯一靠紀律而不是 `synctest` 擋不穩定的地方。
- **兩種時鐘並用，資料層的每個方法都得收 `now`。** `dispatch` 傳的是 `time.Now()`，在 bubble 裡就是假時間，所以端對端測試能斷言 `UpdatedAt` 剛好是預定時間後 1 小時；資料層測試傳固定的 2030 年的時間，不必進 bubble，也能 `t.Parallel`。代價是每個 `store` 方法多一個參數；新增的方法忘了收 `now`、自己呼叫 `time.Now()`，資料層測試就指定不了時間。
- **每一層各測一次「恰好一次」。** `store` 證明第二次 `Begin` 回 `false`，`dispatch` 證明別人先寫了紀錄就不送，端對端證明跨掃描不重複。端對端只說「供應商收到兩次」，沒說哪一層錯；代價是規則改了要改三處。第 5 章〈[測什麼、不測什麼](../chapter5/01_what_to_test.md)〉的判斷照用：每一層測自己公開的行為。
- **只有一兩個套件、沒有時間與關機的邏輯時，端對端測試不值得維護**：各套件的測試加上手動跑一次就夠。

## 重點整理

- 每一層只證明自己的事：資料層對真的 sqlite，客戶端對假的 `RoundTripper`，掃描對假的 `Sender`，handler 用 `httptest.NewRecorder`；整個服務只換掉網路。
- 端對端測試只斷言外部看得到的：供應商收到什麼、紀錄寫了什麼、`Run` 何時回傳；重試的時間點、並行與限流留給 `dispatch` 層，goroutine 交給 `synctest.Test`。
- `synctest` 與 socket 互斥，所以只為測試匯出 `Transport` 與空位址，並把真的伺服器放進一個 bubble 外的測試；每個接縫都要在正文說明理由。
- 不穩定的來源各擋一個：每個測試自己的資料庫、假時間、沒有 socket、`-race -count=3`。
- 沒有被證明的東西要寫下來：真的供應商、檔案資料庫、`main` 與 signal。
