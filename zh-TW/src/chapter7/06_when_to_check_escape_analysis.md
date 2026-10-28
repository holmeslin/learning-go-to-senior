# 什麼時候值得看逃逸分析

## 本集目標

只在量測已經指出「熱點函式每次呼叫都配置」時才看逃逸分析；用 `-gcflags=-m` 找出配置的原因，改成回傳值或 append 式 API，再用 `AllocsPerRun` 把「零配置」鎖進測試。

## 情境

服務每天處理大量存取日誌，指標系統要一個「方法、路徑、狀態碼類別」組成的路由鍵，例如 `GET /users/42 5xx`。`accesslog.Route(line)` 每一行日誌都呼叫一次，所以它是熱點。benchmark 的寫法前作教過：〈[benchmark 與 `b.Loop`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/12_benchmark.html)〉；`-gcflags=-m` 的讀法也教過：〈[逃逸分析](https://holmeslin.github.io/learning-go-from-zero/zh-TW/appendix2/b_escape_analysis.html)〉。這集談的是什麼時候值得看，以及看出原因之後怎麼改。

## 常見寫法

`ParseLine` 回傳指標，`Route` 先組出位元組，再轉成字串：

```go,ignore
{{#include ../../../examples/chapter7/06_when_to_check_escape_analysis/before/route.go:parse}}
{{#include ../../../examples/chapter7/06_when_to_check_escape_analysis/before/route.go:route}}
```

兩段程式都不難讀，測試也全過。完整程式碼在 `examples/chapter7/06_when_to_check_escape_analysis/before/`。

## 問題在哪

先量測，不要先看逃逸分析。下面的 benchmark 一次處理四行日誌，準備工作在迴圈外：

```go,ignore
{{#include ../../../examples/chapter7/06_when_to_check_escape_analysis/before/route_test.go:benchmark}}
```

加上 `-benchmem`，輸出省略了 `pkg:` 行與 `PASS`、`ok` 行。

執行結果（某一次）：

```text
$ go test -run='^$' -bench=. -benchmem ./examples/chapter7/06_when_to_check_escape_analysis/before
goos: darwin
goarch: arm64
cpu: Apple M4 Max
BenchmarkRoute-14    	 9504381	       126.5 ns/op	     272 B/op	       8 allocs/op
```

每個 op 處理四行，8 次配置，等於每一行 2 次。配置是這個熱點的成本，這時才值得問「為什麼配置」。如果 `allocs/op` 是 0，看 `-m` 沒有意義。

接著用 `go build -gcflags=-m` 問編譯器。輸出很長，下面只留成功路徑上的三行；省略了 `inlining call` 行、`leaking param` 行與錯誤路徑上的行，路徑前綴也省略了。

執行結果（某一次）：

```text
$ go build -gcflags=-m ./examples/chapter7/06_when_to_check_escape_analysis/before 2>&1 | grep -E 'Entry|string\(buf\)|make\('
route.go:37:9: &Entry{...} escapes to heap
route.go:50:13: make([]byte, 0, len(e.Method) + len(e.Path) + 6) does not escape
route.go:55:16: string(buf) escapes to heap
```

前兩行對上每行 2 次配置：

- `&Entry{...} escapes to heap`：`ParseLine` 回傳指標，`Entry` 得比這次呼叫活得久，只能放在 heap。
- `string(buf) escapes to heap`：`Route` 回傳字串，字串的內容同樣要活過呼叫。
- `make(...) does not escape`：緩衝區本身留在 stack，這一行不是成本。

省略的行裡還有 `status escapes to heap`，那是 `fmt.Errorf` 的參數轉成 `any`，只發生在錯誤路徑，不在熱點上。

## 較好的做法

原因是「結果要活過函式」，所以改法是不讓函式把結果配置出去。

**回傳值，不回傳指標。** `Entry` 只有兩個字串與一個整數，複製很便宜，值留在呼叫端的 stack 上。

**讓呼叫端傳入緩衝區。** `AppendRoute(dst, line)` 把結果附加在 `dst` 後面，沿用標準庫 `strconv.AppendInt` 一類 `Append` 函式的慣例；原本的 `Route` 留成包裝，不在乎配置的呼叫端不用改：

```go,ignore
{{#include ../../../examples/chapter7/06_when_to_check_escape_analysis/after/route.go:parse}}
{{#include ../../../examples/chapter7/06_when_to_check_escape_analysis/after/route.go:route}}
```

改寫不能改變行為：測試把同一批輸入（含五種壞輸入）餵給 before 與 after，比對結果與錯誤訊息，也檢查出錯時 `AppendRoute` 原樣回傳 `dst`：

```go,ignore
{{#include ../../../examples/chapter7/06_when_to_check_escape_analysis/after/route_test.go:same}}
```

零配置用 `testing.AllocsPerRun` 斷言。它只看配置次數，不看時間，所以在 `-race` 下也成立：

```go,ignore
{{#include ../../../examples/chapter7/06_when_to_check_escape_analysis/after/route_test.go:allocs}}
```

公開的用法與錯誤，用 `Example` 鎖住輸出：

```go,ignore
{{#include ../../../examples/chapter7/06_when_to_check_escape_analysis/after/route_test.go:example}}
```

對 after 再問一次編譯器，只留 `route.go` 裡和緩衝區有關的行：

執行結果（某一次）：

```text
$ go build -gcflags=-m ./examples/chapter7/06_when_to_check_escape_analysis/after 2>&1 | grep -E 'append|dst'
route.go:46:18: leaking param: dst to result ~r0 level=0
route.go:51:14: append escapes to heap
route.go:52:14: append escapes to heap
route.go:53:14: append escapes to heap
route.go:54:15: append escapes to heap
```

`append escapes to heap` 仍然出現，因為 `dst` 的容量不夠時，`append` 必須配置新的陣列。`-m` 回答「可能逃逸」，不回答「每次都配置」；容量夠的時候一次也不配置，這件事由 `AllocsPerRun` 的測試回答。呼叫端要重複使用緩衝區才有效，傳 `nil` 進去就回到每次配置（`Route` 包裝就是這樣）。

`BenchmarkRoute` 在 after 的寫法相同，只是重複使用 `buf`。兩邊各收 `-count=10`，輸出存成 `old.txt`、`new.txt`（用 `grep -v '^pkg:'` 去掉 `pkg:` 行，否則 `benchstat` 會把兩個套件當成不同組態，分開列表），再比較：

執行結果（某一次）：

```text
$ go run golang.org/x/perf/cmd/benchstat@v0.0.0-20261009192801-be2c69fb417e old.txt new.txt
goos: darwin
goarch: arm64
cpu: Apple M4 Max
         │    old.txt    │               new.txt               │
         │    sec/op     │   sec/op     vs base                │
Route-14   138.75n ± 12%   68.86n ± 6%  -50.37% (p=0.000 n=10)

         │  old.txt   │              new.txt              │
         │    B/op    │   B/op    vs base                 │
Route-14   272.0 ± 0%   0.0 ± 0%  -100.00% (p=0.000 n=10)

         │  old.txt   │               new.txt               │
         │ allocs/op  │ allocs/op   vs base                 │
Route-14   8.000 ± 0%   0.000 ± 0%  -100.00% (p=0.000 n=10)
```

`benchstat` 的判斷是三項都有顯著差異（`p=0.000`），時間下降、`B/op` 與 `allocs/op` 歸零。數字換一台機器會不同，可以信的是方向。`benchstat` 怎麼讀，本章〈`b.Loop` 與 `benchstat`：怎麼比較前後結果才可信〉談過。

## 取捨

- **只在熱點上做。** 沒有量測指出配置之前，不需要跑 `-m`；多數函式的配置不影響結果，改成 append 式 API 只會讓呼叫端多管一個緩衝區。找出迴圈裡的配置與拷貝，本章〈常見熱點：配置、拷貝、鎖競爭〉談過。
- **逃逸的決定隨編譯器版本改變。** 上面的 `-m` 輸出是 Go 1.27 的結果；升級之後，同一段程式的決定可能不同，所以別把輸出當成永遠成立的事實。`AllocsPerRun` 的測試就是為這件事留的：升級後如果某個熱點又開始配置，測試會失敗。
- **為了少一次配置把 API 改彆扭，通常不值得。** `AppendRoute` 可以接受，因為 `Append` 是標準庫看得懂的慣例，而且 `Route` 還在。如果要呼叫端預先備好一堆輸出參數或物件才能呼叫，換來的那一點時間多半不值得。
- **其他常見原因讀法相同。** 值存進介面（像上面的 `status`）、閉包捕捉變數、大小不固定的 slice，都是找 `escapes to heap` 那一行，再問「結果為什麼要活過這次呼叫」。
- **before 在多數時候是對的。** 不在熱路徑、配置不是瓶頸時，回傳指標與回傳字串最好讀。

## 重點整理

- 先用 benchmark 的 `-benchmem` 或 profile 確認熱點在配置，才值得看 `-gcflags=-m`。
- `escapes to heap` 的常見原因是結果要活過函式：回傳指標、回傳新字串；改成回傳值，或讓呼叫端傳入緩衝區的 append 式 API。
- `-m` 只說「可能逃逸」；零配置要用 `testing.AllocsPerRun` 在測試裡斷言，時間交給 `benchstat`。
- 逃逸的決定隨編譯器版本改變，用測試鎖住結果，不要背輸出。
- 為了少一次配置把 API 改彆扭通常不值得，只在熱點上做。
