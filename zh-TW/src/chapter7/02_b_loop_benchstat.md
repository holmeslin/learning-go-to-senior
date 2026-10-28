# `b.Loop` 與 `benchstat`：怎麼比較前後結果才可信

## 本集目標

拿到一份 benchmark 結果時，判斷它能不能信：計時範圍對不對、結果有沒有被編譯器丟掉、差異有沒有超出雜訊；並學會用 `-count=10` 加 `benchstat` 比較前後，讀懂 `±`、p 值與 `~`。

## 情境

閘道每收到一批請求，要把來源位址在封鎖名單裡的濾掉。`Filter(ips, blocklist)` 對每個位址掃過整份名單；有人想改成先把名單放進集合，再每個位址查一次，打算用 benchmark 證明值得改。`b.Loop`、`-bench`、`-benchmem` 前作教過：〈[benchmark 與 `b.Loop`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/12_benchmark.html)〉。這集談的是怎麼讓量出來的數字可信；找出時間花在哪是 profile 的事，本章〈`pprof`：先看 CPU 還是 heap〉會談。

## 常見寫法

要改的 `Filter`，和另一個順手一起量的小函式 `IsInternal`：

```go,ignore
{{#include ../../../examples/chapter7/02_b_loop_benchstat/before/filter.go:filter}}
{{#include ../../../examples/chapter7/02_b_loop_benchstat/before/filter.go:internal}}
```

用法與輸出：

```go,ignore
{{#include ../../../examples/chapter7/02_b_loop_benchstat/before/example_test.go:example}}
```

benchmark 照舊寫法：`b.N` 迴圈，每圈準備輸入，跑一次看數字：

```go,ignore
{{#include ../../../examples/chapter7/02_b_loop_benchstat/before/filter_bench_test.go:bench}}
```

完整程式碼在 `examples/chapter7/02_b_loop_benchstat/before/`。

## 問題在哪

先跑，指令在 repo 根目錄。輸出省略了 `pkg:` 行與最後的 `PASS`、`ok` 行。

執行結果（某一次）：

```text
$ go test -run='^$' -bench=. -benchmem -count=3 ./examples/chapter7/02_b_loop_benchstat/before
goos: darwin
goarch: arm64
cpu: Apple M4 Max
BenchmarkIsInternal-14    	1000000000	         0.2886 ns/op	       0 B/op	       0 allocs/op
BenchmarkIsInternal-14    	1000000000	         0.2887 ns/op	       0 B/op	       0 allocs/op
BenchmarkIsInternal-14    	1000000000	         0.2890 ns/op	       0 B/op	       0 allocs/op
BenchmarkFilter-14        	    7768	    144424 ns/op	   73883 B/op	    2875 allocs/op
BenchmarkFilter-14        	    7946	    145104 ns/op	   73884 B/op	    2875 allocs/op
BenchmarkFilter-14        	    8308	    144525 ns/op	   73884 B/op	    2875 allocs/op
```

三個數字都不能信。

**結果沒人用。** `IsInternal` 不到 1 ns，差不多就是迴圈自己的成本。引數是常數、結果沒人用，函式被 inline 之後整段沒有副作用，編譯器可以直接刪掉；量到的是空迴圈。把迴圈換成 `for b.Loop()` 再跑，數字會變大：`b.Loop` 迴圈裡呼叫的函式不會被 inline，引數與結果都保留（見 `go doc testing.B.Loop`）。

**計時範圍含準備工作。** `BenchmarkFilter` 每圈先產生 1000 個位址和 64 筆名單，`allocs/op` 高達兩千多次。`Filter` 本身只配置輸出的 slice 一次（`after` 的 `TestFilterAllocs` 對同樣寫法的 `FilterLinear` 斷言這件事），所以這一行的時間和配置幾乎都是準備工作的；改寫 `Filter` 之後，差異也會被準備工作稀釋。

**一次的數字沒有尺。** 三行 `BenchmarkFilter` 的 `ns/op` 並不相同，同一份程式碼，每次量都不一樣。改寫前後各跑一次、比較其中一行，若差距落在這種波動裡，什麼都證明不了；只有一個樣本時，連波動有多大都不知道。

## 較好的做法

新版的 `Filter` 先把名單放進集合：

```go,ignore
{{#include ../../../examples/chapter7/02_b_loop_benchstat/after/filter.go:filter}}
```

準備工作放在迴圈外，迴圈用 `b.Loop`（寫法前作教過，不重講）。子 benchmark 的名稱用 `key=value`，說得出是哪個版本、哪種輸入大小；名單都不在 `ips` 裡，每個位址都得比完整份名單：

```go,ignore
{{#include ../../../examples/chapter7/02_b_loop_benchstat/after/filter_bench_test.go:bench}}
```

最佳化不能改變行為。`after` 保留舊版 `FilterLinear`，測試讓兩個版本吃同樣的輸入，結果必須一致；配置次數用 `testing.AllocsPerRun` 斷言，`-race` 下也成立：

```go,ignore
{{#include ../../../examples/chapter7/02_b_loop_benchstat/after/filter_test.go:allocs}}
```

量測要重複。`-count=10` 讓每個 benchmark 有 10 個樣本，`-run='^$'` 不跑一般測試。`benchstat` 不進 `go.mod`，一律用固定版本的 `go run` 取得。平常比較前後是 `benchstat old.txt new.txt`；這裡新舊版本在同一個檔案裡，用 `-col /impl` 把名稱裡的 `impl=` 當成欄。輸出省略了 `pkg:` 行。

執行結果（某一次）：

```text
$ go test -run='^$' -bench=. -count=10 ./examples/chapter7/02_b_loop_benchstat/after > after.txt
$ go run golang.org/x/perf/cmd/benchstat@v0.0.0-20261009192801-be2c69fb417e -col /impl after.txt
goos: darwin
goarch: arm64
cpu: Apple M4 Max
                         │    linear     │                 set                  │
                         │    sec/op     │    sec/op     vs base                │
Filter/blocklist=4-14       4.260µ ± 17%   8.089µ ±  2%  +89.88% (p=0.000 n=10)
Filter/blocklist=64-14     90.074µ ±  1%   8.123µ ± 10%  -90.98% (p=0.000 n=10)
Filter/blocklist=1000-14   387.93µ ± 48%   17.91µ ±  1%  -95.38% (p=0.000 n=10)
geomean                     53.00µ         10.56µ        -80.08%
```

逐欄讀：

- 表頭的 `linear`、`set` 是兩個欄；`sec/op` 是單位，`benchstat` 把 `ns/op` 換成秒再挑合適的前綴（`µ` 是微秒）。
- `4.260µ ± 17%`：10 個樣本的中位數，`±` 是這個中位數的 95% 信賴區間。`±` 越大，表示樣本越分散、結果越不穩，這一欄就越不能拿來下結論。
- `vs base`：和第一欄比，變多少。`+89.88%` 是變慢，`-90.98%` 是變快。
- `p=0.000 n=10`：p 值是「差異其實只是雜訊」的可能性；低於 0.05（預設門檻）就標為顯著。`n=10` 是兩邊各用了幾個樣本。
- `geomean` 是各列的幾何平均，只是個概覽；每一列的方向才是重點。

這張表的結論只能是相對的：名單很小時，集合版本顯著比較慢（建集合有成本）；名單大了，顯著比較快。有沒有值得換，取決於實際的名單有多大。

`~` 長什麼樣子？拿同一份程式碼量兩批再比較。兩批交錯，機器的狀態才一樣；`run1.txt`、`run2.txt` 各十個樣本，輸出省略了 `pkg:` 行。

執行結果（某一次）：

```text
$ for _ in $(seq 10); do
>   go test -run='^$' -bench='Filter/impl=set/blocklist=64' ./examples/chapter7/02_b_loop_benchstat/after >> run1.txt
>   go test -run='^$' -bench='Filter/impl=set/blocklist=64' ./examples/chapter7/02_b_loop_benchstat/after >> run2.txt
> done
$ go run golang.org/x/perf/cmd/benchstat@v0.0.0-20261009192801-be2c69fb417e run1.txt run2.txt
goos: darwin
goarch: arm64
cpu: Apple M4 Max
                                │   run1.txt    │            run2.txt            │
                                │    sec/op     │    sec/op     vs base          │
Filter/impl=set/blocklist=64-14   11.068µ ± 32%   8.192µ ± 67%  ~ (p=0.631 n=10)
```

`~` 的意思是「在 p < 0.05 的標準下看不出差異」，不是「兩者相同」。`±` 很大時，連大的差距也會被判成 `~`：先看 `±`，再看 `~`。

## 取捨

- **micro-benchmark 只代表那一小段。** 這份 benchmark 用的是最壞的輸入：1000 個位址、名單全不命中。真實流量的名單大小與命中率不同，結論可能不同。它也不含網路、GC 壓力與其他程式的競爭，整套服務的表現要另外量。
- **同一台機器、同樣條件才能比。** p 值只涵蓋同一批樣本內的波動，不涵蓋兩批之間機器狀態的改變（其他程式在跑、CPU 降頻）。不要拿不同機器、不同時段的結果互比；要比就交錯執行，或乾脆在安靜的固定機器上量。
- **`-count` 有成本。** 越多樣本，`±` 越窄、越分得出小差異，但時間線性增加：after 有 6 個子 benchmark，每個預設要量 1 秒左右，`-count=10` 就是一分鐘起跳。差異本來就很大時（上表名單 1000 那一列），不需要很多樣本。
- **顯著不等於重要。** 樣本夠多，很小的差異也能顯著；改動值不值得，還要看這一段在整個請求裡佔多少，那要靠 profile。
- **before 在只要數量級時夠用。** 差十倍還是差一點點，粗略量一次就看得出來；`benchstat` 是為了差距小、或要當成 PR 證據的時候。
- 第 6 章〈[取消與 `context` 的傳遞](../chapter6/04_cancellation_context.md)〉留下「檢查的粒度用 `b.Loop` 量過再決定」，量法就是這集。benchmark 在 CI 怎麼跑、能擋什麼，本章〈把效能回歸擋在 CI 之外〉會談。

## 重點整理

- 不可信的 benchmark 有三種常見樣子：結果沒人用被編譯器刪掉、準備工作算進計時、只量一次；`b.Loop` 加上迴圈外的準備工作，可以避開前兩種。
- 比較前後用 `-count=10` 加 `benchstat`：看 `±`（穩不穩）、`vs base` 與 p 值（顯著嗎）；`~` 是「看不出差異」，不是「相同」。
- 用子 benchmark 比較不同的輸入大小，名稱寫成 `key=value`；方向可能因大小而相反，不要只量一種大小。
- 最佳化不能改變行為，要有新舊結果一致的測試；確定的結果（配置次數）用 `AllocsPerRun` 斷言，時間只用 `benchstat` 看。
- 只在同一台機器、同樣條件下比較；`benchstat` 用固定版本的 `go run`，不進 `go.mod`。
