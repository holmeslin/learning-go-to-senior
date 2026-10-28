# 先量測，再最佳化

## 本集目標

看到「這段一定很慢」的直覺時，先走完一套流程：定義指標、寫貼近實際的 benchmark、用 profile 找熱點、一次只改一個地方、再量一次；也能分辨什麼時候可以不量就改。

## 情境

服務把一批存取日誌彙總成每條路由的請求數、錯誤數與總耗時。log shipper 重送時，同一個請求編號會出現不只一次，只算第一次。一批 20000 行處理得太慢，跟不上進來的速度。有人覺得「每行都要切欄位、轉數字，一定是這裡慢」，動手改寫。benchmark 與 pprof 的基本用法前作教過：〈[benchmark 與 `b.Loop`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/12_benchmark.html)〉、〈[`pprof`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/appendix2/d_pprof.html)〉。這集談的是調查的順序。

## 常見寫法

憑直覺最佳化：不用 `strings` 與 `strconv`，逐字元掃過每一行，自己把數字轉出來。

```go,ignore
{{#include ../../../examples/chapter7/01_measure_before_optimizing/before/logstat.go:parsefast}}
```

程式變長、變難讀，欄位數、位數、非數字這些邊界也得自己負責。完整程式碼在 `examples/chapter7/01_measure_before_optimizing/before/`；最佳化之前的版本（`strings.Cut` 加 `strconv`）留在同一個套件的 `SummarizeSimple`，只為了拿來比較，平常它就是版本紀錄裡的上一版。

## 問題在哪

**一、指標。** 問題是「跟不上進來的速度」，所以指標是處理一批所花的時間（`sec/op`）。換成記憶體或吞吐量，後面 benchmark 與 profile 看的東西也跟著換。

**二、benchmark。** 貼近實際：一批 20000 行，含重送的行，輸入固定、不靠亂數。兩個版本放在同一個 benchmark 輪流量，準備工作在迴圈外：

```go,ignore
{{#include ../../../examples/chapter7/01_measure_before_optimizing/before/bench_test.go:bench}}
```

`-count=10` 讓每個 benchmark 量十次，`benchstat` 才能分辨差異是不是雜訊；怎麼讀 `±` 與 p 值，本章〈`b.Loop` 與 `benchstat`：怎麼比較前後結果才可信〉會談。`-filter` 只留 `sec/op` 這張表（省略了 `B/op`、`allocs/op` 兩張表），輸出也省略了 `pkg:` 行：

```bash
go test -run='^$' -bench=. -benchmem -count=10 ./examples/chapter7/01_measure_before_optimizing/before > before.txt
go run golang.org/x/perf/cmd/benchstat@v0.0.0-20261009192801-be2c69fb417e -col /impl -row /lines -filter '.unit:sec/op' before.txt
```

執行結果（某一次）：

```text
goos: darwin
goarch: arm64
cpu: Apple M4 Max
      │ handrolled  │            simple             │
      │   sec/op    │   sec/op     vs base          │
20000   152.6m ± 2%   153.5m ± 1%  ~ (p=0.063 n=10)
```

`~` 是 benchstat 判斷沒有顯著差異：手寫解析犧牲了可讀性，卻量不出好處。就算某次 benchstat 把幾個百分點的差距判成顯著，這個量級也解釋不了一批為什麼要花這麼久。時間花在別的地方。

**三、profile。** 用 benchmark 收 CPU profile，不用為此改程式。`go test -cpuprofile` 會在目前目錄留下測試執行檔 `before.test` 與 `cpu.out`，兩個檔案都不要 commit。輸出省略了 `File`、`Type`、`Time`、`Showing`、`Dropped` 五行：

```bash
go test -run='^$' -bench='^BenchmarkSummarize$/^lines=20000$/^impl=handrolled$' -benchtime=3s -cpuprofile cpu.out ./examples/chapter7/01_measure_before_optimizing/before
go tool pprof -top -nodecount=4 before.test cpu.out
```

執行結果（某一次）：

```text
Duration: 3.23s, Total samples = 2910ms (90.14%)
      flat  flat%   sum%        cum   cum%
    1870ms 64.26% 64.26%     1870ms 64.26%  runtime.memequal
     640ms 21.99% 86.25%     2530ms 86.94%  slices.Index[go.shape.[]string,go.shape.string]
     210ms  7.22% 93.47%      210ms  7.22%  runtime.madvise
      40ms  1.37% 94.85%       40ms  1.37%  internal/runtime/atomic.(*UnsafePointer).Load
Total: 2.91s
```

`flat` 是函式自己花的時間，`cum` 連同它呼叫的函式；怎麼從排行榜挑出該改的那一層，本章〈`pprof`：先看 CPU 還是 heap〉會談。大部分時間在字串比對，來自 `slices.Index`，也就是 `slices.Contains`。用 `-list` 看是哪一行呼叫的，輸出只留迴圈開頭的幾行，省略了函式標頭（含檔案路徑）與其餘的行：

```bash
go tool pprof -list='before.summarize$' before.test cpu.out
```

執行結果（某一次）：

```text
      10ms       10ms    102:	for _, line := range lines {
         .          .    103:		e, ok := parse(line)
         .          .    104:		if !ok {
         .          .    105:			continue // 壞掉的行直接略過
         .          .    106:		}
         .      2.53s    107:		if slices.Contains(seen, e.id) {
```

手寫解析所在的 `parse` 那一行幾乎沒有時間，大部分時間在 `seen`：它是 slice，每一行都從頭掃描，n 行要比對約 n²/2 次，批次越大越慢。

## 較好的做法

流程是一個迴圈。上面做了指標、benchmark、profile，接著是第四步，**一次只改一個地方**：把 `seen` 換成 map。解析換回標準庫的寫法不算效能改動：第二步量過了，手寫版沒有好處，複雜度就沒有理由留下。

```go,ignore
{{#include ../../../examples/chapter7/01_measure_before_optimizing/after/logstat.go:summarize}}
```

最佳化不能改變行為，`Example` 與對照測試（拿最佳化之前的版本當標準答案）鎖住它：

```go,ignore
{{#include ../../../examples/chapter7/01_measure_before_optimizing/after/logstat_test.go:example}}
{{#include ../../../examples/chapter7/01_measure_before_optimizing/after/logstat_test.go:same}}
```

第五步，**再量一次**。after 的子 benchmark 也叫 `impl=simple`，名稱相同，`benchstat` 才能並排；`-filter` 讓 before 只留沒有手寫解析的那一列，比的就只有 `seen` 的改動；兩份檔案來自不同套件，`-ignore pkg` 讓它們不被分成兩張表（細節本章〈`b.Loop` 與 `benchstat`：怎麼比較前後結果才可信〉會談）。第一次執行時 `go run` 會先下載 benchstat，輸出省略了下載訊息：

```bash
go test -run='^$' -bench=. -benchmem -count=10 ./examples/chapter7/01_measure_before_optimizing/after > after.txt
go run golang.org/x/perf/cmd/benchstat@v0.0.0-20261009192801-be2c69fb417e -ignore pkg -filter '/impl:simple' before.txt after.txt
```

執行結果（某一次）：

```text
goos: darwin
goarch: arm64
cpu: Apple M4 Max
                                     │  before.txt   │              after.txt              │
                                     │    sec/op     │   sec/op     vs base                │
Summarize/lines=20000/impl=simple-14   153.499m ± 1%   4.757m ± 1%  -96.90% (p=0.000 n=10)

                                     │  before.txt  │              after.txt               │
                                     │     B/op     │     B/op      vs base                │
Summarize/lines=20000/impl=simple-14   2.175Mi ± 0%   2.715Mi ± 0%  +24.85% (p=0.000 n=10)

                                     │ before.txt  │             after.txt              │
                                     │  allocs/op  │  allocs/op   vs base               │
Summarize/lines=20000/impl=simple-14   70.03k ± 0%   70.16k ± 0%  +0.18% (p=0.000 n=10)
```

`sec/op` 顯著下降（p 值小、`n=10`），改善是真的；`B/op` 上升，是 map 的代價。配置次數幾乎沒變，用 `testing.AllocsPerRun` 斷言上限，時間不在測試裡斷言。`regexp` 內部用 `sync.Pool`，`-race` 下配置次數會有出入，所以斷言的是上限（每行少於 6 次，實際約 3.5 次），不斷言精確次數：

```go,ignore
{{#include ../../../examples/chapter7/01_measure_before_optimizing/after/logstat_test.go:allocs}}
```

## 取捨

- **明顯的演算法複雜度問題，不用量就可以改。** 迴圈裡對 slice 做 `Contains` 是 O(n²)，code review 看得出來，指出來就能改。先量測對付的是「憑直覺選錯地方」，不是拖延的藉口。批次只有幾十行時，這個 O(n²) 也不是熱點。
- **多數程式碼，可讀性優先。** 沒有 profile 指著的地方，寫最好讀的版本。before 在什麼情況下是對的：profile 指出解析才是熱點，而且 benchmark 量得出可觀的差距，手寫才值得；那時要把這個 benchmark 留著，說明為什麼。
- **量到的只是這個 benchmark。** 批次大小、重送比例一變，結果就變。整套服務的壓測，第 16 章〈壓測與調校〉會談；正式環境收 profile，第 9 章〈`/debug/pprof` 與診斷端點〉會談。
- **數量也要量，不靠直覺。** 第 6 章〈[什麼時候需要 worker pool，什麼時候每個任務一個 goroutine 就夠](../chapter6/06_worker_pool_or_goroutine_per_task.md)〉與〈[背壓](../chapter6/08_backpressure.md)〉留下的 worker 數量、佇列大小，都用同一套流程量。第 3 章〈[什麼時候用泛型、什麼時候用介面、什麼時候複製貼上](../chapter3/04_generics_interface_or_copy.md)〉的判斷不依據效能，原因也在這裡：效能要量，不能靠推測。

## 重點整理

- 調查效能的流程：定義指標、寫貼近實際的 benchmark、用 profile 找熱點、一次只改一個地方、再量一次比較。
- 憑直覺的最佳化常常改錯地方：手寫解析讓程式難讀，`benchstat` 卻量不出差異（`~`）。
- profile 指出熱點在哪，`go test -cpuprofile` 直接從 benchmark 收；只改熱點，再用 `benchstat` 比較，指標達標就停手。
- 確定的結果（配置次數）用 `AllocsPerRun` 斷言，時間不進測試。
- 明顯的演算法問題不用量就可以改；多數沒有 profile 指著的程式碼，可讀性優先。
