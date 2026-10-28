# 排程器（GMP）怎麼看

## 本集目標

用 G、M、P 說明 goroutine 怎麼被排到執行緒上、`GOMAXPROCS` 限制的是什麼，並判斷 CPU 密集的工作該開多少並行：對齊 `runtime.GOMAXPROCS(0)`，不是越多越快。

## 情境

上傳服務收到一批檔案，要對每個檔案的內容算 SHA-256 來去重複。雜湊是純 CPU 的工作，沒有網路也沒有磁碟。goroutine 前作教過：〈[goroutine](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter9/01_goroutine.html)〉；`benchstat` 的讀法第 7 章〈[`b.Loop` 與 `benchstat`：怎麼比較前後結果才可信](../chapter7/02_b_loop_benchstat.md)〉談過。這集談的是 goroutine 遠多於 CPU 核心時，runtime 在做什麼。

## 常見寫法

goroutine 很便宜，每個 blob 開一個，`WaitGroup` 等全部做完：

```go,ignore
{{#include ../../../examples/chapter8/01_scheduler_gmp/before/sum.go:sum}}
```

結果是對的，下面〈較好的做法〉的測試拿它當比對基準。完整程式碼在 `examples/chapter8/01_scheduler_gmp/before/`。

## 問題在哪

runtime 的排程器有三種角色：

- **G**：goroutine，要執行的工作。
- **M**：作業系統的執行緒，真正在 CPU 上跑程式碼的是它。
- **P**：執行 Go 程式碼的許可，自己帶一條本地的待執行 G 佇列。M 拿到一個 P 才能跑 Go 程式碼；本地佇列放不下的 G 進全域佇列。

`GOMAXPROCS` 就是 P 的數量，也是同一時刻最多幾個 M 在跑 Go 程式碼。它不限制 goroutine 的數量，也不限制執行緒的數量。CPU 密集的 goroutine 不會停下來等 I/O，只會輪流占用 P：開了十萬個，同時在跑的仍然只有 P 那麼多個，其餘都在佇列裡排隊。

`GODEBUG=schedtrace=100` 讓 runtime 每 100 ms 往 stderr 印一行排程器的狀態。`burst` 是一支 `main`，對 20 萬個 64 KiB 的 blob 呼叫 `Sum`（CI 只編譯、不執行它）。輸出在 macOS（darwin/arm64，Apple M4 Max，14 個 P）上取得，省略了每行的 `schedticks` 欄位、程式啟動時 0 ms 的那一行，以及第 3 行之後的輸出；`before` 與 `after` 是同一支程式，只差 import 的套件。

執行結果（某一次）：

```text
$ go build -o burst_before ./examples/chapter8/01_scheduler_gmp/before/burst
$ GODEBUG=schedtrace=100 ./burst_before >/dev/null
SCHED 113ms: gomaxprocs=14 idleprocs=0 threads=15 spinningthreads=0 needspinning=1 idlethreads=0 runqueue=5676 [ 54 74 32 53 47 251 26 72 67 16 56 72 117 0 ]
SCHED 224ms: gomaxprocs=14 idleprocs=0 threads=15 spinningthreads=0 needspinning=1 idlethreads=0 runqueue=4953 [ 125 109 100 122 37 30 124 115 105 27 61 102 3 38 ]
```

- `gomaxprocs=14`：P 的數量。`idleprocs=0`：沒有閒著的 P，CPU 已經被用滿。
- `threads=15`：M 的數量，可以比 P 多，這集不展開。
- `runqueue=5676`：全域佇列裡等著的 G。方括號裡是 14 個 P 各自的本地佇列長度，長度從 0 到兩百多不等，多數 P 的本地佇列都不是空的。

`idleprocs=0` 是重點：CPU 已經滿了，排隊的 goroutine 不可能讓它更滿，多開的只是在等。每個等待中的 goroutine 還占著自己的堆疊。`after` 做同樣的事，佇列是這樣：

執行結果（某一次）：

```text
$ GODEBUG=schedtrace=100 ./burst_after >/dev/null
SCHED 125ms: gomaxprocs=14 idleprocs=0 threads=15 spinningthreads=0 needspinning=1 idlethreads=0 runqueue=0 [ 0 0 0 0 0 0 0 0 0 0 0 0 0 0 ]
SCHED 216ms: gomaxprocs=14 idleprocs=0 threads=15 spinningthreads=0 needspinning=1 idlethreads=0 runqueue=0 [ 0 0 0 0 0 0 0 0 0 0 0 0 0 0 ]
```

兩邊的 `idleprocs` 都是 0，只有佇列長度不同。多開的 goroutine 有沒有換到吞吐量，要用 `benchstat` 看。兩個目錄的 `pkg:` 不同，所以加 `-ignore pkg`（第 7 章〈[`b.Loop` 與 `benchstat`：怎麼比較前後結果才可信](../chapter7/02_b_loop_benchstat.md)〉談過）；`Sum/blobs=N` 是一批 N 個 16 KiB 的 blob，表頭的 `-14` 是這台機器的 `GOMAXPROCS`。

執行結果（某一次）：

```text
$ go test -run='^$' -bench=. -count=10 ./examples/chapter8/01_scheduler_gmp/before > before.txt
$ go test -run='^$' -bench=. -count=10 ./examples/chapter8/01_scheduler_gmp/after > after.txt
$ go run golang.org/x/perf/cmd/benchstat@v0.0.0-20261009192801-be2c69fb417e -ignore pkg before=before.txt after=after.txt
goos: darwin
goarch: arm64
cpu: Apple M4 Max
                    │    before    │                after                 │
                    │    sec/op    │    sec/op     vs base                │
Sum/blobs=1000-14     690.0µ ± 15%   794.3µ ± 21%        ~ (p=0.190 n=10)
Sum/blobs=100000-14   78.13m ± 71%   61.23m ± 10%  -21.62% (p=0.019 n=10)
geomean               7.342m         6.974m         -5.01%
```

要看的是 `before` 有沒有因為並行數多而比較快：沒有。`blobs=1000` 判為 `~`，`blobs=100000` 反而是 `after` 較快。`±` 很大的欄位（這次 `before` 的 71%）不穩，只能看方向，不能把百分比當結論。

## 較好的做法

CPU 密集的工作，並行數對齊 P 的數量：開 `runtime.GOMAXPROCS(0)` 個 goroutine，各自從共用的計數器領下一個編號，做完再領，直到編號用完：

```go,ignore
{{#include ../../../examples/chapter8/01_scheduler_gmp/after/sum.go:foreach}}
{{#include ../../../examples/chapter8/01_scheduler_gmp/after/sum.go:sum}}
```

`GOMAXPROCS(0)` 只讀不改，回傳目前的值。這和第 6 章〈[什麼時候需要 worker pool，什麼時候每個任務一個 goroutine 就夠](../chapter6/06_worker_pool_or_goroutine_per_task.md)〉的判斷不衝突：那一集的上限由下游的容量決定，這裡由 CPU 決定。用 `errgroup` 的 `SetLimit(runtime.GOMAXPROCS(0))` 效果相同，這集只用標準庫。

測試要證明並行數「跟著」`GOMAXPROCS`，而不是剛好等於這台機器的值：把它設成 1、2、4，同時執行的 `fn` 的最大值必須分別是 1、2、4。`synctest` 的假時鐘讓每個 `fn` 都「睡」一秒而不用真的等（第 5 章〈[測時間與並行：用 `testing/synctest`，還是自己注入時鐘](../chapter5/08_synctest_or_clock.md)〉談過）。測試改了全域設定，所以用 `t.Cleanup` 還原，也不呼叫 `t.Parallel`；呼叫 `GOMAXPROCS` 還會關掉 runtime 的自動調整，只該出現在測試裡，細節本章〈容器內的 `GOMAXPROCS`〉會談：

```go,ignore
{{#include ../../../examples/chapter8/01_scheduler_gmp/after/sum_test.go:follows}}
```

測試不斷言這台機器的 CPU 數，也不斷言時間。輸出不變由 `Example` 鎖住，另有測試讓 `after` 與 `before` 吃同樣的輸入、結果必須一致：

```go,ignore
{{#include ../../../examples/chapter8/01_scheduler_gmp/after/sum_test.go:example}}
```

benchmark 的準備工作放在迴圈外，兩個版本寫法相同：

```go,ignore
{{#include ../../../examples/chapter8/01_scheduler_gmp/after/sum_bench_test.go:bench}}
```

## 取捨

- **I/O 密集的工作不受這個限制。** goroutine 在等網路時由 runtime 停放，不占用 P，一個 P 可以服務很多個在等的 goroutine，所以開得多是合理的；這種工作的並行數看下游的容量。檔案 I/O 與阻塞式 syscall 會占住執行緒，本章〈阻塞、syscall 與排程〉會談。
- **混合的工作要拆開看。** 一個任務一半算、一半等，並行數對齊 P 會讓 P 在等待時閒著；先分清楚時間花在算還是花在等，再決定，必要時量測。
- **對齊 P 等於把 CPU 全部拿走。** 同一個行程裡的其他工作（其他請求的 handler）共用這些 P；批次吃滿之後，它們的延遲會變長。要留餘裕就取 `GOMAXPROCS(0)` 的一部分，超載時怎麼辦第 10 章〈負載卸載與降級〉會談。
- **before 在批次很小時沒有問題。** 幾十個工作、偶爾才跑一次，一個 goroutine 一個工作最容易讀；問題出在量大，而且上限沒人管。
- **`GOMAXPROCS` 本身怎麼決定**，在容器裡會跟著配額，本章〈容器內的 `GOMAXPROCS`〉會談。要看每個 goroutine 在哪個 P 上跑、等了多久，用第 7 章〈[`runtime/trace`](../chapter7/04_runtime_trace.md)〉。

## 重點整理

- G 是 goroutine，M 是執行緒，P 是執行 Go 程式碼的許可；`GOMAXPROCS` 是 P 的數量，限制同時跑 Go 程式碼的執行緒數，不限制 goroutine 數。
- CPU 密集的工作開再多 goroutine，多出來的只是在 `schedtrace` 的佇列裡排隊（`idleprocs=0`、`runqueue` 變長），不會讓吞吐量增加。
- CPU 密集工作的並行數對齊 `runtime.GOMAXPROCS(0)`；I/O 密集的工作等待時不占 P，並行數看下游的容量。
- 測試用 `GOMAXPROCS` 設不同的值（`t.Cleanup` 還原、不平行），斷言並行數跟著它，不斷言機器的 CPU 數。
