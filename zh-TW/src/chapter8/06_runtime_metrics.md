# `runtime/metrics`

## 本集目標

要把 runtime 的記憶體、GC、排程狀況回報給監控時，用 `runtime/metrics` 取代自己從 `runtime.ReadMemStats` 湊的指標：從 `metrics.All` 挑名稱、重複使用樣本切片、依 `Kind` 處理值，並知道名稱不存在時該怎麼辦。

## 情境

縮圖服務有一個 goroutine 定期讀 runtime 的記憶體狀況，送到監控系統，並設了一條告警：「heap 超過 800 MiB 就通知值班的人」。這集只談怎麼從 runtime 讀出來；指標怎麼命名、怎麼匯出，第 9 章〈指標（metrics）設計與高基數問題〉會談。GC 的基本觀念前作教過：〈[GC 概念](https://holmeslin.github.io/learning-go-from-zero/zh-TW/appendix2/c_gc.html)〉。

## 常見寫法

呼叫 `runtime.ReadMemStats`，挑幾個欄位，取一個自己看得懂的名字：

```go,ignore
{{#include ../../../examples/chapter8/06_runtime_metrics/before/snapshot.go:snapshot}}
```

程式很短，`MemStats` 的欄位名稱看起來也都跟記憶體有關。完整程式碼在 `examples/chapter8/06_runtime_metrics/before/`。

## 問題在哪

`Sys` 不是 heap。它是 runtime 向作業系統要來的全部記憶體，包含 heap、goroutine 的堆疊與 runtime 自己的資料結構，而且 heap 還給作業系統之後仍然算在裡面。名稱叫 `heap_bytes`，告警就會在負載退掉之後繼續響。

下面的程式（`readings`）持有 64 MiB 的資料再放掉，並排印出兩種讀法。`after` 那三列是 `runtime/metrics` 的指標：`/memory/classes/total:bytes` 對應 `Sys`，`/memory/classes/heap/objects:bytes` 對應 `HeapAlloc`，`/gc/heap/live:bytes` 是上一次 GC 標記到的活物件。為了讓 `live` 反映當下，程式在每次讀取前先呼叫 `runtime.GC()`，正式的服務不該這樣做。

程式讀的是 `Names` 的八個指標，下面只印其中三個加上 `before` 的兩個，單位換成 MiB；三欄依序是剛啟動（`start`）、持有 64 MiB（`hold`）、放掉並 GC 之後（`released`）。

執行結果（某一次）：平台 macOS arm64（Apple M4 Max），Go 1.27，P 的數量 14。

```text
$ go run ./examples/chapter8/06_runtime_metrics/after/readings
P=14
source  metric (MiB)                             start    hold released
before  heap_bytes                                11.9    76.5     76.5
before  heap_alloc_bytes                           0.3    64.3      0.3
after   /memory/classes/total:bytes               11.9    76.5     76.5
after   /memory/classes/heap/objects:bytes         0.3    64.3      0.3
after   /gc/heap/live:bytes                        0.3    64.3      0.3
```

`released` 那一欄的資料已經沒有任何參照，`heap_alloc_bytes` 回到起點，`heap_bytes` 卻原地不動：這些記憶體留在 runtime 手上等著重用，或已經還給作業系統、但位址空間仍然算在 `Sys` 裡。看 `heap_bytes` 的告警會以為 heap 一直沒降下來，實際上 heap 只剩起點那麼多。`heap_bytes` 與 `/memory/classes/total:bytes` 讀到同樣的大小，名稱卻讓人以為是前者的意思。

另外兩個問題在程式裡就看得到。`heap_bytes`、`gc_runs` 是自己取的名字，換了服務、換了團隊，同樣的東西就有不同的名字，儀表板與告警規則無法共用。想多回報一項（例如 GC 用掉多少 CPU），得先在 `MemStats` 找欄位、確認它的意義、改函式再發版。

## 較好的做法

`runtime/metrics` 用字串名稱指定指標，名稱的最後帶單位（`:bytes`、`:cpu-seconds`），完整清單與每個指標的說明都在 `metrics.All()`（`go doc runtime/metrics` 也列了同一份）。先從說明挑出要的指標，把名稱放進一個清單：

```go,ignore
{{#include ../../../examples/chapter8/06_runtime_metrics/after/sampler.go:names}}
```

讀取的 `Sampler` 做三件事。建立時用 `metrics.All()` 查每個名稱的 `Kind`：這個 Go 版本沒有的名稱、值是直方圖的指標記進 `Unsupported`，不讓整個服務因為一個名稱失敗。`Read` 重複使用同一個 `[]metrics.Sample`（`metrics.Read` 的文件鼓勵這樣做），再依 `Value.Kind()` 取 `Uint64` 或 `Float64`；用錯的取法會 panic，所以要先看 `Kind`。呼叫端重複使用同一個 `dst`，讀取就不配置：

```go,ignore
{{#include ../../../examples/chapter8/06_runtime_metrics/after/sampler.go:sampler}}
```

測試不斷言任何數值，只斷言確定的事。清單裡的名稱都存在，型別符合預期（`Kind` 在文件裡保證不會改變）：

```go,ignore
{{#include ../../../examples/chapter8/06_runtime_metrics/after/sampler_test.go:kinds}}
```

不支援的名稱被略過，其餘照常讀：

```go,ignore
{{#include ../../../examples/chapter8/06_runtime_metrics/after/sampler_test.go:unsupported}}
```

「重複使用樣本切片」也由測試鎖住：第一次讀之後，每次 `Read` 的配置次數是 0（`AllocsPerRun` 的結果是確定的，`-race` 下也成立）：

```go,ignore
{{#include ../../../examples/chapter8/06_runtime_metrics/after/sampler_test.go:allocs}}
```

用法與輸出由 `go test` 驗證；數值每次不同，所以範例只印名稱：

```go,ignore
{{#include ../../../examples/chapter8/06_runtime_metrics/after/example_test.go:example}}
```

告警改看 `/gc/heap/live:bytes`，或 `/memory/classes/heap/objects:bytes` 的趨勢，意思與名稱一致。

## 取捨

- **`ReadMemStats` 沒有被淘汰。** 一次性的除錯（臨時印一行、在測試裡看某段程式配置前後的 `HeapAlloc`）用它最快，不必先查名稱。問題出在把它當成長期指標，又沒讀清楚欄位的意思。只想看 heap，也可以用 heap profile：第 7 章〈[`pprof`：先看 CPU 還是 heap](../chapter7/03_pprof_cpu_or_heap.md)〉談過。
- **名稱會隨版本增加，也會被標成棄用。** 例如 `/gc/pauses:seconds` 在說明裡寫著請改用 `/sched/pauses/total/gc:seconds`。清單是自己挑的，新版本的新指標不會自動出現；`Unsupported` 不為空時，啟動時記一行日誌，別默默少一條線。
- **這個 `Sampler` 只處理單一數值。** 直方圖（例如 `/sched/latencies:seconds`）要自己決定怎麼彙整 bucket，留給第 9 章。它也不能被多個 goroutine 同時呼叫，要並行就各建一個。
- **`/memory/classes/total:bytes` 不含 cgo 或 `syscall` 映射的記憶體**，所以和作業系統看到的行程記憶體還是會有差距。heap 與行程記憶體的差別，本章〈記憶體配置與 heap 成長〉談過。
- **`gctrace` 適合一次性的除錯，長期觀察用這集的指標。** 怎麼逐欄讀 `gctrace`，本章〈讀 GC trace〉會談；什麼時候該動 `GOGC`，本章〈什麼時候該調 `GOGC`，什麼時候別動〉談過。

## 重點整理

- `MemStats.Sys` 是 runtime 向作業系統要的全部記憶體，不是 heap；heap 還給作業系統之後它也不會降。
- `runtime/metrics` 以帶單位的名稱選指標，說明與 `Kind` 都在 `metrics.All()`；`Kind` 保證不變。
- 建立時查 `Kind`、略過不支援的名稱；`Read` 重複使用樣本切片，依 `Kind` 取值。
- 測試只斷言名稱、型別、略過行為與零配置，不斷言數值。
- 指標怎麼命名與匯出，第 9 章〈指標（metrics）設計與高基數問題〉會談。
