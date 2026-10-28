# 連線池要設多大、設錯會怎樣

## 本集目標

查詢卡住、或資料庫的連線被用光時，知道先查哪裡（沒關的 `rows`、沒結束的交易）；並能依資料庫的連線上限與服務的副本數，刻意設定 `SetMaxOpenConns`、`SetMaxIdleConns`、`SetConnMaxLifetime`，再用 `db.Stats()` 觀察。

## 情境

你的訂單服務有個背景工作，定期用風控規則掃描訂單，找出第一筆符合的，交給人工覆核。規則寫在 Go 裡，所以 `Store.Find` 把訂單一筆一筆讀出來交給 `match` 判斷，找到就回傳。服務跑 4 個副本，共用一個最多允許 100 條連線的資料庫（數字是舉例）。

`*sql.DB` 是連線池、沒讀完的 `rows` 會占住連線，這些前作〈[`Query` 與 `Rows`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter15/04_query_rows.html)〉與〈[context 與連線池](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter15/08_context_pool.html)〉已經講過，這集只談池子怎麼設、卡住時從哪裡找原因。交易沒結束同樣會占住連線，本章〈交易邊界放在哪裡〉談過；deadline 該由誰設，本章〈查詢逾時該設在哪一層〉談過。

## 常見寫法

在迴圈裡找到就 `return`，沒有關 `rows`；`sql.Open` 之後什麼都不設：

```go,ignore
{{#include ../../../examples/chapter4/04_pool_size/before/orders.go:find}}
```

這樣寫不容易被測試抓到：找不到的時候一路讀到最後一列，`rows` 會自動關閉。只有在迴圈中途回傳（找到了、`Scan` 失敗），連線才不會歸還。連線池全用預設值：連線數不設上限，閒置的只留 2 條。完整程式碼在 `examples/chapter4/04_pool_size/before/`。

## 問題在哪

下面的 `Example` 各自開一個只屬於自己的記憶體 SQLite（共享快取、名稱唯一，池子裡的每條連線才看得到同一份資料；`newDB` 的參數是 `SetMaxOpenConns` 的值）。先看池子很小的情況，這裡用 1，讓現象一次出現。`// Output:` 由 `go test` 逐字比對：

```go,ignore
{{#include ../../../examples/chapter4/04_pool_size/before/example_test.go:small}}
```

- `Find` 回傳了，`in use` 卻是 1：沒有人在用，那條連線被沒關的 `rows` 占著，不會歸還。
- 下一個查詢要連線，池子的上限是 1，只能排隊，`wait count: 1` 記下這次排隊。deadline 一到，以 `context deadline exceeded` 結束。錯誤訊息裡沒有任何一個字提到 `rows`，從呼叫端看，只是「查詢逾時」。
- 池子設 10，只是撐到第 10 次洩漏，之後所有查詢都卡住。

池子不設上限呢？

```go,ignore
{{#include ../../../examples/chapter4/04_pool_size/before/example_test.go:unlimited}}
```

不會卡住，也沒有錯誤：三次呼叫開了三條連線，三條都還占著。洩漏沒有消失，只是從「服務卡住」變成「資料庫的連線被一條條吃掉」。伺服器型資料庫都有連線上限（SQLite 沒有，所以這裡看不到被拒絕的那一步），而且 4 個副本共用同一個上限，一個副本漏光，其他副本也連不上。

## 較好的做法

兩件事，目的不同。第一件是把資源關好，這是修 bug：

```go,ignore
{{#include ../../../examples/chapter4/04_pool_size/after/orders.go:find}}
```

和 before 只差一行 `defer rows.Close()`。有了它，每一條離開路徑都歸還連線；讀完後自動關閉不衝突，`Close` 可以重複呼叫。迴圈結束後檢查 `rows.Err()`，讀到一半失敗才不會被當成「找不到」。

第二件是刻意設定池子，這是安全網：

```go,ignore
{{#include ../../../examples/chapter4/04_pool_size/after/orders.go:open}}
```

- `maxOpen` 由呼叫端傳入（`main` 從部署設定讀進來）。`Open` 拒絕 0 與負數，因為 `database/sql` 把它們當成不設上限，正是 before 的預設值。
- 閒置上限跟著 `maxOpen`。預設只留 2 條，尖峰過後多出來的連線被關掉，下次又得重新連線；`db.Stats()` 的 `MaxIdleClosed` 會記下被關掉的次數。
- `SetConnMaxLifetime` 讓連線定期換新，避開被防火牆、負載平衡器默默切斷的老連線。

同樣的情境，這次的 `// Output:`：

```go,ignore
{{#include ../../../examples/chapter4/04_pool_size/after/example_test.go:example}}
```

池子一樣只有一條，但 `Find` 之後 `in use` 是 0，同樣的查詢馬上拿到連線，`wait count` 是 0。測試把這件事鎖住了：`TestFindReturnsConnection` 對三條離開路徑（找到、讀完、`Scan` 失敗）各連續呼叫三次，只有一條連線也過得去，並檢查 `InUse` 與 `WaitCount` 都是 0；`TestOpenSetsPool` 同時占住三條連線再歸還，驗證閒置上限和池子一樣大。拿掉 `defer rows.Close()`，前一個測試就會失敗。完整程式碼在 `examples/chapter4/04_pool_size/after/`。

## 取捨

- **數字沒有通用答案。** 上限從資料庫的連線上限出發：100 條，扣掉維運工具、migration 要留的 20 條，再除以 4 個副本，每個副本最多 20 條。擴成 8 個副本，同一個數字就會超過資料庫的上限，所以它要跟著部署設定走，不要寫死在程式裡。
- **太小會排隊，太大會擠掉別人。** 池子小於並行的查詢數，請求排隊、延遲上升，`WaitCount` 一路累加；池子大，副本一多就超過資料庫的上限，尖峰時連新連線都被拒絕。先估並行的查詢數，上線後看 `db.Stats()` 有沒有排隊再調。
- **上限是安全網，不是修復。** before 的洩漏，池子設再大也只是晚一點發生。上限的價值是讓失敗留在自己的服務裡（等連線、逾時），而不是拖垮資料庫與其他副本。
- **閒置連線也占資料庫的名額。** `maxIdle` 設得和 `maxOpen` 一樣，是把名額留給這個副本；副本很多、流量很不平均時，要想想值不值得。`SetConnMaxLifetime` 要比中間設備切斷連線的時間短，確切的數字看你的網路環境。
- 這些統計要長期收集與告警，第 9 章會談。

before 在這些情況下反而可以接受：

- 單機的小工具、一次性的腳本：同時只有一兩條連線，預設值就夠，記得 `defer rows.Close()` 就好。
- 小池子本身不是錯。單一檔案的 SQLite、或要保護很脆弱的資料庫時，`SetMaxOpenConns(1)` 是刻意的選擇；前提是資源都有歸還。

## 重點整理

- 連線被占住最常見的原因是沒關的 `rows` 與沒結束的交易；用 `defer rows.Close()`、`defer tx.Rollback()` 保證每條離開路徑都歸還。
- 症狀是查詢逾時，錯誤訊息卻不提 `rows`；沒人在用時 `InUse` 不是 0、`WaitCount` 一直增加，就是線索。
- 池子不設上限，洩漏就變成資料庫的連線被一條條吃掉，連其他副本都受影響；`maxOpen` 至少是 1，從「資料庫上限 ÷ 副本數」估起。
- `SetMaxIdleConns` 預設只留 2 條、`SetConnMaxLifetime` 避開被切斷的老連線，三個設定都要刻意選，不要沿用預設值。
- 上限是安全網不是修復，數字取決於資料庫與部署，沒有通用答案。
