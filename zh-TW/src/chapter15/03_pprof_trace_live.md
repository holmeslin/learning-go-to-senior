# 用 `pprof` 與 trace 在現場診斷

## 本集目標

事故進行中、現場還在的時候，判斷先收什麼證據：重啟之前先收 goroutine 與 heap profile，隔一陣再收一次，用 `go tool pprof -top -base` 比較差異，從堆疊找出 goroutine 卡在哪一行；收完，才重啟。

## 情境

提醒 worker 的事故進行到 T+60 分：報名服務的記憶體持續上升，T+40 分容器被終止過一次，重啟後又上升，報名 API 仍然大量 5xx。事故指揮官決定關掉 worker 的設定開關、重啟服務（理由見本章〈[先止血，再找原因](02_mitigate_first.md)〉，分工見本章〈[事故中的角色分工](01_incident_roles.md)〉）。重啟之前，調查者（提醒 worker 的作者）有幾分鐘可以收證據。

工具本身不重講：診斷端點與 `curl` 收 profile，見第 9 章〈[`/debug/pprof` 與診斷端點](../chapter9/06_debug_pprof.md)〉；goroutine 洩漏的長相，見第 6 章〈[goroutine 洩漏怎麼發生、怎麼在測試裡抓到](../chapter6/09_goroutine_leak.md)〉；trace，見第 7 章〈[`runtime/trace`](../chapter7/04_runtime_trace.md)〉。這集談事故中的順序與判斷。

## 常見寫法

記憶體曲線爬到上限，最快的動作是重啟。下面是假設的版本，不是這次事故的紀錄：

```text
維護者：記憶體又快到上限了，先重啟，等一下再看原因。
（服務重啟，記憶體曲線掉回去，錯誤率短暫下降。）
（調查者在重啟之後去收 goroutine profile。）
```

範例在 `examples/chapter15/03_pprof_trace_live/`：`before/` 是事故版本的送出程式，`after/` 是修正版。`go run ./examples/chapter15/03_pprof_trace_live/after/cmd/slowpush -mode before` 在本機啟動慢速的假供應商與不斷送出提醒的 worker，診斷端點掛在獨立的 `ServeMux`、只聽 `127.0.0.1:18150`；剛啟動的狀態，就等於重啟之後。以下的輸出都來自同一台機器的同一次重現：`uname -sm` 是 `Darwin arm64`，`go version` 是 `go version go1.27.0 darwin/arm64`；重現腳本在 repo 外，每條指令照列。

執行結果（某一次）：

```text
$ curl -s 'http://127.0.0.1:18150/debug/pprof/goroutine?debug=1' | head -n 1
goroutine profile: total 7
```

## 問題在哪

重啟清掉三樣東西：卡住的 goroutine、它們占著的連線與記憶體，還有「增加得多快」的趨勢，留下的只有症狀。上面只有個位數的 goroutine，沒有任何堆疊可看；這次事故在 T+40 分被動重啟過一次，現場已經清過一回。之後只能猜：供應商變慢？資料庫連線不夠？GC 壓力？每個猜測都說得通，沒有一個能證實。要等記憶體再爬上來才有新的證據，不知道要等多久，5xx 也持續；供應商若先恢復，卡住的 goroutine 會自己結束，現場就沒了。

## 較好的做法

重啟已經決定了，所以收證據有時間盒：幾分鐘，由便宜的先收，每份都用 `curl -o` 存成檔案，檔案在調查者的機器上，不會隨重啟消失。第一次快照：

```bash
curl -s -o goroutine1.debug1.txt 'http://127.0.0.1:18150/debug/pprof/goroutine?debug=1'
curl -s -o goroutine1.pb.gz http://127.0.0.1:18150/debug/pprof/goroutine
curl -s -o heap1.pb.gz 'http://127.0.0.1:18150/debug/pprof/heap?gc=1'
curl -s -o goroutine1.debug2.txt 'http://127.0.0.1:18150/debug/pprof/goroutine?debug=2'
```

`debug=1` 把堆疊相同的 goroutine 併成一組、標出數量；`debug=2` 是每個 goroutine 的完整堆疊與等待狀態；`.pb.gz` 留給 `go tool pprof`；`heap` 加 `gc=1`，讓這個請求先做一次 GC，數字才反映現在。下面是 `debug=1` 檔裡含 `Sender.post` 的那一組，與 `debug=2` 檔裡屬於它的一個 goroutine；省略了其他組、位址與絕對路徑、參數，以及中間的 `net/http` 框架（`...`）：

執行結果（某一次）：

```text
240 @ 0x104a8d008 0x104a68268 ... 0x104a94394
#	net/http.(*persistConn).roundTrip	transport.go:3069
...
#	before.Sender.post	sender.go:44

goroutine 21 [select]:
net/http.(*persistConn).roundTrip(...)
	transport.go:3069
...
before.Sender.post(...)
	sender.go:44
created by before.Sender.SendAll in goroutine 1
	sender.go:32
```

一整批 goroutine 停在同一行：`sender.go:44`，`post` 呼叫 `Client.Do` 的地方，狀態是 `select`，在等回應。`created by` 指出它們都是 `SendAll` 的 `sender.go:32` 開出來的。這兩處就是要改的程式：

```go,ignore
{{#include ../../../examples/chapter15/03_pprof_trace_live/before/sender.go:send}}
```

隔一陣（這裡是 12 秒），用同樣的指令再收一次 `goroutine2.*` 與 `heap2.pb.gz`。單看一次分不出「本來就有」和「正在增加」，兩份用 `-base` 比：只留第二份減第一份，`-focus` 只留經過 `Sender.post` 的呼叫鏈。`go tool pprof` 的輸出只留表格，表頭以上各行都省略了；heap 的 `-focus=dialConn` 只留 worker 的 HTTP client，因為示範的假供應商與 worker 在同一個行程，heap 裡也有伺服器那一邊的配置：

執行結果（某一次）：

```text
$ grep -h '^goroutine profile' goroutine1.debug1.txt goroutine2.debug1.txt
goroutine profile: total 1207
goroutine profile: total 2407
$ go tool pprof -top -cum -nodecount=4 -focus='Sender\.post' -base goroutine1.pb.gz goroutine2.pb.gz
      flat  flat%   sum%        cum   cum%
         0     0%     0%        240 20.00%  github.com/holmeslin/learning-go-to-senior/examples/chapter15/03_pprof_trace_live/before.Sender.post
         0     0%     0%        240 20.00%  net/http.(*Client).Do (inline)
         0     0%     0%        240 20.00%  net/http.(*Client).do
         0     0%     0%        240 20.00%  net/http.(*Client).send
$ go tool pprof -sample_index=inuse_space -top -cum -nodecount=4 -focus=dialConn -base heap1.pb.gz heap2.pb.gz
      flat  flat%   sum%        cum   cum%
         0     0%     0%     1.51MB 23.12%  net/http.(*Transport).dialConn
         0     0%     0%     1.51MB 23.12%  net/http.(*Transport).dialConnFor
         0     0%     0%     1.51MB 23.12%  net/http.(*Transport).startDialConnForLocked.func1
       1MB 15.41% 15.41%        1MB 15.41%  bufio.NewReaderSize (inline)
```

第二次大約是第一次的兩倍；增加的 goroutine 有五分之一（20%）經過 `Sender.post`，其餘是同一批請求在 HTTP client 裡的 `readLoop`、`writeLoop`，與示範的假供應商那一邊。增加的記憶體在 `dialConn` 底下，是每個卡住的請求各占著的一條連線與它的緩衝區。

堆疊已經指出原因，trace 只在堆疊解釋不了時才收。這裡收 2 秒，補一個問題：這些 goroutine 有沒有在執行？`go tool trace` 開的本機網頁，`/goroutines` 的表格每列是「起點函式、數量、總執行時間」，下面只留兩列：

執行結果（某一次）：

```text
$ curl -s -o trace.out 'http://127.0.0.1:18150/debug/pprof/trace?seconds=2'
$ BROWSER=true go tool trace -http=127.0.0.1:18152 trace.out &
$ curl -s http://127.0.0.1:18152/goroutines | tr -d '\n\t' | sed -e 's#</tr>#\n#g' | sed -e 's#<[^>]*>#|#g' -e 's/||*/|/g' -e 's/^[ |]*//' -e 's/[ |]*$//' | grep -E 'Sender.post|writeLoop'
github.com/holmeslin/learning-go-to-senior/examples/chapter15/03_pprof_trace_live/before.Sender.post|530|3.944767ms
net/http.(*persistConn).writeLoop|530|834.88µs
```

兩組數量一樣多，執行時間加起來只有幾毫秒，錄的卻是 2 秒：它們在等，不在算。結論和堆疊一致，但 trace 說不出卡在哪一行，所以事故中堆疊先收。證據存好，記錄者記下「profile 已收」，指揮官才說重啟。

修正是兩件事：每個請求有逾時，同時進行的請求有上限（上限的做法見第 6 章〈[`errgroup` 的並行上限](../chapter6/03_errgroup_limit.md)〉，逾時怎麼沿呼叫鏈分配、從哪一層定下，見第 10 章〈[逾時預算](../chapter10/01_timeout_budget.md)〉）。測試用假的 `http.RoundTripper` 取代網路，在 `testing/synctest` 的 bubble 裡跑，2 秒的逾時瞬間走完；`SendAll` 回傳時若還有 goroutine 卡著，bubble 結束時 `synctest.Test` 會 panic。上限的測試是 `TestSendAllRespectsLimit`，完整程式碼在 `examples/chapter15/03_pprof_trace_live/after/`：

```go,ignore
{{#include ../../../examples/chapter15/03_pprof_trace_live/after/sender.go:sender}}
```

```go,ignore
{{#include ../../../examples/chapter15/03_pprof_trace_live/after/sender_test.go:timeout}}
```

這兩個測試放到 `before/` 會失敗：before 的 `SendAll` 立刻回傳 nil，沒有逾時的錯誤，bubble 結束時還有卡住的 goroutine，`synctest` 直接 panic。同樣的指令對 `-mode after` 啟動的修正版再收一次，兩次快照的 goroutine 總數相同，`-base` 的差異是空的：

執行結果（某一次）：

```text
$ grep -h '^goroutine profile' goroutine1.debug1.txt goroutine2.debug1.txt
goroutine profile: total 17
goroutine profile: total 17
$ go tool pprof -top -nodecount=4 -base goroutine1.pb.gz goroutine2.pb.gz
      flat  flat%   sum%        cum   cum%
```

## 取捨

- **收證據也有成本。** `debug=1` 與 `.pb.gz` 用的 goroutine profile 只在開始與結束時短暫停住世界，`debug=1` 的檔案大小幾乎不變；`debug=2` 隨 goroutine 數量成長，第二次的檔案約是第一次的兩倍（`wc -c`），而且 Go 1.27.0 的 `runtime.Stack(buf, true)` 在格式化全部堆疊的整段期間讓世界停住。所以先收 `debug=1`，需要 `created by` 與等待狀態時再收 `debug=2`。
- **CPU profile 與 trace 放最後。** 它們只在收的期間有成本（第 9 章〈[`/debug/pprof` 與診斷端點](../chapter9/06_debug_pprof.md)〉談過），trace 比 profile 貴，CPU profile 又看不到在等的 goroutine（第 7 章〈[`runtime/trace`](../chapter7/04_runtime_trace.md)〉談過）。這次 CPU 不忙，所以沒有收 CPU profile。
- **heap 看不到 goroutine 的堆疊。** goroutine 自己占的記憶體不在 heap profile 裡，profile 是取樣的，`gc=1` 還要多做一次 GC。記憶體上升的主因，要看 goroutine 的數量。
- **診斷端點事故中也只能從內部存取。** 不要為了方便把管理 port 對外開（第 9 章〈[`/debug/pprof` 與診斷端點](../chapter9/06_debug_pprof.md)〉談過存取控制）；存下來的檔案有函式名稱與路徑，當成敏感資料。
- **時間盒要有人守。** 每多收一分鐘，報名 API 多失敗一分鐘；到了時間，不管收到多少都重啟。
- **直接重啟什麼時候是對的。** 原因已經確定（同樣的症狀、已知的原因與修正），或影響大到每分鐘都等不起；至少先收 `debug=1`，幾秒就好，事後才有東西可對照。
- **修正也有代價。** 範例把上限與逾時寫死成常數，正式環境要從設定讀（第 12 章〈[設定與環境變數](../chapter12/08_config_env.md)〉）。逾時太短，慢但正常的回應也會失敗；上限太小，批次拖長（最壞是批次筆數除以上限、無條件進位，再乘以逾時，上面第二個測試是 20 則 6 秒）；逾時不保證供應商沒有處理，所以請求帶冪等鍵（第 10 章〈[冪等性](../chapter10/03_idempotency.md)〉）。把 worker 拆成獨立部署是另一件事，本章〈行動項目與追蹤〉會談。

## 重點整理

- 重啟會清掉卡住的 goroutine 與 heap；已經決定重啟時，先用幾分鐘收證據，每份存成檔案。
- 由便宜的先收：goroutine `debug=1`、`.pb.gz`、heap（`gc=1`），`debug=2` 與 trace 視需要；兩次之間隔一陣。
- 單看一次分不出「本來就有」和「正在增加」；`go tool pprof -top -base` 比兩份，`-focus` 只留 `debug=1` 指出的那一組。
- 堆疊回答「卡在哪一行」，trace 回答「有沒有在執行」；事故中堆疊先收。
- 修正是逾時加並行上限，測試用假的 `RoundTripper` 與 `synctest`；同樣的指令對修正版再收，goroutine 總數不再上升。
