# `pprof`：先看 CPU 還是 heap

## 本集目標

看到症狀時，判斷該收哪一種 profile：CPU 用滿看 CPU profile，GC 頻繁或記憶體成長看 heap profile，並分清 `alloc_space` 與 `inuse_space`；會用 flat 與 cum 從排行榜找到該改的那一層。

## 情境

服務每個請求都會寫一行 access log，`logstats.Summarize` 把一批 log 彙總成每個路徑的請求數、錯誤數與 P95 延遲，路徑裡的數字會換成 `:id`。上線後監控看到 GC 很頻繁、記憶體曲線呈鋸齒狀。`go tool pprof` 的基本用法前作教過：〈[`pprof`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/appendix2/d_pprof.html)〉、〈[benchmark 與 `b.Loop`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/12_benchmark.html)〉、〈[GC 概念](https://holmeslin.github.io/learning-go-from-zero/zh-TW/appendix2/c_gc.html)〉。這集談的是先看哪一種。

## 常見寫法

`normalize` 為路徑預留 4096 位元組，想著「一次預留夠，就不用在寫入時擴張」：

```go,ignore
{{#include ../../../examples/chapter7/03_pprof_cpu_or_heap/before/logstats.go:summarize}}
```

測試全過，完整程式碼在 `examples/chapter7/03_pprof_cpu_or_heap/before/`。出了症狀，第一個動作是開 CPU profile，因為「慢」就是 CPU 的事。

## 問題在哪

profile 從 `BenchmarkSummarize`（`b.Loop`，2000 行輸入在迴圈外準備）收，不必改程式。一個指令同時收 CPU 與記憶體兩份，`-o` 把測試執行檔留下來，之後 `-list` 才找得到原始碼；檔案寫在目前的目錄，不要加進版本控制：

```bash
go test -run='^$' -bench=Summarize -benchmem -cpuprofile=cpu.out -memprofile=mem.out -o before.test ./examples/chapter7/03_pprof_cpu_or_heap/before
```

先照直覺讀 CPU profile。這集每一份 `go tool pprof` 的輸出都省略了 `Time`、`Duration`、`Showing`、`Dropped` 這幾行：

執行結果（某一次）：

```text
$ go tool pprof -top -nodecount=5 before.test cpu.out
File: before.test
Type: cpu
      flat  flat%   sum%        cum   cum%
     620ms 26.50% 26.50%      620ms 26.50%  runtime.pthread_cond_wait
     370ms 15.81% 42.31%      370ms 15.81%  runtime.usleep
     310ms 13.25% 55.56%      310ms 13.25%  runtime.madvise
     260ms 11.11% 66.67%      260ms 11.11%  runtime.kevent
     230ms  9.83% 76.50%      230ms  9.83%  runtime.pthread_cond_signal
```

前幾名全是 runtime 內部的函式（執行緒的休眠與喚醒、把記憶體還給作業系統的 `madvise`），沒有一個是 `logstats` 的。看得出 runtime 很忙，卻沒說誰造成的；函式名稱隨作業系統而異，結論相同。原因在於 CPU profile 回答「時間花在哪」，GC 的成本不會記在配置者名下。照這份排行榜去「最佳化」，只會改到不相干的地方。

## 較好的做法

先依症狀選 profile：CPU 用滿看 CPU profile；GC 頻繁、記憶體成長看 heap profile；延遲高但 CPU 不忙，兩者都不是答案，本章〈`runtime/trace`〉會談，block 與 mutex profile 也在本章後面的集出現。

上面的指令已經收好 `mem.out`。heap profile 帶著四種樣本，用 `-sample_index` 指定看哪一種。`inuse_space` 是取樣時（以最近一次完成的 GC 為準）還留著的記憶體，`alloc_space` 是累計配置過的記憶體，後者決定 GC 多常啟動。先看 `inuse_space`，問「有沒有東西留著不放」：

執行結果（某一次）：

```text
$ go tool pprof -sample_index=inuse_space -top -nodecount=3 before.test mem.out
File: before.test
Type: inuse_space
      flat  flat%   sum%        cum   cum%
 2051.23kB 80.01% 80.01%  2051.23kB 80.01%  runtime.mallocgc
  512.50kB 19.99%   100%   512.50kB 19.99%  hash/crc32.simpleMakeTable (inline)
         0     0%   100%   512.50kB 19.99%  hash/crc32.init
```

只有幾 MB，來源是 runtime 與標準庫自己的初始化，沒有 `logstats`：沒有洩漏。症狀是配置得太快，不是留得太久，改看 `alloc_space`，並用 `-cum` 依累計排序：

執行結果（某一次）：

```text
$ go tool pprof -sample_index=alloc_space -top -cum -nodecount=8 before.test mem.out
File: before.test
Type: alloc_space
      flat  flat%   sum%        cum   cum%
    0.04GB  0.43%  0.43%     8.61GB 99.94%  github.com/holmeslin/learning-go-to-senior/examples/chapter7/03_pprof_cpu_or_heap/before.Summarize
         0     0%  0.43%     8.61GB 99.94%  github.com/holmeslin/learning-go-to-senior/examples/chapter7/03_pprof_cpu_or_heap/before_test.BenchmarkSummarize
         0     0%  0.43%     8.61GB 99.94%  testing.(*B).run1.func1
         0     0%  0.43%     8.61GB 99.94%  testing.(*B).runN
         0     0%  0.43%     8.58GB 99.51%  github.com/holmeslin/learning-go-to-senior/examples/chapter7/03_pprof_cpu_or_heap/before.normalize
         0     0%  0.43%     8.54GB 99.06%  bytes.(*Buffer).Grow (inline)
         0     0%  0.43%     8.54GB 99.06%  bytes.(*Buffer).grow
    8.54GB 99.06% 99.49%     8.54GB 99.06%  bytes.growSlice
```

flat 是函式自己配置的，cum 連同它呼叫的函式一起算。flat 最大的是 `bytes.growSlice`，占 99%，但它是標準庫，不能改。往上看 cum：`Buffer.Grow` 也是標準庫，再往上第一個自己寫的函式是 `normalize`，flat 是 0、cum 99.51%，配置全發生在它呼叫的東西裡。`Summarize` 的 cum 雖然是 100%，但 flat 只有 0.43%，差的全在下一層的 `normalize`。該改的是 `normalize` 這一層。用 `-list` 找出是哪一行（省略了 `ROUTINE` 行的檔案路徑、`Total` 行與沒有數字的原始碼行，行號是 `before/logstats.go` 的行號）：

執行結果（某一次）：

```text
$ go tool pprof -sample_index=alloc_space -list='before.normalize' before.test mem.out
ROUTINE ======================== .../before.normalize in .../before/logstats.go
         0     8.58GB (flat, cum) 99.51% of Total
         .          .     68:func normalize(path string) string {
         .          .     69:	var b bytes.Buffer
         .     8.54GB     70:	b.Grow(maxPathLen)
         .       40MB     82:	return b.String()
```

罪魁是每一行都預留 4096 位元組、卻只用幾十位元組。修正只動這一件事：緩衝區由 `Summarize` 持有，每行重複使用；`string(key)` 只用來查 map 時，編譯器不會複製，只有第一次看到新路徑才配置：

```go,ignore
{{#include ../../../examples/chapter7/03_pprof_cpu_or_heap/after/logstats.go:summarize}}
```

after 的測試先保證行為沒變，再鎖住配置次數。`AllocsPerRun` 斷言的是確定的結果，不是時間：

```go,ignore
{{#include ../../../examples/chapter7/03_pprof_cpu_or_heap/after/logstats_test.go:same}}
{{#include ../../../examples/chapter7/03_pprof_cpu_or_heap/after/logstats_test.go:allocs}}
```

before 與 after 各收 `-count=10`，輸出存成 `before.txt`、`after.txt`，再比較。`-ignore pkg` 讓兩個套件路徑不同的結果排在同一張表；`benchstat` 怎麼讀，本章〈`b.Loop` 與 `benchstat`：怎麼比較前後結果才可信〉談過。

執行結果（某一次）：

```text
$ go run golang.org/x/perf/cmd/benchstat@v0.0.0-20261009192801-be2c69fb417e -ignore pkg before.txt after.txt
goos: darwin
goarch: arm64
cpu: Apple M4 Max
             │ before.txt  │              after.txt              │
             │   sec/op    │   sec/op     vs base                │
Summarize-14   952.5µ ± 5%   198.8µ ± 4%  -79.13% (p=0.000 n=10)

             │   before.txt   │              after.txt               │
             │      B/op      │     B/op      vs base                │
Summarize-14   8072.19Ki ± 0%   41.02Ki ± 0%  -99.49% (p=0.000 n=10)

             │  before.txt  │             after.txt              │
             │  allocs/op   │ allocs/op   vs base                │
Summarize-14   4057.00 ± 0%   65.00 ± 0%  -98.40% (p=0.000 n=10)
```

三項差異都顯著（p=0.000）：配置的位元組與次數大幅減少，這個 benchmark 的時間也跟著下降。

## 取捨

- profile 只反映 benchmark 打到的路徑與輸入。輸入和實際流量差太多，就會看錯地方；正式環境怎麼收 profile，第 9 章〈`/debug/pprof` 與診斷端點〉會談。
- heap profile 是取樣（預設平均每配置 512 KiB 記錄一次），適合找大戶，不適合算精確的次數；精確的次數用 `AllocsPerRun`。
- `alloc_space` 大不代表一定要改，只有症狀指向配置速率才值得動；GC 怎麼調，第 8 章〈什麼時候該調 `GOGC`，什麼時候別動〉會談。
- 修正把緩衝區交給 `Summarize` 持有，換來比較不直覺的 `appendNormalized(dst, ...)`。常見的配置寫法與修正，本章〈常見熱點：配置、拷貝、鎖競爭〉會談。

before 在什麼情況下是對的？`normalize` 只在啟動時呼叫幾次，或每次處理的行數很少時，多預留 4 KiB 不會有任何症狀，這時簡單的寫法就夠，不需要打開 profile。

## 重點整理

- 先看症狀再選 profile：CPU 用滿看 CPU profile，GC 頻繁或記憶體成長看 heap profile，延遲高而 CPU 不忙兩者都不是答案。
- `go test -cpuprofile`、`-memprofile` 加 `-o` 可以直接從 benchmark 收 profile，不必改程式。
- heap profile 用 `-sample_index` 指定：`alloc_space` 看配置得多快（影響 GC 頻率），`inuse_space` 看留著多少（找洩漏）。
- flat 找自己動手的那一行，cum 往上找第一個自己寫的函式；標準庫與 runtime 的 flat 再大也不是要改的地方。
