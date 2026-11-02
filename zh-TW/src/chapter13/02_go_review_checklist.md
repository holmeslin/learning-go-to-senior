# Go 程式碼的 review 檢查清單

## 本集目標

review Go 程式碼時，手上有一份依本書各章排好的清單，逐項問一遍；也分得出哪些問題工具已經擋下、不必再人工看，哪些只有人看得出。

## 情境

本章〈[先看設計，再看細節](01_design_before_details.md)〉裡，reviewer 對候補名單 PR 留下設計意見 R1，作者把遞補移進 `Event`。方向對了，接下來逐項檢查細節。憑感覺看，每次看的東西都不一樣；清單的用處，是讓每個 PR 都被問到同樣的問題。

這個 PR 的 CI 是綠的：格式、`go vet`、`go test -race` 與 lint 都過了。綠燈只代表工具問得出來的都沒問題。

## 清單

每項一行，只問判斷，不重講；要看理由，點連結。編號是下面對照用的。

1. **套件與 API。** 這段程式放在對的套件與層嗎？見第 1 章〈[套件切分的依據（依職責，不依型別）](../chapter1/01_package_by_responsibility.md)〉。
2. **套件與 API。** 新增的匯出名稱（變數、欄位、方法也算）是你願意長期承諾的嗎？既有匯出的簽名有沒有被改掉？見第 1 章〈[匯出名稱與 API 穩定性](../chapter1/04_exported_api_stability.md)〉。
3. **套件與 API。** 名稱說得出它在做什麼嗎，從呼叫端讀起來順嗎？見第 1 章〈[套件命名與 `util` 陷阱](../chapter1/02_package_naming.md)〉（談套件與匯出名稱；本書沒有專談區域變數命名，這是最接近的一集）。
4. **錯誤。** 呼叫端分得出所有需要不同處理的結果嗎？文件寫明會回傳哪些錯誤了嗎？見第 2 章〈[錯誤是 API 契約的一部分](../chapter2/07_error_contract.md)〉。
5. **錯誤。** 錯誤只在一層處理，其他層加上下文就回傳嗎？見第 2 章〈[在哪一層處理、哪一層只回傳](../chapter2/03_handle_at_boundary.md)〉。
6. **抽象。** 介面由使用端定義，只含用到的方法嗎？見第 3 章〈[在使用端定義介面](../chapter3/01_define_interface_at_consumer.md)〉。
7. **資料層。** 一個用例改好幾筆資料時，交易範圍由用例決定嗎？見第 4 章〈[交易邊界放在哪裡](../chapter4/02_transaction_boundary.md)〉。
8. **測試。** 新行為的每種輸入（失敗、重複、額滿）都有測試，而且斷言了回傳什麼錯誤，不是只有成功路徑嗎？見第 5 章〈[覆蓋率的意義與限制](../chapter5/11_coverage_meaning_limits.md)〉。
9. **測試。** 新增的每個分支都有測試走過嗎？見第 5 章〈[覆蓋率的意義與限制](../chapter5/11_coverage_meaning_limits.md)〉。
10. **測試。** 並行與時間相關的測試靠 `testing/synctest` 或注入的時鐘，不靠 `time.Sleep` 嗎？見第 5 章〈[測時間與並行：用 `testing/synctest`，還是自己注入時鐘](../chapter5/08_synctest_or_clock.md)〉。
11. **並行。** 每個 goroutine 都有人負責等它結束嗎？見第 6 章〈[goroutine 的生命週期歸誰管](../chapter6/01_goroutine_ownership.md)〉。
12. **並行。** 鎖裡有沒有呼叫會阻塞、或會呼叫外部程式碼的函式？見第 6 章〈[channel 與 mutex 怎麼選](../chapter6/07_channel_or_mutex.md)〉。
13. **效能。** 說「這樣比較快」的 PR 有量測嗎？見第 7 章〈[先量測，再最佳化](../chapter7/01_measure_before_optimizing.md)〉。
14. **runtime。** 有沒有動 `GOGC` 這類 runtime 參數？有量測的根據嗎？見第 8 章〈[什麼時候該調 `GOGC`，什麼時候別動](../chapter8/02_when_to_tune_gogc.md)〉。
15. **可觀測性。** 日誌的變數寫成欄位，沒有拼進訊息嗎？見第 9 章〈[結構化日誌：`log/slog` 的欄位與 handler 怎麼選](../chapter9/01_structured_logging.md)〉。
16. **韌性。** 每個對外的呼叫都有逾時，而且是從呼叫端的 `ctx` 往下算嗎？見第 10 章〈[逾時預算](../chapter10/01_timeout_budget.md)〉。
17. **安全。** 來自使用者的輸入，在入口限制大小、拒絕未知欄位並驗證了嗎？見第 11 章〈[輸入驗證](../chapter11/01_input_validation.md)〉。
18. **建置。** 動到 `go.mod` 時，一次只升一個相依嗎？見第 12 章〈[`go.mod` 與相依升級](../chapter12/06_go_mod_dependency_upgrade.md)〉。

## 工具已經看過的，不必再看

清單裡沒有「縮排對不對」「有沒有沒用到的變數」，因為工具會擋，reviewer 不用逐行去找：

- `gofmt`：排版與縮排。
- `go vet`：`Printf` 的格式與引數對不對（`printf`）、把含鎖的值傳來傳去（`copylocks`）這類編譯器不報的可疑寫法。這兩個工具前作教過：〈[`gofmt` 與 `go vet`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/08_gofmt_vet.html)〉。
- `go test -race`：測試跑得到的資料競爭。
- `golangci-lint` 的預設規則：沒檢查的錯誤（`errcheck`）、沒用到的程式（`unused`）、白費的賦值（`ineffassign`），以及 `staticcheck` 找得到的常見誤用。本書的設定與每一條排除，見第 12 章〈[`golangci-lint` 與 CI](../chapter12/05_golangci_lint_ci.md)〉。

它們的共同點：不必看意圖，就能判斷對錯。設計對不對、名稱有沒有說明意圖、錯誤的語意、並行的正確性、測試有沒有測到行為，工具判斷不了，所以 lint 不取代 review。

候補名單 PR 的 `before/` 就是例子：`go vet` 與 `go test -race` 本來就是綠的（建置流程每次都跑）；`.golangci.yml` 排除 `before/`，只因為它刻意寫壞。把排除那一行拿掉、其他設定不變，預設規則對它也沒有意見：

```bash
sed '/\/before\//d' .golangci.yml > /tmp/lint-before.yml
go tool -modfile=tools/lint/go.mod golangci-lint run -c /tmp/lint-before.yml ./examples/chapter13/01_design_before_details/before/...
```

執行結果（某一次，macOS arm64、Go 1.27.0、golangci-lint 2.15.0，在 repo 根目錄執行）：

```text
0 issues.
```

## 套用到候補名單 PR

R1 屬於第 1 項，本章〈[先看設計，再看細節](01_design_before_details.md)〉已經先處理。R2 到 R7 各落在清單的哪一項，下面逐一指出；意見怎麼寫，本章〈怎麼寫 review 意見〉會談，要不要擋，本章〈什麼時候該擋、什麼時候可以放行〉會談。`Register` 同時碰到兩項：

```go,ignore
{{#include ../../../examples/chapter13/01_design_before_details/before/event.go:register}}
```

`Register` 一開頭呼叫的 `normalize`（`validate.go`）去掉名稱前後的空白，結果是空的就回傳 `ErrEmptyName`；`MaxWaitlist` 是候補名單的人數上限，下面會看到。

- **第 4 項（錯誤）：R2。** 名額已滿、排進候補的那條路徑回傳 `nil`，和報名成功一模一樣。呼叫端分不出兩種結果，也就沒辦法回不同的狀態碼；作者的測試還把這個 `nil` 當成預期。
- **第 8 項（測試）：R4。** 重複報名的檢查只看 `e.confirmed`，已經在候補名單的人再報名一次，會被再排進去。作者測了重複報名，但只測正式名單。覆蓋率報告也看不出來：缺的是一個檢查，不是一行沒跑到的程式。

遞補與通知在 `PromoteNext`：

```go,ignore
{{#include ../../../examples/chapter13/01_design_before_details/before/waitlist.go:promote}}
```

- **第 12 項（並行）：R3。** `e.mu` 還沒放開就呼叫 `e.notifier.Notify(nx)`。通知很慢時，`Register` 與 `Cancel` 都卡在同一把鎖上。`-race` 不會報，因為沒有人在沒同步的情況下讀寫同一份資料，只是有人在鎖裡等太久。
- **第 3 項（名稱）：R7。** `cf`、`wl`、`nx` 要猜才知道是什麼，本章〈[先看設計，再看細節](01_design_before_details.md)〉逐行看到的意見裡也有這一則。這一項問的是同一個問題，只是範圍小到區域變數；清單連到的那一集談的是套件與匯出名稱。工具也只幫得上一半：`varnamelen` 量的是名稱長度與作用範圍，不是有沒有說明意圖，第 12 章〈[`golangci-lint` 與 CI](../chapter12/05_golangci_lint_ci.md)〉把規則全開時它是最吵的幾條之一，本書沒開。

候補上限在 `MaxWaitlist`：

```go,ignore
{{#include ../../../examples/chapter13/01_design_before_details/before/waitlist.go:limit}}
```

- **第 2 項（API）：R5。** 匯出的可變變數成了套件對外的承諾：所有活動共用同一個值，任何套件都能改；清單連到的那一集談的是函式、方法與欄位，匯出的變數原則相同。同一項也問得到新增的 `PromoteNext` 與 `Notifier`，每個新匯出的名稱都要問「願意長期承諾嗎」。
- **第 9 項（測試）：R6。** 沒有任何測試取消的是候補者本人。`go test -coverprofile` 的報告會把 `Cancel` 裡處理候補名單的那兩行標成沒執行過。本書的 CI 沒有跑覆蓋率，第 12 章〈[`golangci-lint` 與 CI](../chapter12/05_golangci_lint_ci.md)〉建議的做法是只報告、不當門檻，所以要有人去看那份報告。

其餘各項：第 5、6、11 項過關（錯誤只在 handler 轉成狀態碼，`Notifier` 由使用端定義、只有一個方法，PR 沒有開 goroutine），第 10 項過關（測試沒有 `time.Sleep`），第 7、13、14、16 到 18 項沒有對象（沒有資料庫、沒有效能或 runtime 的主張、沒有對外連線的程式、沒有新的輸入入口、沒有動 `go.mod`）。第 15 項看得到取消端點多了一行 `fmt.Printf` 的除錯輸出，它沒有 R 編號：那是作者送出前就該自己清掉的東西，本章〈作者的自我審查〉會談。

## 取捨

- 清單是提問，不是過關條件。每一項過關，只代表沒看到那一類問題；清單也問不出新的設計問題，R1 是先看設計才看見的，第 1 項只是把同一個問題再問一次。
- 清單太長沒人用。這 18 項依本書的章節排，是為了每項都連得到理由，不是 review 的順序（順序見本章〈[先看設計，再看細節](01_design_before_details.md)〉）。團隊該把自己踩過的坑加進去，把一直用不上的刪掉。
- 十幾行的小改動不必逐項問，掃一眼標題就好。熟悉的程式碼，有經驗的人憑直覺就會看到；清單對新人，和對不熟悉的領域（例如不常寫的並行）最有用。
- 工具與人的界線會移動。團隊若定了規矩（例如不許有匯出的套件變數），可以改成 lint 規則，把那一項從清單拿掉；代價是規則有維護成本，第 12 章〈[`golangci-lint` 與 CI](../chapter12/05_golangci_lint_ci.md)〉談過。
- 「CI 綠、清單也過」不等於可以合併。哪些問題該擋，本章〈什麼時候該擋、什麼時候可以放行〉會談。

## 重點整理

- 清單依本書的章節排成 18 項，每項一行、連到談那個判斷的集，review 時逐項問，不憑感覺。
- 格式、`Printf` 的格式字串、沒檢查的錯誤、資料競爭由 `gofmt`、`go vet`、`-race` 與 lint 擋下，reviewer 不必逐行找。
- 候補名單 PR 的 R2 到 R7，沒有一則被擋下的工具攔住：R2 對到第 4 項、R3 第 12 項、R4 第 8 項、R5 第 2 項、R6 第 9 項、R7 第 3 項。
- 工具綠燈只代表可以機械判斷的部分沒問題；錯誤契約、持鎖呼叫外部、匯出的可變變數、缺的測試與名稱，要靠人看。
- 清單是提問，不是過關條件；它是提醒，不替代設計判斷。
