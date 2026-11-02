# 壓測與調校

## 本集目標

用書附的程式在本機壓測，依批次時間、goroutine、heap 與 profile 判斷哪個數字要調、哪個不用動。調校的目標是 SLO：預定時間後 1 小時內送出，不是並行上限越大越好。

## 情境

提醒服務的並行上限 20 與請求逾時 5 秒，是第 15 章事故之後憑感覺訂的：有上限、有逾時，事故就不會重演。事後檢討把「壓測沒有模擬慢的供應商」記成接受的風險，重新評估的時間是「下一次規劃壓測的時候」（第 15 章〈[行動項目與追蹤](../chapter15/05_action_items.md)〉），也就是上線之前。

供應商限制每秒 50 次，一場活動約 2 萬人，一批的下限是 20000 ÷ 50 = 400 秒，比 5 分鐘（300 秒）的掃描間隔長。`Run` 一批做完才等下一個 tick，`time.Ticker` 對來不及收的接收者會丟掉 tick，所以批次不會重疊：

```go,ignore
{{#include ../../../examples/chapter16/reminder/internal/dispatch/dispatch.go:run}}
```

壓測要模擬的就是這兩件事：供應商變慢、一批比間隔長。範例有三個行程：`cmd/slowprovider`（假供應商，旗標 `-delay`、`-rate429`，只聽 loopback）、`cmd/seed`（灌一場活動，旗標 `-registrants`）與服務本體。重現腳本在 repo 外，指令照列。平台是 `uname -sm` 的 `Darwin arm64`，`go version` 是 `go version go1.27.0 darwin/arm64`。規模縮成 1/100：200 位報名者代替 2 萬人；限流、並行上限、逾時與供應商的延遲都是真的數字，所以一批的下限是 200 ÷ 50 = 4 秒。

## 常見寫法

調校前的設定值，沒有量過：

```text
REMINDER_SEND_CONCURRENCY=20   並行上限
REMINDER_REQUEST_TIMEOUT=5s    每個請求的逾時
REMINDER_RATE_LIMIT=50         供應商的限制，外部給的
REMINDER_SCAN_INTERVAL=5m      掃描間隔
```

這組數字有理由：20 個同時在路上，不會像事故那樣沒有上限；5 秒夠一個正常的請求，最壞的一則也在關機期限內做完（本章〈[並行](05_concurrency.md)〉）。測試全部通過，因為它們用假的供應商與假的時間，問的是「恰好一次」「上限有擋住」，不問「這個數字夠不夠」。

## 問題在哪

腳本照這些指令跑第一批（`$W` 放執行檔、資料庫與日誌），供應商的延遲與 `REMINDER_SEND_CONCURRENCY` 由腳本的變數帶入。先等假供應商的啟動日誌再起服務，否則連線被拒的那段時間，掃到的提醒全部變成待重試。腳本每次印一個標頭加三行：批次時間（`reminder.scan.duration` 的 `sum`，秒）、內部 API 的各狀態數量、取樣到的 goroutine 與 heap 最大值（個與位元組）；`goroutine.txt` 只在文中引用，不貼。供應商的延遲試三種：20 毫秒、1 秒、6 秒（比逾時還慢），20 毫秒那一輪的輸出省略：

```bash
go build -o "$W/reminder" ./examples/chapter16/reminder/cmd/reminder   # slowprovider、seed 同
"$W/seed" -db "$W/reminder.db" -event 1 -registrants 200
"$W/slowprovider" -addr 127.0.0.1:9400 -delay 1s -rate429 0.1 2> "$W/provider.log" &
REMINDER_PROVIDER_URL=http://127.0.0.1:9400 REMINDER_SEND_ENABLED=true REMINDER_DB_PATH="$W/reminder.db" \
REMINDER_API_ADDR=127.0.0.1:8480 REMINDER_DIAG_ADDR=127.0.0.1:6460 \
REMINDER_SEND_CONCURRENCY=20 "$W/reminder" 2> "$W/service.log" &
curl -s -o cpu.pb.gz 'http://127.0.0.1:6460/debug/pprof/profile?seconds=3' &
curl -s -o goroutine.txt 'http://127.0.0.1:6460/debug/pprof/goroutine?debug=1'    # 就緒後 1 秒
curl -s http://127.0.0.1:6460/debug/metrics      # 每 0.2 秒一次，到 reminder.scan.duration count=1
curl -s http://127.0.0.1:8480/v1/events/1/reminder-counts
```

執行結果（某一次）：

```text
--- before-1s: registrants=200 provider_delay=1s concurrency=20
reminder.scan.duration count=1 sum=10.431861709
{"event_id":1,"counts":{"failed":0,"retrying":16,"sending":0,"sent":184}}
goroutines_max=94 heap_bytes_max=5203152
--- before-6s: registrants=200 provider_delay=6s concurrency=20
reminder.scan.duration count=1 sum=50.4143115
{"event_id":1,"counts":{"failed":0,"retrying":200,"sending":0,"sent":0}}
goroutines_max=90 heap_bytes_max=5598384
```

- **供應商快（20 毫秒），一批約 4 秒**，是限流決定的下限，並行上限沒有用上；調校後也一樣。
- **每則 1 秒，一批約是 2.5 倍。** 同時處理數約等於每秒完成數乘以延遲（第 10 章〈[負載卸載與降級](../chapter10/07_load_shedding.md)〉）：每則 1 秒，要維持每秒 50 則，得有 50 個同時在路上；上限 20 只有每秒 20 則。全規模約 1000 秒，是掃描間隔的三倍多，但還在 1 小時之內。
- **每則 6 秒，一則都沒送出。** 每個請求占滿 5 秒的逾時，200 則全部變成待重試，一批約是 12 倍。全規模是 20000 × 5 ÷ 20 = 5000 秒，約 83 分鐘，超過 1 小時：SLO 失守。
- **goroutine 與 heap 有上限。** 供應商從 1 秒慢到 6 秒，goroutine 峰值相近，heap 也沒有往上走，沒有第 15 章事故的樣子。3 秒的 CPU profile（`go tool pprof -top -cum -nodecount=40 cpu.pb.gz`，只留 `Duration` 一行、欄位標題與兩列）：

執行結果（某一次）：

```text
Duration: 3.03s, Total samples = 120ms ( 3.96%)
      flat  flat%   sum%        cum   cum%
         0     0% 58.33%       40ms 33.33%  github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/internal/dispatch.(*Dispatcher).deliver
         0     0% 58.33%       40ms 33.33%  modernc.org/sqlite/lib._sqlite3Step
```

CPU 幾乎閒置，批次的時間花在等供應商與等限流，不在算（選哪種 profile 見第 7 章〈[`pprof`：先看 CPU 還是 heap](../chapter7/03_pprof_cpu_or_heap.md)〉，現場收 profile 見第 15 章〈[用 `pprof` 與 trace 在現場診斷](../chapter15/03_pprof_trace_live.md)〉）。唯一指得出名字的是 `deliver` 裡的 sqlite 寫入，`Begin` 與 `Mark*` 各一次。它值不值得最佳化，先量它占多少（第 7 章〈[先量測，再最佳化](../chapter7/01_measure_before_optimizing.md)〉）。benchmark 用 `b.Loop`，在檔案資料庫上量一則的兩次寫入（第 7 章〈[`b.Loop` 與 `benchstat`：怎麼比較前後結果才可信](../chapter7/02_b_loop_benchstat.md)〉）：

```go,ignore
{{#include ../../../examples/chapter16/reminder/internal/store/bench_test.go:bench}}
```

執行結果（某一次）：

```text
$ go test -run='^$' -bench=BeginAndMarkSent -count=10 ./examples/chapter16/reminder/internal/store > bench.txt
$ go run golang.org/x/perf/cmd/benchstat@v0.0.0-20261009192801-be2c69fb417e bench.txt
goos: darwin
goarch: arm64
pkg: github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/internal/store
cpu: Apple M4 Max
                    │  bench.txt   │
                    │    sec/op    │
BeginAndMarkSent-14   144.0µ ± 37%
```

每則約 0.14 毫秒，雜訊 ± 37%（提交寫進檔案，時間不穩）；限流讓兩則之間隔 20 毫秒，寫入約是它的 140 分之一。寫入就算降到零，一批也只短不到 1%，所以沒有要比較的改動，也沒有 `benchstat` 的前後比較。

## 較好的做法

依證據只改一個數字：並行上限 20 改成 50，寫回服務的預設值。依據是兩個算式：並行上限要不小於限流乘以供應商慢時的延遲，1 秒就是 50，之後瓶頸才換成限流；供應商每則都拖滿逾時的最壞情況，第一輪是 20000 × 5 ÷ 50 = 2000 秒，約 33 分鐘，留下近半小時給重試。

```go,ignore
{{#include ../../../examples/chapter16/reminder/internal/config/config.go:defaults}}
```

同樣的指令，拿掉 `REMINDER_SEND_CONCURRENCY=20`，用執行檔的預設值：

執行結果（某一次）：

```text
--- after-1s: registrants=200 provider_delay=1s concurrency=default
reminder.scan.duration count=1 sum=5.000442167
{"event_id":1,"counts":{"failed":0,"retrying":20,"sending":0,"sent":180}}
goroutines_max=184 heap_bytes_max=6299520
--- after-6s: registrants=200 provider_delay=6s concurrency=default
reminder.scan.duration count=1 sum=20.999522083
{"event_id":1,"counts":{"failed":0,"retrying":200,"sending":0,"sent":0}}
goroutines_max=180 heap_bytes_max=6612880
```

- 供應商快時沒有差別；每則 1 秒，一批回到限流的下限（4 秒加最後一個請求的 1 秒）；每則 6 秒，一批約縮到 0.4 倍，全規模約 2000 秒，在 1 小時之內。供應商慢到逾時，還是一則都送不出去，這件事調校救不了。
- **代價是 goroutine。** 進行中的一則占 3 個 goroutine（送出的呼叫端一個、HTTP 連線的讀與寫各一個），`goroutine.txt` 裡 `deliver`、`readLoop`、`writeLoop` 三組各剛好是並行上限那麼多個；峰值約是調校前的 2 倍。heap 的最大值同一組設定重跑也差 1 MB 上下，看不出和並行上限有關。
- **關機期限的檢查仍成立。** 算式裡沒有並行上限：進行中的送出同時各自逾時，最壞還是一個請求逾時加查詢逾時（第 10 章〈[逾時預算](../chapter10/01_timeout_budget.md)〉）。批次進行到一半送 `SIGTERM`，50 則同時在路上、都在等逾時：

執行結果（某一次）：

```text
--- stop-6s: registrants=200 provider_delay=6s concurrency=default, SIGTERM 1s after ready
exit code 0, seconds from SIGTERM to exit: 4.8
shutdown deadline passed lines: 0
retrying 50
no record 150
```

約一個請求逾時（5 秒）就結束，遠小於預設的 30 秒；進行中的 50 則記為待重試，沒有停在寄送中，沒開始的 150 則沒有紀錄，重啟後會再挑到。`TestLoadDefaults` 隨預設值更新，關機期限的下限（5s + 3×5s + 5s）沒變。

## 取捨

- **結果只在這台機器、這組參數下成立。** 縮小 1/100 與「乘以 100」假設吞吐量不變；沒量的有全規模的 `Due` 一次載入 2 萬筆、真實網路與 TLS、供應商真正的延遲分布。固定延遲的假供應商量得出「上限夠不夠」，量不出「逾時多短會砍掉正常的慢回應」，所以逾時沒動；那要上線後看 `reminder.send` 的 span 時間（本章〈[可觀測性與韌性](06_observability_resilience.md)〉）。逾時也在關機期限的算式裡，動它就要一起動 `ShutdownTimeout`。
- **目標是 SLO，不是數字越大越好。** 上限加到 100（另跑一輪，輸出省略），一批的時間和 50 時幾乎一樣，goroutine 峰值卻更高，只多了代價。限流是外部給的上限：一批的下限 400 秒調不掉，掃描間隔縮短也不會讓它更快，所以兩個都沒動。每小時最多 50 × 3600 = 180000 則，一場 2 萬人用掉九分之一；同時開始的場次一多，靠的是跟供應商談額度，不是調參數。
- **調校會牽動別的數字。** 本章〈[可觀測性與韌性](06_observability_resilience.md)〉的 goroutine 告警，起始值 200 約是閒置時的 6 倍，是上限 20 時訂的，那時正常批次的峰值約是閒置的 3 倍；上限 50 的正常批次峰值已是閒置的 6 倍多，和門檻幾乎一樣高，供應商一慢，稍有波動就會響。門檻要跟著並行上限走，取峰值的 2 倍，改成 400，仍遠低於失控時的數量。
- **before 在供應商穩定快速時是對的。** 每則都在 0.4 秒內，20 ÷ 0.4 不小於 50，限流就是瓶頸（本章〈[並行](05_concurrency.md)〉的算式），調高只會多出 goroutine。調校是證據指出瓶頸之後才做，不是上線前的儀式。

## 重點整理

- 壓測要模擬供應商慢與一批比間隔長：縮小規模、真實的限流與延遲，先等假供應商就緒再起服務，同樣的指令調校前後各跑一次。
- 並行上限不小於限流乘以供應商慢時的延遲；再大只多出 goroutine，批次時間不變。
- 限流與掃描間隔不是調校的對象：外部給的上限調不掉，調校的目標是 SLO，最壞情況的第一輪要在 1 小時內做完。
- 沒有熱點就不要最佳化：profile 指出 CPU 閒置、benchmark 量出寫入遠小於限流的間隔，所以只改了一個設定值。
