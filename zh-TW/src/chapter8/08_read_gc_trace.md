# 讀 GC trace

## 本集目標

看到一行 `GODEBUG=gctrace=1` 的輸出，說得出每一欄的意思，並判斷它在說「GC 太頻繁」「heap 目標被 `GOMEMLIMIT` 壓住」還是「GC 吃掉太多 CPU」，再決定該回頭讀本章哪一集。

## 情境

服務的延遲變差，懷疑是 GC。CPU profile 看得到 GC 的函式在忙，卻看不到 GC 多久跑一次、heap 目標多大。`gctrace` 每輪 GC 印一行，這些都在裡面。GC 的概念（標記清除、`GOGC`、`GOMEMLIMIT`）前作〈[GC 概念](https://holmeslin.github.io/learning-go-from-zero/zh-TW/appendix2/c_gc.html)〉教過，`pprof` 第 7 章〈[`pprof`：先看 CPU 還是 heap](../chapter7/03_pprof_cpu_or_heap.md)〉談過。這集只談怎麼讀那一行。

## 產生 trace 的程式

下面這支 `main` 模擬有常駐快取的服務：`-live` 指定常駐資料的大小（含指標的小物件，GC 每輪都要掃描），再由每個 P 一個 worker 不斷配置用完就丟的緩衝區，直到累計配置 `-total` MiB。它會吃上百 MiB 的記憶體，所以只放在 `main` 套件，CI 只編譯不執行：

```go,ignore
{{#include ../../../examples/chapter8/08_read_gc_trace/demo/main.go:demo}}
```

以下的 trace 都在 macOS（arm64，Apple M4 Max，Go 1.27.0）上取得，除非另外寫明，P 的數量是 14。每個情況只挑出指定行號的幾行貼上，其餘省略。

## 一行 trace，逐欄讀

`runtime` 套件文件逐欄說明了格式，並註明格式可能改變。下面的 `Cycle` 把每一欄對應到欄位，`Parse` 拆一行 trace，`Example` 用一行取自上面那支程式的 trace 示範：

```go,ignore
{{#include ../../../examples/chapter8/08_read_gc_trace/gctrace/gctrace.go:cycle}}
{{#include ../../../examples/chapter8/08_read_gc_trace/gctrace/gctrace_test.go:example}}
```

幾個判斷時最常用的欄位：

- **`GC 占 9%`**：程式啟動以來累計的比例，不是這一輪的。短暫的尖峰會被平均掉。
- **牆上時間三段**：第一與第三段是 stop-the-world，程式真的被暫停；中間的 mark and scan 與程式並行。
- **CPU 時間五段**：第一與最後一段是暫停階段所有 P 的 CPU 加總。中間三個依序是 assist、background、idle：assist 是 GC 的工作插在配置裡，由正在配置的 goroutine 自己負擔，這一欄大代表程式被拉去幫 GC。
- **heap 開始、結束、存活**：開始到結束通常會變大，因為標記期間程式仍在配置。目標與存活量的差，大致是下一輪 GC 開始前程式可以配置的空間。
- **P 的數量**：`runtime` 文件對應 `/sched/gomaxprocs:threads`。文件只寫 `GC 占` 是 "percentage of time spent in GC since program start"，沒有交代怎麼算，所以兩份 trace 的 P 數不同時，不要直接比這個比例；比較前先確認這一欄相同。

## 三種常見的情況

### GC 太頻繁

常駐資料只有 1 MiB，其餘全是用完就丟的緩衝區。`GOGC` 算出的目標比 4 MB 小時，runtime 仍以 4 MB 為最小的 heap 目標（見〈[GC 指南](https://go.dev/doc/gc-guide)〉）：

執行結果（某一次）：

```text
$ go build -o demo ./examples/chapter8/08_read_gc_trace/demo
$ GODEBUG=gctrace=1 ./demo -live=1 2> frequent.txt
$ sed -n '200,201p' frequent.txt
gc 200 @0.080s 13%: 0.082+0.14+0.004 ms clock, 1.1+0.070/0.23/0.029+0.068 ms cpu, 3->4->1 MB, 4 MB goal, 0 MB stacks, 0 MB globals, 14 P
gc 201 @0.081s 13%: 0.033+0.14+0.002 ms clock, 0.46+0.026/0.21/0+0.040 ms cpu, 4->4->1 MB, 4 MB goal, 0 MB stacks, 0 MB globals, 14 P
```

第 200、201 次的時間點只差一毫秒左右，goal 停在 4 MB，存活量只有 1 MB。每一輪都很便宜（暫停階段合計不到 0.3 ms），單看一輪沒有問題，問題在頻率。先確認它真的影響延遲或 `GC 占` 那一欄；沒有影響就不必動。有影響的話，先減少配置（第 7 章〈[常見熱點：配置、拷貝、鎖競爭](../chapter7/05_common_hotspots.md)〉），再考慮調大 `GOGC`，本章〈[什麼時候該調 `GOGC`，什麼時候別動](02_when_to_tune_gogc.md)〉談過。

### heap 目標被 `GOMEMLIMIT` 壓住

同一個負載（常駐 128 MiB），一次不設限制，一次設 `GOMEMLIMIT=200MiB`：

執行結果（某一次）：

```text
$ GODEBUG=gctrace=1 ./demo -live=128 2> base.txt
$ GOMEMLIMIT=200MiB GODEBUG=gctrace=1 ./demo -live=128 2> limit.txt
$ sed -n '10p' base.txt
gc 10 @0.122s 8%: 0.12+2.7+0.045 ms clock, 1.7+7.2/9.5/2.3+0.63 ms cpu, 343->406->198 MB, 403 MB goal, 0 MB stacks, 0 MB globals, 14 P
$ sed -n '40p' limit.txt
gc 40 @0.495s 9%: 0.11+1.9+0.097 ms clock, 1.6+7.4/6.3/5.5+1.3 ms cpu, 171->182->147 MB, 178 MB goal, 0 MB stacks, 0 MB globals, 14 P
$ wc -l base.txt limit.txt
      14 base.txt
      64 limit.txt
      78 total
```

沒有限制時，goal 是存活量的兩倍上下（`GOGC=100` 的預設）；設了限制，goal 被壓到接近限制，遠低於前者，同一個負載 GC 的次數（`wc -l` 數的行數，每輪一行；`total` 是 `wc` 自己加的總計）也多了好幾倍，這次是 14 行對 64 行。這是 `GOMEMLIMIT` 在運作，不一定是問題：限制本來就是為了換記憶體的上限。要看的是限制離存活量有多遠，怎麼設，本章〈[`GOMEMLIMIT` 與容器記憶體上限](03_gomemlimit_container.md)〉談過。

### GC 占用的 CPU 比例過高

限制縮到 2 個 P（`GOMAXPROCS=2`），`GOMEMLIMIT=150MiB` 幾乎等於存活的資料：

執行結果（某一次）：

```text
$ GOMAXPROCS=2 GOMEMLIMIT=150MiB GODEBUG=gctrace=1 ./demo -live=128 -total=1024 2> busy.txt
$ sed -n '40p' busy.txt
gc 40 @0.398s 62%: 0.017+7.4+0.001 ms clock, 0.035+13/0.12/0+0.003 ms cpu, 151->166->151 MB, 151 MB goal, 0 MB stacks, 0 MB globals, 2 P
```

三處同時指向它：`GC 占` 過半（這次是 62%；上面兩個情況都在二成以下，但 P 的數量不同，只看量級）；開始時的 heap、存活量與 goal 幾乎相等，程式沒有空間可以配置，GC 一結束又得開始；CPU 時間裡 assist 遠大於 background，配置的 goroutine 被拉去做 GC。差別在原因：存活的資料本身接近上限，或是 `GOGC` 調得太低，兩者在 trace 上都會看到 goal 貼著存活量。前者的處理（調高限制、減少存活的資料）在本章〈[`GOMEMLIMIT` 與容器記憶體上限](03_gomemlimit_container.md)〉，那一集也談過 runtime 替 GC 能用的 CPU 設的上限；`GC 占` 是啟動以來累計的比例，和那個上限量的不是同一件事，兩個數字不能直接比。後者在本章〈[什麼時候該調 `GOGC`，什麼時候別動](02_when_to_tune_gogc.md)〉談過。

## 取捨

- **`gctrace` 適合除錯，不適合長期觀察。** 它印到標準錯誤、一輪一行，要用環境變數重啟程式才打得開，也沒有彙總。長期觀察 GC 用本章〈[`runtime/metrics`](06_runtime_metrics.md)〉談過的指標；`runtime` 文件在每一欄旁邊註明了對應的指標（例如 goal 是 `/gc/heap/goal:bytes`）。指標怎麼設計與匯出，第 9 章〈[指標（metrics）設計與高基數問題](../chapter9/03_metrics_cardinality.md)〉談過。
- **格式可能隨版本改變。** 文件明寫格式可能改變，上面的 `gctrace` 套件只對應 Go 1.27；欄位的定義以 [`runtime` 套件文件](https://pkg.go.dev/runtime)為準，不要把解析 trace 寫進正式環境的流程。
- **一行不是趨勢。** 大小取整數的 MB，小的 heap 常常是 0；數字也每次不同。要看的是一段 trace 裡 goal、存活量與頻率的相對變化，不是某個數字本身。
- **大多數服務不需要調。** 預設值夠用，trace 看不出頻繁、被壓住或比例過高，就沒有東西要改。
- **trace 本身有輸出量。** GC 很頻繁時（像第一個情況）一秒就是幾百行，除錯完要關掉。

## 重點整理

- 一行 `gctrace` 依序是：第幾次、啟動後的秒數、累計的 GC 比例、三段牆上時間、五段 CPU 時間、heap 開始／結束／存活、goal、stacks、globals、P 的數量。
- 讀 trace 看相對關係：goal 與存活量的差、相鄰兩輪的間隔、assist 在 CPU 時間裡的比例。
- 頻繁、被 `GOMEMLIMIT` 壓住、比例過高是三種不同的症狀，各自回頭讀 `GOGC`、`GOMEMLIMIT` 那兩集。
- `GC 占` 是累計值，P 的數量不同不能直接比。
- `gctrace` 只用來除錯；長期觀察用 `runtime/metrics`，格式以 `runtime` 文件為準。
