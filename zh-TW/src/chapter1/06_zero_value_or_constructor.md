# 什麼時候讓零值可用，什麼時候該強制用建構函式

## 本集目標

設計型別時，判斷它該讓零值直接可用，還是該把建構函式變成唯一的入口。

## 情境

你在寫一個小套件 `tally`，統計每個頁面的造訪次數，再把結果輸出成報表。有兩個型別：`Counter` 記次數，`Reporter` 把次數寫到某個 `io.Writer`。零值為什麼值得設計，前作〈[讓零值有用](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter3/15_useful_zero_value.html)〉已經講過，這集只談什麼時候該做、什麼時候不該做。

## 常見寫法

兩個型別都把欄位設成不匯出，再各配一個建構函式：

```go,ignore
{{#include ../../../examples/chapter1/06_zero_value_or_constructor/before/tally.go:counter}}
```

```go,ignore
{{#include ../../../examples/chapter1/06_zero_value_or_constructor/before/tally.go:reporter}}
```

完整程式碼在 `examples/chapter1/06_zero_value_or_constructor/before/`。

## 問題在哪

建構函式只是慣例，編譯器不會逼呼叫端使用它。呼叫端寫 `var c tally.Counter`，或把 `Counter` 當成別的 struct 的欄位，都能編譯、也能通過 `go vet`，直到執行才出事：

```go,ignore
var c tally.Counter
c.Add("/home")
```

執行結果：

```text
panic: assignment to entry in nil map

goroutine 1 [running]:
example.com/demo/tally.(*Counter).Add(...)
	tally/tally.go:25
main.main()
	cmd1/main.go:7 +0x38
exit status 2
```

`Reporter` 也一樣，但訊息更難讀：

```go,ignore
var r tally.Reporter
r.Report(c)
```

堆疊裡的位址每次不同，檔案路徑前綴已省略。

執行結果（某一次）：

```text
panic: runtime error: invalid memory address or nil pointer dereference
[signal SIGSEGV: segmentation violation code=0x2 addr=0x18 pc=0x100ccec48]

goroutine 1 [running]:
fmt.Fprintf({0x0, 0x0}, {0x100cd6ced, 0x6}, {0xd2bdc19ad30, 0x2, 0x2})
	.../src/fmt/print.go:217 +0x58
example.com/demo/tally.(*Reporter).Report(0xd2bdc19ad88, 0xd2bdc19ae90)
	tally/tally.go:57 +0xbc
main.main()
	cmd2/main.go:9 +0xb0
exit status 2
```

堆疊停在 `fmt.Fprintf`，錯誤訊息裡看不出是哪個欄位沒設。before 目錄的測試把這兩種情況寫成 `TestZeroValuePanics`，確認它們真的會 panic：

```go,ignore
{{#include ../../../examples/chapter1/06_zero_value_or_constructor/before/tally_test.go:test}}
```

這兩個 panic 的成因不同，該用不同的方法處理。

## 較好的做法

`Counter` 的欄位只是一個 map，不需要外部資訊就能建立，所以讓零值可用，在第一次寫入時才建立 map（lazy 初始化）。`NewCounter` 也可以拿掉：

```go,ignore
{{#include ../../../examples/chapter1/06_zero_value_or_constructor/after/tally.go:counter}}
```

讀取 nil map 是安全的，`Count` 與 `Paths` 不用另外處理。呼叫端怎麼用都不會 panic，包括嵌在別的 struct 裡：

```go,ignore
{{#include ../../../examples/chapter1/06_zero_value_or_constructor/after/tally_test.go:test}}
```

`Reporter` 不一樣：它需要一個 `io.Writer`，這是呼叫端才知道的資訊，沒有合理的預設值，零值不可能有用。Go 沒辦法強制呼叫端用建構函式，不匯出型別、`var r tally.reporter` 這類寫法也只是換來更難用的 API。比較實際的做法，是讓誤用又便宜又明確：型別照樣匯出、欄位不匯出，文件註解寫明「請用 `NewReporter`」；萬一拿到零值，方法回傳一個說得清楚的錯誤，而不是 nil pointer panic。

```go,ignore
{{#include ../../../examples/chapter1/06_zero_value_or_constructor/after/tally.go:reporter}}
```

測試鎖住這個行為，零值與 `NewReporter(nil)` 都一樣：

```go,ignore
{{#include ../../../examples/chapter1/06_zero_value_or_constructor/after/tally_test.go:zeroreporter}}
```

錯誤訊息直接說出原因與解法：

執行結果：

```text
tally: reporter has no writer, create it with NewReporter
```

兩種做法放在一起，就是呼叫端看到的樣子，`Example` 由 `go test` 驗證：

```go,ignore
{{#include ../../../examples/chapter1/06_zero_value_or_constructor/after/example_test.go:example}}
```

完整程式碼在 `examples/chapter1/06_zero_value_or_constructor/after/`。

## 取捨

讓零值可用不是免費的：

- 每個寫入的方法都要檢查是否已初始化，漏掉一個就是 panic。
- 零值可用的型別通常要靠指標接收者，不要複製它：初始化之後複製，兩份會共用同一個 map；初始化之前複製，兩份又各自獨立。

把建構函式當作預期的入口也有代價：

- 擋不住誤用，只能讓它被發現得早、讀得懂：每個方法開頭都要檢查，漏掉一個就又回到 panic。
- `Report` 多了 `error` 回傳值，呼叫端要處理；`NewReporter(nil)` 也要到 `Report` 才報錯。要在建構時就擋，得讓 `NewReporter` 回傳 `error`，這是額外的成本。

before 不加檢查的做法在這些情況下反而可以：型別只在自己的套件內部使用，呼叫端都是你自己；或是型別有複雜的不變條件，需要建構函式驗證參數，這時匯出型別、不匯出欄位、附一個建構函式，是常見而且夠用的做法。

判斷方法：問自己「不給任何參數，這個型別有沒有合理的預設行為？」有，就讓零值可用；沒有，就用建構函式。

## 重點整理

- 零值可用與強制建構函式是兩種都對的設計，依型別「有沒有合理的預設行為」選擇。
- 欄位只是 map、slice 這類不需要外部資訊的資源，用 lazy 初始化讓零值可用；讀取 nil map 是安全的，只有寫入會 panic。
- 型別依賴呼叫端才知道的東西（`io.Writer`、`*sql.DB`），零值不可能有用：匯出型別、不匯出欄位，文件註解寫「請用建構函式」。
- Go 沒辦法強制呼叫端用建構函式；資深的做法是讓誤用便宜又明確：零值的方法回傳清楚的錯誤，不要等到 nil pointer panic。
- 強制建構函式的成本是每個方法都要檢查、多一個 `error` 要處理；零值可用的成本是每個寫入方法都要檢查初始化。
