# `/debug/pprof` 與診斷端點

## 本集目標

判斷診斷端點該掛在哪：`net/http/pprof` 放在獨立的 `ServeMux`，只聽本機或內部網路的管理用 port，不和對外的服務共用 `http.DefaultServeMux`；並知道怎麼從執行中的服務收 CPU、heap、goroutine profile。

## 情境

庫存查詢服務上線後，監控看到一顆核心被吃滿、記憶體只增不減、goroutine 的數量慢慢上升。本機重現不了，只能從執行中的服務直接收 profile。`go tool pprof` 的基本用法前作教過：〈[`pprof`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/appendix2/d_pprof.html)〉；`ServeMux` 與 `httptest` 見〈[`ServeMux` 路由](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter14/02_servemux.html)〉、〈[`httptest`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter14/09_httptest.html)〉。profile 怎麼讀，第 7 章〈[`pprof`：先看 CPU 還是 heap](../chapter7/03_pprof_cpu_or_heap.md)〉談過。這集談的是端點掛在哪、開放給誰。

## 常見寫法

`net/http/pprof` 的文件這樣教：`import _` 一行，套件的 `init` 就把端點註冊到 `http.DefaultServeMux`。服務自己的路由也註冊在那裡，對外就是 `http.ListenAndServe(addr, nil)`：

```go,ignore
{{#include ../../../examples/chapter9/06_debug_pprof/before/stock.go:handler}}
```

本機開發一切正常，`curl localhost` 就收得到 profile。完整程式碼在 `examples/chapter9/06_debug_pprof/before/`。

## 問題在哪

`ListenAndServe(addr, nil)` 用的是 `DefaultServeMux`，所以對外的 handler 就是上面的 `NewHandler()`。對它發請求：

```go,ignore
{{#include ../../../examples/chapter9/06_debug_pprof/before/example_test.go:example}}
```

業務路由回 200，`/debug/pprof/` 也回 200，還列出所有 profile。連得到服務的人都能收：`profile?seconds=` 讓對方在你的服務上啟動 CPU profiling，`trace` 一樣；profile 裡有函式名稱與原始碼路徑，等於把內部結構交出去。

## 較好的做法

對外與管理分成兩個 handler，各用自己的 `ServeMux`。管理用的那個明確列出要開的端點：

```go,ignore
{{#include ../../../examples/chapter9/06_debug_pprof/after/stock.go:admin}}
```

`cmdline` 刻意不註冊：它回傳完整的命令列，參數裡可能有密碼，而 `go tool pprof` 用不到。對外的 `NewPublicHandler` 只有業務路由。測試證明兩邊各管各的：

```go,ignore
{{#include ../../../examples/chapter9/06_debug_pprof/after/example_test.go:example}}
```

```go,ignore
{{#include ../../../examples/chapter9/06_debug_pprof/after/handler_test.go:public}}
```

要留意：只要 import 了 `net/http/pprof`，不管有沒有 `_`，`DefaultServeMux` 都有診斷端點，下面的測試把這件事鎖住。所以安全來自對外的 server 不用 `DefaultServeMux`，不是來自沒有 import：

```go,ignore
{{#include ../../../examples/chapter9/06_debug_pprof/after/handler_test.go:defaultmux}}
```

兩個 handler 各接一個 listener，管理用的只聽本機。這個示範兩個都聽 loopback，方便在自己的機器上試；正式環境對外的會聽 `:8080` 之類的位址，管理用的維持本機或內部網路。示範程式監聽 port，只放在 `main` 套件，CI 只編譯、不執行：

```go,ignore
{{#include ../../../examples/chapter9/06_debug_pprof/after/cmd/demo/main.go:main}}
```

這個示範刻意帶了三種症狀：一個不停算雜湊的 goroutine、一塊 32 MiB 的常駐記憶體、八個永遠等不到訊息的 goroutine。用 `go run ./examples/chapter9/06_debug_pprof/after/cmd/demo` 啟動，另開終端機在 repo 根目錄執行下面的指令；`.out` 檔寫在目前的目錄，不要加進版本控制。以下的輸出都在 macOS（arm64，Apple M4 Max，Go 1.27.0）上取得。先確認兩個位址：

執行結果（某一次）：

```text
$ curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:18095/debug/pprof/
404
$ curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:18096/debug/pprof/
200
```

收 CPU profile，`seconds=5` 讓這個請求停五秒、收這五秒的樣本：

```bash
curl -s -o cpu.out 'http://127.0.0.1:18096/debug/pprof/profile?seconds=5'
```

以下每份 `go tool pprof` 輸出都只留表格，表頭以上的各行（`File`、`Type`、`Time`、`Duration`、`Showing`、`Dropped`，goroutine 那份還有 `Active filters` 與 `focus`）都省略了：

執行結果（某一次）：

```text
$ go tool pprof -top -nodecount=3 cpu.out
      flat  flat%   sum%        cum   cum%
    1030ms 45.58% 45.58%     1030ms 45.58%  crypto/internal/fips140/sha256.blockSHA2
     230ms 10.18% 55.75%     1380ms 61.06%  crypto/internal/fips140/sha256.(*Digest).Write
     200ms  8.85% 64.60%     1470ms 65.04%  crypto/internal/fips140/sha256.(*Digest).checkSum
```

前幾名都是標準庫的 `sha256`，往上找第一個自己寫的函式，flat 與 cum 怎麼讀，第 7 章〈[`pprof`：先看 CPU 還是 heap](../chapter7/03_pprof_cpu_or_heap.md)〉談過。heap 與 goroutine profile 預設是呼叫當下的快照，不必等：

```bash
curl -s -o heap.out http://127.0.0.1:18096/debug/pprof/heap
curl -s -o goroutine.out http://127.0.0.1:18096/debug/pprof/goroutine
```

執行結果（某一次）：

```text
$ go tool pprof -sample_index=inuse_space -top -nodecount=3 heap.out
      flat  flat%   sum%        cum   cum%
      32MB 95.52% 95.52%       32MB 95.52%  main.retain (inline)
    1.50MB  4.48%   100%     1.50MB  4.48%  runtime.mallocgc
         0     0%   100%       32MB 95.52%  main.main
```

`inuse_space` 直接指出誰留著 32 MB。goroutine profile 用 `-focus` 只留經過 `main.leak` 的呼叫鏈：

執行結果（某一次）：

```text
$ go tool pprof -top -cum -focus='main\.leak' goroutine.out
      flat  flat%   sum%        cum   cum%
         0     0%     0%          8 61.54%  main.leak.func1
         0     0%     0%          8 61.54%  runtime.chanrecv
         0     0%     0%          8 61.54%  runtime.chanrecv1
         8 61.54% 61.54%          8 61.54%  runtime.gopark
```

八個 goroutine 停在同一個函式的同一個 channel 接收，和第 6 章〈[goroutine 洩漏怎麼發生、怎麼在測試裡抓到](../chapter6/09_goroutine_leak.md)〉的洩漏同一個長相，差別是這裡不必等測試。單看一次的數字分不出「本來就有」還是「在長」，每隔一陣收一份，同一個函式的數字一直增加才是洩漏。懶得自己比兩份，就加 `seconds=N`：文件寫明 heap、goroutine 等 profile 帶 `seconds` 會回傳這 N 秒的差量（delta），只看增加的部分。

## 取捨

- **CPU profile 只在收的期間有成本。** 取樣期間 runtime 要額外記錄，期間之外沒有；同一個行程同一時間只能有一個，`StartCPUProfile` 在已經啟用時會回傳錯誤。所以端點可以常開，收的時候用短的 `seconds`，不要預設的 30 秒掛著。
- **heap 與 goroutine profile 預設是快照，也能收差量。** 不帶 `seconds` 是當下的快照，隨時可以收；帶 `seconds=N`（allocs、block、goroutine、heap、mutex、threadcreate 都支援）是這 N 秒的差量，適合追趨勢。兩種都要放在管理用的 port 上，內容和 CPU profile 一樣敏感。
- **`expvar` 不是 profile。** `/debug/vars` 輸出 JSON 的計數，套件文件寫明預設有 `cmdline` 與 `memstats`，也可以用 `expvar.Publish` 加自己的；它回答「數字是多少」，不回答「哪個函式」。它同樣在 import 時註冊到 `DefaultServeMux`，要放進管理用的 mux，用 `expvar.Handler()`；而且 `cmdline` 也在其中，上面不註冊 `cmdline` 的考量一樣適用：命令列有秘密，就別開 `/debug/vars`。
- **偶發的問題等到發生才收 profile 就太晚。** 第 7 章〈[`runtime/trace`](../chapter7/04_runtime_trace.md)〉談過的 flight recorder 只在記憶體裡保留最近一段 trace，偵測到異常才存檔；存下來的檔不要經由公開端點給人下載。管理端點的 `trace?seconds=` 則是從收的那一刻起錄一段。
- **存取控制比「只聽本機」多一層。** loopback 擋得住外面，擋不住同一台機器或同一個網路命名空間裡的其他行程。管理用的 port 放在內部網路時，要靠網路層（防火牆、跳板機、`ssh -L` 轉發）或在 handler 前加驗證；這集不教這些設定。用 HTTP 端點調 `slog.LevelVar` 的日誌層級（本章〈[日誌層級與欄位設計](02_log_levels_and_fields.md)〉談過 `LevelVar`）也是管理操作，那個端點一樣掛在管理用的 mux。
- **before 什麼時候是對的？** 本機開發，或整個行程只有自己會連的小工具、沒有對外的 listener，一行 `import _` 最省事。

## 重點整理

- `import _ "net/http/pprof"` 把端點註冊到 `DefaultServeMux`；服務用同一個 mux 對外，profile 就跟著公開。
- 診斷端點放在獨立的 `ServeMux`、明確註冊要開的 handler，只聽本機或內部網路；對外的 handler 用測試證明回 404。
- import `net/http/pprof` 本身就會註冊到 `DefaultServeMux`，所以重點是對外的 server 不用它。
- 正式環境用 `curl` 收 profile、`go tool pprof` 讀；CPU profile 用短的 `seconds`，goroutine 的數字要隔一陣收兩份才看得出洩漏。
- 診斷資料有成本也有風險：`expvar` 與 flight recorder 各有定位，存取控制不能只靠「只聽本機」。
