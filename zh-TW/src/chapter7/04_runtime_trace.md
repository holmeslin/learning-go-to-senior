# `runtime/trace`

## 本集目標

遇到「延遲很高，CPU 卻不忙」，知道 CPU profile 在這裡幫不上忙，改用 `runtime/trace` 看 goroutine 在等什麼，並用轉成 pprof 的阻塞 profile 當佐證。

## 情境

登入服務用 PBKDF2（標準庫 `crypto/pbkdf2`）雜湊密碼。尖峰時登入的延遲一路升高，監控上整台機器卻只用到一兩顆核心。`go test -bench` 與 `pprof` 的基本用法前作教過：〈[benchmark 與 `b.Loop`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/12_benchmark.html)〉、〈[`pprof`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/appendix2/d_pprof.html)〉；`benchstat` 本章〈[`b.Loop` 與 `benchstat`：怎麼比較前後結果才可信](02_b_loop_benchstat.md)〉談過，從 benchmark 收 CPU profile 本章〈[`pprof`：先看 CPU 還是 heap](03_pprof_cpu_or_heap.md)〉談過。這集談 CPU profile 看不到的那一半。

## 常見寫法

雜湊很吃 CPU，怕登入尖峰把所有核心吃滿、擠壞其他請求，就把雜湊全交給一個 goroutine，其他人透過 channel 排隊。第 6 章〈[channel 與 mutex 怎麼選](../chapter6/07_channel_or_mutex.md)〉談過這種單一擁有者的形狀：

```go,ignore
{{#include ../../../examples/chapter7/04_runtime_trace/before/hasher.go:hasher}}
```

功能完全正確：下面〈較好的做法〉的測試與 `Example` 對 before 也全部通過。完整程式碼在 `examples/chapter7/04_runtime_trace/before/`。

## 問題在哪

先把症狀寫成 benchmark：一次同時進來 `burst` 個登入請求，量全部處理完要多久。before 的版本多一行 `defer h.Close()`，其餘相同。

```go,ignore
{{#include ../../../examples/chapter7/04_runtime_trace/after/hasher_bench_test.go:bench}}
```

症狀的數字放在〈較好的做法〉的 `benchstat` 表裡，看 before 那一欄：一次請求的時間是基準；32 個請求若能分到多顆核心，不該是它的 32 倍，但 before 的 `burst=32` 大約就是 32 倍，像是一個接一個處理。名稱尾巴的 `-14` 是這台機器的 `GOMAXPROCS`。

**CPU profile 看不出原因。** CPU profile 只取樣正在執行的執行緒，排隊的 goroutine 沒有用到 CPU，從原理上就不會出現在裡面。輸出省略了 `Time:` 行。

執行結果（某一次）：

```text
$ go test -o before.test -run='^$' -bench='HashBurst/burst=32$' -benchtime=30x -cpuprofile=before.cpu ./examples/chapter7/04_runtime_trace/before
$ go tool pprof -top -nodecount=4 before.test before.cpu
File: before.test
Type: cpu
Duration: 1.11s, Total samples = 900ms (81.18%)
Showing nodes accounting for 840ms, 93.33% of 900ms total
Showing top 4 nodes out of 33
      flat  flat%   sum%        cum   cum%
     700ms 77.78% 77.78%      700ms 77.78%  runtime.pthread_cond_signal
      60ms  6.67% 84.44%      190ms 21.11%  crypto/internal/fips140/pbkdf2.Key[go.shape.interface { BlockSize int; Reset; Size int; Sum []uint8; Write  }]
      60ms  6.67% 91.11%       60ms  6.67%  crypto/internal/fips140/sha256.blockSHA2
      20ms  2.22% 93.33%      100ms 11.11%  crypto/internal/fips140/sha256.(*Digest).Sum
```

排行榜是 runtime 喚醒執行緒的函式加上一點 `sha256`，沒有任何一行寫著「在等」。唯一的線索是 `Total samples` 佔 `Duration` 的比例：它以牆上時間為分母，多顆核心同時在忙會超過 100%，這裡不到 100%，平均連一顆核心都沒填滿。哪個函式排第一會隨平台與負載不同，別拿它下結論。

**trace 看得到等待。** `go test -trace` 記錄每一次 goroutine 的建立、阻塞與喚醒；`go tool trace` 開一個本機的網頁，首頁的文字說明了幾個入口。下面的數字，是對收好的 trace 執行 `go tool trace` 後，用 `curl` 抓下列頁面的文字讀出來的（20 個 burst、每個 32 個請求）：

- 〈View trace by proc〉：每顆邏輯處理器一條時間軸，標出每一刻在那顆處理器上跑的是哪個 goroutine（首頁說明）。這一頁由瀏覽器畫圖，文字裡看不到圖；它的資料（`/jsontrace?view=proc`）加總之後，雜湊 goroutine 共跑了約 610 ms，幾乎等於整段 trace 的長度（約 614 ms）：同一時刻只有它一個在做雜湊。它會在處理器之間移動，所以出現在五條軸上，其他處理器幾乎沒有東西在跑。
- 〈Goroutine analysis〉：依起點函式分組，每組顯示數量與執行時間。`before.New.func1`（那個雜湊 goroutine）只有 1 個，執行時間 610 ms；`WaitGroup.Go` 開出來的請求 goroutine 有 640 個，執行時間加起來不到 1 ms（占整個程式執行時間的 0.12%）。點進後一組的表格，其中 620 個 goroutine 的 `Block time (chan send)` 超過 `Execution time` 的十倍：時間都花在把工作送進 channel 的等待上。
- 〈Synchronization blocking profile〉：等 mutex、channel 這類同步操作所花的時間，依呼叫堆疊畫成圖（首頁說明）。

最後一項可以直接轉成 pprof 文字當佐證。`Type: delay` 的單位是 goroutine 被阻塞的時間，所有 goroutine 加總，所以可以遠大於牆上時間；輸出省略了 `Dropped` 行。

執行結果（某一次）：

```text
$ go test -o before.test -run='^$' -bench='HashBurst/burst=32$' -benchtime=20x -trace=before.trace ./examples/chapter7/04_runtime_trace/before
$ go tool trace -pprof=sync before.trace > before.sync.pprof
$ go tool pprof -top -nodecount=4 before.test before.sync.pprof
File: before.test
Type: delay
Showing nodes accounting for 12504.02ms, 100% of 12504.60ms total
Showing top 4 nodes out of 18
      flat  flat%   sum%        cum   cum%
 9467.75ms 75.71% 75.71%  9467.75ms 75.71%  runtime.chansend1
 2425.68ms 19.40% 95.11%  2425.68ms 19.40%  runtime.chanrecv1
  610.60ms  4.88%   100%   610.60ms  4.88%  sync.(*WaitGroup).Wait
         0     0%   100% 10068.43ms 80.52%  github.com/holmeslin/learning-go-to-senior/examples/chapter7/04_runtime_trace/before.(*Hasher).Hash
```

阻塞時間的八成落在 `(*Hasher).Hash`，主要是等著把工作送進那條 channel（`chansend1`），其次是等結果（`chanrecv1`）。`WaitGroup.Wait` 是 benchmark 自己在等。症狀有了名字：序列化點就是那一個 goroutine。

## 較好的做法

PBKDF2 的 `Key` 不共用任何狀態，同時呼叫是安全的，單一擁有者本來就不必要。想限制 CPU 的用量，用一個「並行上限」，而不是把並行數釘死成 1。直接在呼叫端的 goroutine 裡算：

```go,ignore
{{#include ../../../examples/chapter7/04_runtime_trace/after/hasher.go:hasher}}
```

行為不變由測試保證。同時 32 個人登入，每個人拿回的要等於自己的密碼直接交給 `pbkdf2.Key` 的結果，不能被別人的覆蓋；`Example` 則鎖住一組固定輸入的輸出：

```go,ignore
{{#include ../../../examples/chapter7/04_runtime_trace/after/hasher_test.go:concurrent}}
{{#include ../../../examples/chapter7/04_runtime_trace/after/hasher_test.go:example}}
```

兩個版本都用 `-count=10` 量，比較時加 `-ignore pkg`（本章〈[`b.Loop` 與 `benchstat`：怎麼比較前後結果才可信](02_b_loop_benchstat.md)〉談過），輸出因此沒有 `pkg:` 行。

執行結果（某一次）：

```text
$ go test -run='^$' -bench=. -count=10 ./examples/chapter7/04_runtime_trace/before > before.txt
$ go test -run='^$' -bench=. -count=10 ./examples/chapter7/04_runtime_trace/after > after.txt
$ go run golang.org/x/perf/cmd/benchstat@v0.0.0-20261009192801-be2c69fb417e -ignore pkg before=before.txt after=after.txt
goos: darwin
goarch: arm64
cpu: Apple M4 Max
                      │    before    │                after                │
                      │    sec/op    │   sec/op     vs base                │
HashBurst/burst=1-14     938.2µ ± 0%   935.2µ ± 0%        ~ (p=0.143 n=10)
HashBurst/burst=32-14   30.054m ± 0%   3.057m ± 1%  -89.83% (p=0.000 n=10)
geomean                  5.310m        1.691m       -68.16%
```

`burst=32` 的差異遠大於雜訊（`±`），`benchstat` 判為顯著；`burst=1` 只有一個請求，沒有東西可以並行，`~` 符合預期。阻塞 profile 也同步佐證（做法同上，省略 `Dropped` 行）：

執行結果（某一次）：

```text
$ go tool pprof -top -nodecount=4 after.test after.sync.pprof
File: after.test
Type: delay
Showing nodes accounting for 242.24ms, 100% of 242.24ms total
Showing top 4 nodes out of 12
      flat  flat%   sum%        cum   cum%
  162.44ms 67.05% 67.05%   162.44ms 67.05%  runtime.chanrecv1
   79.81ms 32.95%   100%    79.81ms 32.95%  sync.(*WaitGroup).Wait
         0     0%   100%    80.90ms 33.40%  github.com/holmeslin/learning-go-to-senior/examples/chapter7/04_runtime_trace/after_test.BenchmarkHashBurst
         0     0%   100%    79.81ms 32.95%  github.com/holmeslin/learning-go-to-senior/examples/chapter7/04_runtime_trace/after_test.BenchmarkHashBurst.func1
```

`Hash` 從阻塞的排行榜消失；剩下的是 benchmark 與 `testing` 自己的等待。

## 取捨

- **trace 比 profile 貴。** 它記錄每一個排程事件，檔案隨錄製的時間與 goroutine 數量成長：同一個 benchmark 錄 20 個 burst 約 5.5 萬位元組，錄 200 個約 46 萬位元組（`wc -c`，某一次），執行期間也有額外成本。所以只收短時間，`go test -trace` 搭配很小的 `-benchtime`。
- **正式環境偶發的問題用 flight recorder。** 不可能一直錄 trace，`runtime/trace` 的 `FlightRecorder` 只在記憶體裡保留最近一小段，等到偵測到異常（例如某次請求特別慢）才用 `WriteTo` 存下來，見 [`runtime/trace` 文件](https://pkg.go.dev/runtime/trace)。在正式環境怎麼取得診斷資料，第 9 章〈`/debug/pprof` 與診斷端點〉會談。
- **只要總量，不要時間軸，`-blockprofile` 更便宜。** 它取樣阻塞事件，和 trace 轉出的阻塞 profile 讀法一樣。鎖競爭怎麼量與怎麼縮小臨界區，本章〈常見熱點：配置、拷貝、鎖競爭〉會談。
- **排程器的細節只點到為止。** `GOMAXPROCS` 對結果的影響，第 8 章〈[排程器（GMP）怎麼看](../chapter8/01_scheduler_gmp.md)〉談過。
- **before 的動機是對的。** 限制雜湊吃掉多少 CPU 是正當的需求，錯在把上限釘成 1。上限該是接近核心數的 N，寫法見第 6 章〈[`errgroup` 的並行上限](../chapter6/03_errgroup_limit.md)〉。

## 重點整理

- 延遲高而 CPU 不忙時，CPU profile 看不到等待的 goroutine；改收 trace。
- `go test -trace` 收，`go tool trace` 看各 goroutine 在等什麼，`-pprof=sync` 把阻塞時間依堆疊轉成文字佐證。
- 序列化的修正要保持行為：用測試對照獨立算出的結果，再用 `benchstat` 比較。
- trace 的檔案隨時間變大、成本也高，只收短時間；偶發問題用 `trace.FlightRecorder`。
- 想限制 CPU 用量，設並行上限，不要把並行數釘成 1。
