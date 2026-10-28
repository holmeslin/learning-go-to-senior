# 記憶體配置與 heap 成長

## 本集目標

看到記憶體曲線只升不降時，認出三種常見的程式碼原因：沒有上限的快取、子切片留住整塊大陣列、刪除後不會縮小的 map；知道各自怎麼改、怎麼用測試鎖住。也分清 heap 在長與行程的 RSS 在長，以及 runtime 什麼時候把記憶體還給作業系統。

## 情境

webhook 接收服務每收到一筆通知（最多 64 KiB）就讀進一塊緩衝區處理，同時維護三個長壽命的元件：`Cache` 記住客戶最近一次的摘要，`Seen` 記住收過的請求代號，`Pending` 記住還沒送達的通知。服務跑了幾天，監控上的記憶體只升不降。GC 的基本概念前作教過：〈[GC 概念](https://holmeslin.github.io/learning-go-from-zero/zh-TW/appendix2/c_gc.html)〉。這集談的是 GC 幫不上忙的情況：只要還有指標指著，物件就是活的，程式還用不用都一樣。

## 常見寫法

`Cache` 是一張只進不出的 map：

```go,ignore
{{#include ../../../examples/chapter8/04_heap_growth/before/cache.go:cache}}
```

`Seen` 只需要通知開頭 16 位元組的請求代號，順手切一段存起來（子切片共用底層陣列，前作教過：〈[切片共用底層陣列](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter2/18_shared_backing_array.html)〉）：

```go,ignore
{{#include ../../../examples/chapter8/04_heap_growth/before/seen.go:seen}}
```

`Pending` 在通知送達時 `delete`（〈[`delete` 與走訪 map](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter2/28_map_delete_range.html)〉）；尖峰過後只剩少數幾筆：

```go,ignore
{{#include ../../../examples/chapter8/04_heap_growth/before/pending.go:pending}}
```

每個元件的邏輯都對，完整程式碼在 `examples/chapter8/04_heap_growth/before/`。

## 問題在哪

先看 `Seen`。這個 `Example` 把切片的容量印出來：

```go,ignore
{{#include ../../../examples/chapter8/04_heap_growth/before/example_test.go:example}}
```

長度是 16，容量卻是整塊緩衝區剩下的長度。只要這個切片還被存著，整個 64 KiB 的陣列都不能回收。

三個元件各用 `demo` 量一次。`demo` 在讀數前先呼叫 `runtime.GC()`，再讀 `/gc/heap/live:bytes`（上一次 GC 標記為還活著的 heap；怎麼讀指標，本章〈`runtime/metrics`〉會談）；強制 GC 只是為了量測，不是調校手法。它會刻意配置數百 MiB，所以放在 `main` 套件，`go test` 不會跑。平台是 macOS（arm64）、14 個 P，每次輸出的第一行 `GOMAXPROCS 14` 省略了：

執行結果（某一次）：

```text
$ go run ./examples/chapter8/04_heap_growth/after/demo -case=cache -mode=before
before cache    live heap    81.9 MiB
$ go run ./examples/chapter8/04_heap_growth/after/demo -case=seen -mode=before
before seen     live heap   125.3 MiB
$ go run ./examples/chapter8/04_heap_growth/after/demo -case=pending -mode=before
before pending  live heap    80.3 MiB
```

cache 放了 50 萬個客戶，一筆都沒被丟掉。seen 收了 2000 筆 64 KiB 的通知，2000 塊緩衝區全部還活著，只為了留 2000 個 16 位元組的代號。pending 尖峰時有 100 萬筆，之後只剩 1000 筆，空間卻還在。這三種都不是 GC 的問題：記憶體一直有人指著，GC 再跑也回收不了。

用 heap profile 找時，看 `inuse_space`（第 7 章〈[`pprof`：先看 CPU 還是 heap](../chapter7/03_pprof_cpu_or_heap.md)〉談過）。但 profile 記的是「誰配置的」，不是「誰留著」：seen 會指到配置緩衝區的那一行，還得沿著引用往回找是誰在持有。

## 較好的做法

三個元件各改一處。`Cache` 加上筆數上限，用 `container/list` 排出最近使用的順序，滿了就丟最久沒用到的：

```go,ignore
{{#include ../../../examples/chapter8/04_heap_growth/after/cache.go:cache}}
```

`Seen` 只複製需要的 16 位元組。`make` 加 `copy` 的容量一定等於長度；`bytes.Clone` 的結果可能多留一點空間：

```go,ignore
{{#include ../../../examples/chapter8/04_heap_growth/after/seen.go:seen}}
```

`Pending` 的 map 不會自己縮小，尖峰過後由呼叫端呼叫 `Compact`，把剩下的搬進一張剛好夠大的新 map，舊的交給 GC：

```go,ignore
{{#include ../../../examples/chapter8/04_heap_growth/after/pending.go:pending}}
```

測試只斷言確定的值：筆數上限、複製後的容量。heap 大小隨機器與版本而異，不斷言：

```go,ignore
{{#include ../../../examples/chapter8/04_heap_growth/after/ingest_test.go:cache}}
{{#include ../../../examples/chapter8/04_heap_growth/after/ingest_test.go:seen}}
```

`Compact` 的測試只能確認內容沒變；map 內部多大看不到，那一項靠量。同一台機器、同樣的指令換成 `-mode=after`（同樣省略第一行 `GOMAXPROCS 14`）：

執行結果（某一次）：

```text
$ go run ./examples/chapter8/04_heap_growth/after/demo -case=cache -mode=after
after  cache    live heap     2.3 MiB
$ go run ./examples/chapter8/04_heap_growth/after/demo -case=seen -mode=after
after  seen     live heap     0.4 MiB
$ go run ./examples/chapter8/04_heap_growth/after/demo -case=pending -mode=after
after  pending  live heap     0.4 MiB
```

三項都只剩 before 的一小部分。cache 的大小由上限（1 萬筆）決定，不再隨客戶數成長。

**heap 與 RSS。** 修好之後，監控上的 RSS（作業系統認為行程占用的實體記憶體）不一定馬上跟著降。GC 回收的是 heap 裡的物件，空出來的頁面先留在 runtime 手上，之後才還給作業系統：`/memory/classes/heap/free:bytes` 是閒置但還沒還的，`released` 是已經還的。`runtime/debug` 的文件說，不呼叫任何東西時 runtime 也會在背景逐步歸還；`debug.FreeOSMemory` 則是強制 GC 再盡量歸還。用 `demo -case=rss` 看這三個欄位和 `VmRSS`（讀 `/proc/self/status`，所以要在 Linux 跑）。環境是 `golang:1.27-alpine` image、`--cpus=4 --memory=1g`、cgroup v2、4 個 P；輸出省略了第一行 `GOMAXPROCS 4`：

```bash
CGO_ENABLED=0 GOOS=linux go build -o demo-linux ./examples/chapter8/04_heap_growth/after/demo
docker run --rm --cpus=4 --memory=1g -v "$PWD:/w:ro" golang:1.27-alpine /w/demo-linux -case=rss
```

執行結果（某一次）：

```text
holding 256 MiB             live  256.1  free    0.1  released    7.2  VmRSS 259.6 MiB
after runtime.GC()          live    0.1  free  256.1  released    7.1  VmRSS 259.6 MiB
after debug.FreeOSMemory()  live    0.1  free    1.5  released  261.7  VmRSS 5.1 MiB
```

GC 之後 `live` 幾乎歸零，原本占用的量全變成 `free`，VmRSS 幾乎沒變；`FreeOSMemory` 之後 `released` 變大、VmRSS 才降下來。所以看監控要分開看：`live` 持續上升才是程式碼的問題；RSS 高而 `live` 低，通常是 `free` 還沒還。runtime 在設了記憶體上限時會更積極歸還，本章〈`GOMEMLIMIT` 與容器記憶體上限〉談過。

## 取捨

- **複製有成本。** `Seen.Add` 每次多配置一次、多拷貝 16 位元組。通知本來就小，或處理完請求就放掉時，子切片沒有問題；只有「來源很大、又長期保存」才值得複製。
- **上限怎麼定沒有公式。** 看每筆的大小、記憶體預算與命中率；太小，命中率下降、重算變多。LRU 每筆還多一個 `list` 節點，而且這個 `Cache` 不是並行安全的，多個 goroutine 要加鎖（第 6 章〈[channel 與 mutex 怎麼選](../chapter6/07_channel_or_mutex.md)〉）。
- **map 不縮小通常不是問題。** 空出來的位置之後會被新的 key 重用；尖峰會反覆回來的服務，重建只是讓下一次尖峰重新擴張。只有曾經很大、之後長期很小才值得 `Compact`，重建的當下新舊兩張 map 也會同時存在。
- **別為了讓 RSS 好看定期呼叫 `debug.FreeOSMemory`。** 它每次都強制一次 GC，下一輪配置又得向作業系統要頁面。要不要調 GC，本章〈什麼時候該調 `GOGC`，什麼時候別動〉談過。
- **before 在什麼情況下是對的？** 快取的 key 本來就有界（例如客戶只有幾百個）、程式很短命（CLI、批次工作，結束時記憶體全還），都不必加上限或重建。

## 重點整理

- GC 只回收沒人指著的物件：沒有上限的快取、留住大陣列的子切片、刪除後不縮小的 map，都是「還有人指著」，GC 幫不上忙。
- 快取加筆數上限；長期保存大來源的一小段時複製出來（容量等於長度）；曾經很大、之後長期很小的 map 才重建。
- 測試斷言確定的值（筆數上限、複製後的容量）；heap 的變化用手動貼上的讀數佐證，不放進測試。
- heap 在長看 `live`，RSS 還包含 runtime 手上還沒還給作業系統的閒置頁面，兩者要分開看。
