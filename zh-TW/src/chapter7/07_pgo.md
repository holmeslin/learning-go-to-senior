# PGO

## 本集目標

看到「熱點是一個介面呼叫」時，判斷值不值得用 PGO：會收一份有代表性的 profile、存成 `default.pgo`、確認編譯器真的套用它，再用 `benchstat` 判斷有沒有變快。

## 情境

`accesscount` 讀一批存取紀錄，依規則統計筆數與流量。規則是介面 `Rule`，有 `5xx`（狀態碼 500 以上）和 `path:/api/`（路徑前綴，忽略大小寫）兩種，規則在執行時由參數決定。它是監控用的工具，線上幾乎都是 `5xx`。benchmark 與 pprof 前作教過：〈[benchmark 與 `b.Loop`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/12_benchmark.html)〉、〈[`pprof`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/appendix2/d_pprof.html)〉；`benchstat` 的讀法本章〈[`b.Loop` 與 `benchstat`：怎麼比較前後結果才可信](02_b_loop_benchstat.md)〉談過，怎麼從 benchmark 收 profile 本章〈[`pprof`：先看 CPU 還是 heap](03_pprof_cpu_or_heap.md)〉談過。

熱點程式放在 `rules` 套件，`main` 只讀參數。因為 PGO 靠函式名稱對上 profile：從測試收到的 profile 裡，`main` 套件的函式帶完整的 import path，`go build` 建出的執行檔裡卻叫 `main.…`，對不上就不會套用。

## 常見寫法

統計迴圈對每筆紀錄呼叫一次 `rule.Match`，建置時沒有 profile：

```go,ignore
{{#include ../../../examples/chapter7/07_pgo/before/rules/rules.go:tally}}
```

規則到執行時才知道，編譯器看不出 `rule` 是哪個型別，只能走介面呼叫，`Match` 的內容也無法 inline 進迴圈。完整程式碼在 `examples/chapter7/07_pgo/before/`（`main` 套件加 `rules` 套件）。

## 問題在哪

這不是配置問題。`-benchmem` 顯示兩種規則都是 0 allocs/op，成本花在每筆紀錄一次的間接呼叫。輸出省略了 `pkg:`、`PASS` 與 `ok` 行。

執行結果（某一次）：

```text
$ go test -run='^$' -bench=. -benchmem ./examples/chapter7/07_pgo/before/rules
goos: darwin
goarch: arm64
cpu: Apple M4 Max
BenchmarkTally/rule=path_prefix-14         	    3608	    328538 ns/op	       0 B/op	       0 allocs/op
BenchmarkTally/rule=server_error-14        	   10000	    103209 ns/op	       0 B/op	       0 allocs/op
```

這是基準線，不是錯誤。這個呼叫值不值得省，要看 profile 怎麼說。

## 較好的做法

**一、收 profile。** 用貼近線上的負載：線上幾乎都是 `5xx`，所以只跑這個子 benchmark，存成 main 套件旁的 `default.pgo`。benchmark 用 `b.Loop`，準備工作在迴圈外：

```go,ignore
{{#include ../../../examples/chapter7/07_pgo/after/rules/rules_test.go:bench}}
```

```bash
go test -run='^$' -bench='Tally/rule=server_error' -benchtime=2s \
  -cpuprofile=examples/chapter7/07_pgo/after/default.pgo -o /dev/null ./examples/chapter7/07_pgo/after/rules
```

`default.pgo` 就是一般的 CPU profile，大約 1 KB，commit 進 repo；程式或負載改了，就重跑這兩行。讀它用 `go tool pprof -top -nodecount=2 examples/chapter7/07_pgo/after/default.pgo`，輸出省略了 `Time:`、`Duration:` 這幾行。

執行結果（某一次）：

```text
File: rules.test
Type: cpu
Showing nodes accounting for 2270ms, 98.70% of 2300ms total
Showing top 2 nodes out of 6
      flat  flat%   sum%        cum   cum%
    1680ms 73.04% 73.04%     2300ms   100%  github.com/holmeslin/learning-go-to-senior/examples/chapter7/07_pgo/after/rules.Tally
     590ms 25.65% 98.70%      610ms 26.52%  github.com/holmeslin/learning-go-to-senior/examples/chapter7/07_pgo/after/rules.(*ServerError).Match
```

一行的 `Match` 自己占了一份，代表它是一次真的呼叫，沒有被 inline。

**二、確認 profile 真的被套用。** `go build` 預設是 `-pgo=auto`，會在 main 套件目錄找 `default.pgo`；`go test` 不會，上面的 profile 放在 `after/`，所以測試要明講 `-pgo=`。確認用 `-gcflags=-m`，只留 PGO 做的決定（輸出省略了其他行）：

```bash
go build -o /dev/null -pgo=off -gcflags='github.com/holmeslin/...=-m' ./examples/chapter7/07_pgo/after 2>&1 | grep -E 'PGO|inlining call to \(\*ServerError\)'
go build -o /dev/null -pgo=auto -gcflags='github.com/holmeslin/...=-m' ./examples/chapter7/07_pgo/after 2>&1 | grep -E 'PGO|inlining call to \(\*ServerError\)'
```

第一行沒有輸出：沒有 profile，就沒有這兩種決定。第二行的輸出：

執行結果（某一次）：

```text
examples/chapter7/07_pgo/after/rules/rules.go:76:16: PGO devirtualizing interface call rule.Match to (*ServerError).Match
examples/chapter7/07_pgo/after/rules/rules.go:76:16: inlining call to (*ServerError).Match
```

這就是 PGO 做的兩件事：profile 說這個呼叫幾乎都送給 `*ServerError`，編譯器就在前面加一個型別檢查，符合時改成直接呼叫（去虛擬化），再把這個熱點呼叫 inline 進迴圈。另外 `go version -m` 看建出的執行檔，會多一行 `build -pgo=…/default.pgo`。

**三、比較有沒有 profile。** 同一份程式，只差 `-pgo`，各收十次樣本，再交給 `benchstat`（讀法見前述那集）：

```bash
go test -run='^$' -bench=. -count=10 -pgo=off ./examples/chapter7/07_pgo/after/rules > off.txt
go test -run='^$' -bench=. -count=10 -pgo=examples/chapter7/07_pgo/after/default.pgo ./examples/chapter7/07_pgo/after/rules > on.txt
go run golang.org/x/perf/cmd/benchstat@v0.0.0-20261009192801-be2c69fb417e off.txt on.txt
```

輸出省略了 `pkg:` 行。

執行結果（某一次）：

```text
goos: darwin
goarch: arm64
cpu: Apple M4 Max
                           │   off.txt    │               on.txt                │
                           │    sec/op    │   sec/op     vs base                │
Tally/rule=path_prefix-14     327.0µ ± 1%   357.1µ ± 1%   +9.20% (p=0.000 n=10)
Tally/rule=server_error-14   102.35µ ± 0%   79.86µ ± 1%  -21.98% (p=0.000 n=10)
geomean                       182.9µ        168.9µ        -7.70%
```

profile 裡的 `5xx` 規則，`benchstat` 判斷顯著變快（`-`、p 值遠小於 0.05）。profile 沒看到的路徑前綴規則沒有受惠，`benchstat` 這次的判斷甚至是變慢：去虛擬化只替 profile 裡最熱的對象，其他規則要多過一次型別檢查，才走回原本的介面呼叫。

PGO 只改機器碼，不改程式碼。after 的測試鎖住兩件事：配置次數（有沒有 profile 都不會變）與輸出。`before/rules` 有同一個 `Example`，兩邊的結果必須一致：

```go,ignore
{{#include ../../../examples/chapter7/07_pgo/after/rules/rules_test.go:allocs}}
{{#include ../../../examples/chapter7/07_pgo/after/rules/rules_test.go:example}}
```

## 取捨

- **改善通常有限。** 上面的幅度來自一個只有一個迴圈的 benchmark，不能搬到整支服務。官方文件〈[Profile-guided optimization](https://go.dev/doc/pgo)〉的說法是一般程式約 2% 到 14%，常見的是個位數。profile 裡沒有明顯的小呼叫熱點，就不會有感覺。
- **profile 要有代表性。** 上面的路徑前綴規則就是例子；官方文件也說，不具代表性的 profile 幾乎得不到改善。正式環境的 profile 怎麼收，第 9 章〈`/debug/pprof` 與診斷端點〉會談；從 benchmark 收的，要持續跟實際的負載比對。
- **程式改了要更新。** profile 靠函式名稱與行號對應，改動之後對不上的部分就失效，而且沒有錯誤訊息。重新產生的指令要寫在文件或 Makefile 裡，並確認 `-gcflags=-m` 還看得到 PGO 的決定。
- **建置時間會變長。** 每個編譯單元都要讀 profile，官方文件說可以量到建置時間增加。這個範例太小，量不出差別；自己的程式用 `time go build -pgo=off` 與 `time go build -pgo=auto` 比較。
- **`default.pgo` 是二進位檔。** diff 看不出內容，審查時只能看大小與產生指令。

before 在什麼情況下是對的？熱點不在介面呼叫上、profile 不好收、或一次性的命令列工具，沒有 profile 的建置更簡單，也更容易重現。

## 重點整理

- PGO 用 CPU profile 讓編譯器對熱點呼叫做去虛擬化與 inline；它改機器碼，不改程式的行為。
- 流程：用有代表性的負載收 profile，存成 main 套件旁的 `default.pgo`，重新建置，再用 `benchstat` 比較有無 profile。
- 用 `-gcflags=-m` 與 `go version -m` 確認 profile 真的套用，不要只看速度。
- profile 沒看到的路徑不會變快，甚至可能變慢；改善通常是個位數百分比，程式改了就要重新產生。
