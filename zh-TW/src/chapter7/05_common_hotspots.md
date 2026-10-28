# 常見熱點：配置、拷貝、鎖競爭

## 本集目標

認得 profile 最常指出的三種熱點：迴圈裡的配置與拷貝、一把全域鎖；知道各自怎麼改、怎麼證明有改善，並能用 `testing.AllocsPerRun` 把配置次數鎖住，不讓它被改回去。

## 情境

日誌服務有兩條熱路徑。`Report` 把一批請求紀錄裡的失敗請求（狀態碼 400 以上）整理成文字報告；`Metrics.Observe` 在每個請求結束時累計各路由的請求數，幾百個 goroutine 同時呼叫。假設 profile 已經指到這兩處（怎麼找到熱點，本章〈`pprof`：先看 CPU 還是 heap〉會談）。benchmark 與工具的用法前作教過：〈[benchmark 與 `b.Loop`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/12_benchmark.html)〉、〈[`strings.Builder` 與 `bytes`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter12/06_builder_bytes.html)〉、〈[`sync.Mutex`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter9/04_mutex.html)〉。

## 常見寫法

`Report` 先用 `append` 挑出失敗請求，再用 `+=` 一行行接成字串；路徑要小寫、去掉查詢字串，順手在 `[]byte` 與 `string` 之間轉來轉去：

```go,ignore
{{#include ../../../examples/chapter7/05_common_hotspots/before/report.go:report}}
```

`Metrics` 用一把鎖保護一張 map，在鎖裡面才組出 key：

```go,ignore
{{#include ../../../examples/chapter7/05_common_hotspots/before/metrics.go:metrics}}
```

每一行都看得懂，測試也全過。完整程式碼在 `examples/chapter7/05_common_hotspots/before/`。

## 問題在哪

**配置與拷貝。** 看 before 的配置次數，指令加了 `-benchmem`；輸出省略了 `pkg:` 行與最後的 `PASS`、`ok` 行。

執行結果（某一次）：

```text
$ go test -run='^$' -bench=Report -benchmem -benchtime=1000x ./examples/chapter7/05_common_hotspots/before
goos: darwin
goarch: arm64
cpu: Apple M4 Max
BenchmarkReport/entries=100-14         	    1000	      3249 ns/op	   10192 B/op	     106 allocs/op
BenchmarkReport/entries=1000-14        	    1000	     74590 ns/op	  675079 B/op	    1009 allocs/op
```

筆數是 10 倍，配置次數也大約是 10 倍，每筆失敗請求約 4 次。來源有三處：`append` 沒有預留容量，每次空間不夠就配置一塊更大的、把舊的拷過去；`out +=` 每一圈都配置一個新字串，再把前面全部的內容拷一遍，總拷貝量隨筆數的平方成長；`route` 先轉成 `[]byte`、`bytes.ToLower` 再拷一份、`string(b)` 又拷一份，每一次拷貝都是一份新的路徑；`strconv.Itoa` 也配置一次。

**鎖競爭。** 沒有競爭時，鎖幾乎沒有成本；問題要在多個 goroutine 同時搶時才出現，所以 benchmark 用 `RunParallel`，讓多個 goroutine 同時呼叫（`b.Loop` 是單一 goroutine 的寫法，`RunParallel` 改用 `pb.Next()` 控制迴圈）：

```go,ignore
{{#include ../../../examples/chapter7/05_common_hotspots/after/bench_test.go:observe}}
```

`go test -mutexprofile` 會記下 goroutine 為了搶鎖等了多久。迴圈次數固定，才能和 after 比較；`-top -cum` 按累計時間排序、只列前 5 個節點，輸出省略了 `File:`、`Time:` 與 `Dropped` 三行，過長的套件路徑也維持原樣。

執行結果（某一次）：

```text
$ go test -run='^$' -bench=Observe -benchtime=2000000x -mutexprofile=mutex_before.out ./examples/chapter7/05_common_hotspots/before
$ go tool pprof -top -cum -nodecount=5 mutex_before.out
Type: delay
Showing nodes accounting for 4050.72ms, 100% of 4052.10ms total
Showing top 5 nodes out of 17
      flat  flat%   sum%        cum   cum%
         0     0%     0%  3496.17ms 86.28%  github.com/holmeslin/learning-go-to-senior/examples/chapter7/05_common_hotspots/before.(*Metrics).Observe
         0     0%     0%  3496.17ms 86.28%  github.com/holmeslin/learning-go-to-senior/examples/chapter7/05_common_hotspots/before_test.BenchmarkObserve.func1
         0     0%     0%  3496.17ms 86.28%  testing.(*B).RunParallel.func1
 2626.76ms 64.82% 64.82%  3081.88ms 76.06%  sync.(*Mutex).Unlock
 1423.97ms 35.14%   100%  1423.97ms 35.14%  runtime.unlock (inline)
```

type 是 `delay`，意思是等待時間的總和（可以超過實際經過的時間，因為每個 goroutine 各算各的）。等待時間記在放鎖的那一行，所以 `sync.(*Mutex).Unlock` 的 flat 最大；往上看 cum，源頭是 `Observe`，絕大部分的等待來自這一把鎖。讀法和 CPU profile 一樣：flat 指出誰在花時間，cum 指出該改哪一層。第 6 章〈[channel 與 mutex 怎麼選](../chapter6/07_channel_or_mutex.md)〉留下的問題「鎖競爭是否成為瓶頸」，量法就是這一段。

## 較好的做法

**配置與拷貝。** 事先知道上限就一次配好：失敗清單預留 `len(entries)`，`strings.Builder` 用 `Grow` 預估輸出大小；`route` 改用 `strings`，不經過 `[]byte`（路徑已經是小寫時，`strings.ToLower` 原樣回傳，不配置）；狀態碼用 `strconv.AppendInt` 寫進堆疊上的小陣列，不再 `Itoa`：

```go,ignore
{{#include ../../../examples/chapter7/05_common_hotspots/after/report.go:report}}
```

配置次數用測試鎖住。`AllocsPerRun` 的結果是確定的，`-race` 下也一樣，所以可以斷言；時間不行，不能放進測試。斷言是上限而不是「剛好」，編譯器之後變聰明、配置更少時不會讓測試失敗：

```go,ignore
{{#include ../../../examples/chapter7/05_common_hotspots/after/report_test.go:allocs}}
```

改寫不能改變結果。`report_test.go` 另有一個測試，拿 after 與 before 的 `Report` 對同一批輸入（含大小寫、查詢字串、空路徑與 399 這種邊界）比較，輸出必須一致；`Metrics` 也有同樣的測試。常見的輸出如下，由 `go test` 驗證：

```go,ignore
{{#include ../../../examples/chapter7/05_common_hotspots/after/example_test.go:example}}
```

**鎖競爭。** 按 key 的雜湊值把 map 分成 16 個分片，每個分片一把鎖。要先算出 key 才知道落在哪個分片，所以字串串接自然移出鎖外，臨界區只剩 map 的更新：

```go,ignore
{{#include ../../../examples/chapter7/05_common_hotspots/after/metrics.go:metrics}}
```

同樣的次數重收 after 的 mutex profile。輸出只留 `Showing nodes accounting for` 那一行，可以和 before 的同一行直接比。

執行結果（某一次）：

```text
$ go test -run='^$' -bench=Observe -benchtime=2000000x -mutexprofile=mutex_after.out ./examples/chapter7/05_common_hotspots/after
$ go tool pprof -top -cum -nodecount=5 mutex_after.out
Showing nodes accounting for 669.84ms, 100% of 669.89ms total
```

同樣 200 萬次 `Observe`，等待時間的總和少了很多。最後用 `benchstat` 比較兩邊各 10 次的結果。輸出省略了 `B/op` 那張表；表頭的 `-14` 是這台機器的 `GOMAXPROCS`；`Observe` 的 allocs/op 沒有變（`~`），可見這裡改的只有鎖：

執行結果（某一次）：

```text
$ go test -run='^$' -bench=. -benchmem -count=10 ./examples/chapter7/05_common_hotspots/before > before.txt
$ go test -run='^$' -bench=. -benchmem -count=10 ./examples/chapter7/05_common_hotspots/after > after.txt
$ go run golang.org/x/perf/cmd/benchstat@v0.0.0-20261009192801-be2c69fb417e -ignore=pkg before.txt after.txt
goos: darwin
goarch: arm64
cpu: Apple M4 Max
                       │  before.txt   │              after.txt              │
                       │    sec/op     │   sec/op     vs base                │
Report/entries=100-14     2.849µ ± 10%   1.022µ ± 9%  -64.13% (p=0.000 n=10)
Report/entries=1000-14   72.928µ ±  1%   9.142µ ± 2%  -87.47% (p=0.000 n=10)
Observe-14                179.6n ±  6%   115.4n ± 2%  -35.76% (p=0.000 n=10)
geomean                   3.342µ         1.025µ       -69.32%

                       │  before.txt   │              after.txt               │
                       │   allocs/op   │ allocs/op   vs base                  │
Report/entries=100-14     106.000 ± 0%   2.000 ± 0%  -98.11% (p=0.000 n=10)
Report/entries=1000-14   1009.000 ± 0%   2.000 ± 0%  -99.80% (p=0.000 n=10)
Observe-14                  1.000 ± 0%   1.000 ± 0%        ~ (p=1.000 n=10) ¹
geomean                     47.47        1.587       -96.66%
¹ all samples are equal
```

三列的 sec/op 都顯著下降（p 值都遠小於 0.05）。`Report` 的差距隨筆數放大：before 的耗時成長得比筆數快，after 大致跟著筆數走，與平方成長的推論吻合。`benchstat` 怎麼讀，本章〈`b.Loop` 與 `benchstat`：怎麼比較前後結果才可信〉談過。

## 取捨

- **before 在多數程式裡是對的。** 不在熱路徑、筆數只有幾十筆時，`+=` 與沒預留容量的 `append` 最好讀，改了也量不出差異。沒有 profile 指著，就不要動。
- **預留容量要估，估錯有代價。** `Grow` 估少了結果仍然正確，只是多配置一兩次；估多了浪費記憶體。after 的失敗清單預留 `len(entries)`，失敗請求很少時就是浪費。
- **`sync.Pool` 只在 profile 指出大量短命物件時才值得。** 例如每個請求都配置一塊大緩衝區。它要自己 `Get`／`Put` 配對、取出後重設內容，而且 `-race` 下 `Pool` 會隨機丟掉物件，所以經過它的路徑不能斷言精確的配置次數。這一集的熱點用預留容量就解決了，不需要它。
- **分片只在 profile 證明有鎖競爭時才做。** 它讓程式變長：`Count` 要先找分片，要列出全部或加總，就得把每個分片鎖一遍。分片數是猜的；熱門的 key 只有幾個時，它們可能落在同一個分片，鎖還是同一把。單一數值用 `atomic` 就不需要鎖。
- **配置為什麼發生，這一集沒有回答。** 結果被回傳、存進介面或被閉包捕捉時，值會從堆疊移到 heap，本章〈什麼時候值得看逃逸分析〉會談。
- **斷言只鎖得住這一個輸入。** `AllocsPerRun` 的測試用路徑都是小寫的樣本；路徑含大寫時，`ToLower` 會為每筆配置一次。要鎖哪種輸入，由熱路徑上實際的資料決定。把它放進每次 CI、時間交給固定機器，本章〈把效能回歸擋在 CI 之外〉會談。

## 重點整理

- 迴圈裡的 `+=`、沒預留容量的 `append`、`[]byte` 與 `string` 來回轉換，是 profile 最常指出的配置與拷貝；已知大小就預留，轉換用 `strings` 一路做完。
- 配置次數用 `AllocsPerRun` 在 after 的測試斷言上限；時間不斷言，交給 `benchstat` 比較。
- 鎖競爭用 `RunParallel` 加 `-mutexprofile` 量：等待時間記在 `Unlock`，看 cum 找到是哪一層的鎖；縮小臨界區、分片是兩種修法。
- 改寫要有測試保證結果與改寫前一致。
- `sync.Pool` 與分片都有複雜度的代價，沒有 profile 的證據不要用。
