# 什麼時候該調 `GOGC`，什麼時候別動

## 本集目標

看到「記憶體偏高」或「GC 吃 CPU」時，判斷該不該動 `GOGC`：先確認 GC 真的是瓶頸，先拿掉手動 `runtime.GC()`、減少配置，最後才依量測調整；並知道大多數服務用預設值就好。

## 情境

會員名冊服務啟動時把 20 萬位會員載進記憶體，之後每個請求帶一批會員 ID，回傳報表。監控上 heap 看起來偏高，有人想讓記憶體早點還回去。標記清除、`GOGC` 與 `GOMEMLIMIT` 前作教過：〈[GC 概念](https://holmeslin.github.io/learning-go-from-zero/zh-TW/appendix2/c_gc.html)〉。這集談的是什麼時候該動它。本集貼出的輸出，都在 macOS（arm64、Apple M4 Max、14 個 P）、Go 1.27 上收，每筆都是某一次的結果。

## 常見寫法

每批處理完就手動收一次 GC，想著「暫存資料不要堆在記憶體裡」：

```go,ignore
{{#include ../../../examples/chapter8/02_when_to_tune_gogc/before/roster.go:handle}}
```

另一種常見的做法是把環境變數 `GOGC` 設得很低（例如 10），逼 GC 勤快一點。完整程式碼在 `examples/chapter8/02_when_to_tune_gogc/before/`（只有 benchmark，沒有測試）。

## 問題在哪

用 `GODEBUG=gctrace=1` 讓 runtime 每次 GC 印一行到 stderr，負載是 3000 批、每批 2000 個 ID。每個欄位怎麼讀，本章〈讀 GC trace〉會談；這集只用三處：開頭的 `gc N`（第幾次 GC）、`N%`（程式啟動以來花在 GC 的時間比例）、結尾的 `(forced)`（由 `runtime.GC()` 觸發）。輸出只留最後一行（前面的行不貼），並數出總共幾次、其中幾次是 forced；結尾的 `14 P` 是這次執行的 P 數量。

```bash
go build -o ingest ./examples/chapter8/02_when_to_tune_gogc/after/cmd/ingest
GODEBUG=gctrace=1 ./ingest -impl=before 2>gc.txt
```

執行結果（某一次）：

```text
$ grep -E ' P( \(forced\))?$' gc.txt | tail -1
gc 3003 @5.360s 18%: 0.003+1.2+0.002 ms clock, 0.055+0/4.5/10+0.032 ms cpu, 14->14->14 MB, 28 MB goal, 0 MB stacks, 0 MB globals, 14 P (forced)
$ grep -E ' P( \(forced\))?$' gc.txt | grep -c forced
3000
$ grep -cE ' P( \(forced\))?$' gc.txt
3003
```

3003 次 GC 裡有 3000 次是手動觸發的，和批數一致。最後一行的 `14->14->14 MB` 依序是 GC 開始時的 heap、結束時的 heap 與存活的 heap：開始與結束一樣大，這次 GC 沒有收到東西，卻得把 14 MB 的名冊重新標記一遍。程式啟動以來花在 GC 的時間占 18%。這是 gctrace 的佐證，要再確認是哪些函式在花時間，用 CPU profile，收法見第 7 章〈[`pprof`：先看 CPU 還是 heap](../chapter7/03_pprof_cpu_or_heap.md)〉。

## 較好的做法

調 `GOGC` 前依序做三件事：先確認 GC 真的是瓶頸（上面的 forced 與占比）；再拿掉手動 GC、減少配置；最後才依量測調 `GOGC`。第二步，`Handle` 一次配好緩衝區，用 `strconv.AppendInt` 與 `append` 直接寫入，不再為每個 ID 組一個字串；回收交給 runtime。這些改法第 7 章〈[常見熱點：配置、拷貝、鎖競爭](../chapter7/05_common_hotspots.md)〉談過：

```go,ignore
{{#include ../../../examples/chapter8/02_when_to_tune_gogc/after/roster.go:handle}}
```

輸出照舊，由 `go test` 驗證：

```go,ignore
{{#include ../../../examples/chapter8/02_when_to_tune_gogc/after/example_test.go:example}}
```

改寫不能改變結果，測試拿 before 當基準；配置次數用 `AllocsPerRun` 斷言：

```go,ignore
{{#include ../../../examples/chapter8/02_when_to_tune_gogc/after/roster_test.go:same}}
{{#include ../../../examples/chapter8/02_when_to_tune_gogc/after/roster_test.go:allocs}}
```

before 與 after 的 `BenchmarkHandle` 各收 10 次，存成 `before.txt`、`after.txt` 再比較。兩份來自不同套件，所以加 `-ignore pkg`，`benchstat` 的讀法第 7 章〈[`b.Loop` 與 `benchstat`：怎麼比較前後結果才可信](../chapter7/02_b_loop_benchstat.md)〉談過：

執行結果（某一次）：

```text
$ go test -run='^$' -bench=. -benchmem -count=10 ./examples/chapter8/02_when_to_tune_gogc/before > before.txt
$ go test -run='^$' -bench=. -benchmem -count=10 ./examples/chapter8/02_when_to_tune_gogc/after > after.txt
$ go run golang.org/x/perf/cmd/benchstat@v0.0.0-20261009192801-be2c69fb417e -ignore pkg before.txt after.txt
goos: darwin
goarch: arm64
cpu: Apple M4 Max
          │  before.txt   │              after.txt              │
          │    sec/op     │   sec/op     vs base                │
Handle-14   1312.72µ ± 2%   41.04µ ± 1%  -96.87% (p=0.000 n=10)

          │  before.txt   │              after.txt               │
          │     B/op      │     B/op      vs base                │
Handle-14   255.12Ki ± 0%   64.00Ki ± 0%  -74.91% (p=0.000 n=10)

          │  before.txt   │             after.txt              │
          │   allocs/op   │ allocs/op   vs base                │
Handle-14   8001.000 ± 0%   1.000 ± 0%  -99.99% (p=0.000 n=10)
```

三項都顯著下降（p=0.000）。兩件事一起改了，所以時間的差距不分哪一件貢獻多少；配置次數從每個 ID 約 4 次降到每批 1 次。

第三步，名冊的 14 MB 是服務本來就要留著的。同一支 after 程式在三個 `GOGC` 下各跑一次，負載相同（3000 批），只取最後一行 gctrace。`GOGC=10` 是上面「調很低」的寫法，`GOGC=100` 是預設值：

執行結果（某一次）：

```text
$ GOGC=10 GODEBUG=gctrace=1 ./ingest 2>&1 >/dev/null | grep -E ' P( \(forced\))?$' | tail -1
gc 152 @0.340s 17%: 0.007+1.3+0.008 ms clock, 0.10+1.0/5.2/8.8+0.11 ms cpu, 15->15->14 MB, 15 MB goal, 0 MB stacks, 0 MB globals, 14 P
$ GOGC=100 GODEBUG=gctrace=1 ./ingest 2>&1 >/dev/null | grep -E ' P( \(forced\))?$' | tail -1
gc 17 @0.137s 4%: 0.026+1.3+0.010 ms clock, 0.37+0/4.9/10+0.14 ms cpu, 27->28->15 MB, 31 MB goal, 0 MB stacks, 0 MB globals, 14 P
$ GOGC=400 GODEBUG=gctrace=1 ./ingest 2>&1 >/dev/null | grep -E ' P( \(forced\))?$' | tail -1
gc 4 @0.127s 1%: 0.012+1.6+0.009 ms clock, 0.16+0/6.2/6.0+0.13 ms cpu, 77->79->15 MB, 82 MB goal, 0 MB stacks, 0 MB globals, 14 P
```

`gc N` 是從程式開始累計的次數，`goal` 是這次 GC 結束時 heap 的目標。三行的存活量差不多（結尾的 `MB` 都是 14 到 15，是那份名冊），差別在 `GOGC` 越小，GC 越勤、`goal` 越貼著存活量，GC 占比越高；`GOGC` 越大，次數與占比越低，代價是 `goal`，也就是 heap 可能長到的大小，跟著變大。第 7 章〈[`pprof`：先看 CPU 還是 heap](../chapter7/03_pprof_cpu_or_heap.md)〉留下的問題「GC 怎麼調」，答案就是這張取捨：用記憶體換 CPU。

## 取捨

- **大多數服務用預設值就好。** 以這個例子來說，拿掉手動 GC 並減少配置之後，預設值下的 GC 占比只剩個位數；這時調大 `GOGC` 能省的，最多就是剩下的那幾個百分點，換來更大的 heap。沒有 gctrace 或 profile 顯示 GC 占比高，就不要動。
- **記憶體夠、GC 又確實吃 CPU 時調大。** 用環境變數 `GOGC` 放在部署設定最簡單，不必改程式；程式裡用 `debug.SetGCPercent` 也行，前作教過。調完要用同樣的負載再量一次，結論只看相對變化。
- **記憶體受限時別靠 `GOGC`。** 它只看「新配置的量佔存活量的比例」，不知道容器還剩多少記憶體；調大會讓 `goal` 越過上限，調小只是多花 CPU。這時要用 `GOMEMLIMIT`，本章〈`GOMEMLIMIT` 與容器記憶體上限〉會談；`GOGC=off` 沒有上限保護，也在那一集談。
- **before 的手動 GC 偶爾是對的。** 一次性的大量載入之後、沒有請求在跑的時間點，或批次工具在特定階段想先收一次，單次 `runtime.GC()` 沒有問題；錯的是放進每個請求都會走的路徑。
- **`gctrace` 適合除錯，長期觀察用指標。** 它的格式可能隨版本改變，逐欄的讀法本章〈讀 GC trace〉會談；長期收 GC 的次數與占比，本章〈`runtime/metrics`〉會談。
- **貼出的數字只代表這台機器的某一次。** 存活量 14 MB、批次大小與 P 的數量都會改變比例；關係（`GOGC` 越大，次數越少、`goal` 越大）才是可以帶走的。

## 重點整理

- 調 `GOGC` 前的順序：確認 GC 是瓶頸（gctrace 的占比、CPU profile）、拿掉手動 `runtime.GC()`、減少配置，最後才調。
- 在請求路徑上呼叫 `runtime.GC()` 通常只會多付 CPU：存活的資料每次都要重新標記。
- `GOGC` 是用記憶體換 CPU：調大，GC 次數與占比下降，heap 目標變大；調小則相反。
- 大多數服務用預設值；記憶體受限時改用 `GOMEMLIMIT`，不是把 `GOGC` 調小。
- 數字都是某一台機器的某一次，結論只看相對變化。
