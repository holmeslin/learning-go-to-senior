# 什麼時候可以 `panic`，什麼時候一定要回傳 `error`

## 本集目標

分辨「程式寫錯了」與「執行時會發生的狀況」：前者可以 `panic`，後者一定要回傳 `error`；並知道 `Must` 開頭的函式在什麼條件下是正當的 `panic`。

## 情境

你寫了一個 `sched` 套件，把 `"every 5m"` 這種字串解析成排程。有的字串寫死在程式裡，有的來自設定檔或使用者的請求。別的團隊把它拿去用，你管不到他們怎麼呼叫。

`panic` 與 `recover` 的語法，前作〈[`panic`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter5/12_panic.html)〉與〈[`recover`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter5/13_recover.html)〉已經講過。這集只談什麼時候該用。

## 常見寫法

輸入不合法就 `panic`，呼叫端不用寫 `if err != nil`，用起來很乾淨：

```go,ignore
{{#include ../../../examples/chapter2/06_panic_or_error/before/sched/sched.go:parse}}
```

完整程式碼在 `examples/chapter2/06_panic_or_error/before/`。

## 問題在哪

字串寫死在程式裡時，這沒問題。問題是字串來自使用者：它寫錯不是程式的 bug，是每天都會發生的事，而 panic 的影響範圍是整個程式。下面把 `Parse` 放進 goroutine 裡呼叫（為了單檔可以執行，這裡內嵌一份同樣會 panic 的 `parse`），`main` 還特地包了 `recover`：

```go,exit=2
package main

import (
	"fmt"
	"strings"
	"time"
)

// 和 before/sched 的 Parse 一樣：格式不對就 panic（這裡省略了間隔必須為正的檢查）。
func parse(spec string) time.Duration {
	rest, ok := strings.CutPrefix(spec, "every ")
	if !ok {
		panic(fmt.Sprintf("parse schedule %q: want \"every <duration>\"", spec))
	}
	d, err := time.ParseDuration(rest)
	if err != nil {
		panic(fmt.Sprintf("parse schedule %q: %v", spec, err))
	}
	return d
}

func main() {
	// main 這一層包了 recover，看起來很安全。
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("recovered:", r)
		}
	}()

	done := make(chan struct{})
	go func() {
		fmt.Println("interval:", parse("evry 10m"))
		done <- struct{}{}
	}()
	<-done
	fmt.Println("done")
}
```

執行結果（某一次）：

```text
panic: parse schedule "evry 10m": want "every <duration>"

goroutine 6 [running]:
main.parse(...)
main.main.func2()
created by main.main in goroutine 1
```

程式以結束碼 2 結束，`recovered:` 和 `done` 都沒有印出來。`recover` 只能攔下同一個 goroutine 裡的 panic，`main` 的 `defer` 管不到別的 goroutine，所以一個請求寫錯字串，整個服務跟著結束（goroutine 的細節第 6 章會談）。輸出省略了檔案路徑與參數位址，goroutine 的編號每次不一定相同。

套件的作者沒辦法預期呼叫端會在哪裡呼叫它，所以不能把「輸入不合法」留給呼叫端去 `recover`。

## 較好的做法

先問：這個錯是程式寫錯（不變條件被破壞），還是執行時會發生的狀況？後者一律回傳 `error`：

```go,ignore
{{#include ../../../examples/chapter2/06_panic_or_error/after/sched/sched.go:parse}}
```

呼叫端拿到的錯誤訊息，由 `Example` 的 `// Output:` 逐字驗證：

```go,ignore
{{#include ../../../examples/chapter2/06_panic_or_error/after/sched/sched_test.go:example}}
```

字串寫死在程式裡的人，不想為了一個常數寫 `if err != nil`，而且寫錯了也只能是 bug。這種情況才另外提供 `Must` 開頭的版本，轉呼 `Parse`，出錯就 `panic`：

```go,ignore
{{#include ../../../examples/chapter2/06_panic_or_error/after/sched/sched.go:must}}
```

用在套件層級的變數上，程式一啟動就會發現寫錯（和 `regexp.MustCompile` 的用法一樣）：

```go,ignore
{{#include ../../../examples/chapter2/06_panic_or_error/after/sched/sched_test.go:must}}
```

完整程式碼與測試在 `examples/chapter2/06_panic_or_error/after/`，測試涵蓋 `MustParse` 對合法字串回傳、對不合法字串 panic，且 panic 的值就是 `Parse` 回傳的 error。

## 取捨

不變條件被破壞時，`panic` 比回傳一個沒人處理的錯誤誠實。如果 `MustParse` 改成回傳 `(Schedule, error)`，套件層級的變數宣告沒有地方寫 `if err != nil`，只能 `nightly, _ = ...` 把錯誤吞掉；`Parse` 出錯時回傳零值 `Schedule`，之後整個程式就帶著 `Interval` 為 0 的排程默默跑下去。`panic` 讓問題在啟動的當下浮現，堆疊指向寫錯的那一行。

`Must` 版本也有界線：

- 只給寫死的常數。`MustParse(req.Spec)` 等於把 before 的問題請回來。
- 多一個匯出的函式要維護，而且要讓它和 `Parse` 的行為一致，所以它只該轉呼 `Parse`。
- 標準庫的 `regexp.MustCompile` 與 `template.Must` 也是這個慣例：給寫死的字串用，另有回傳 `error` 的版本。

before 在什麼情況下反而是對的：所有輸入都在你自己控制之下的小工具或腳本，寫錯就是要立刻停下來，這時 `panic` 比到處傳 `error` 省事。函式庫的內部也一樣：標準庫的 `time.NewTicker` 收到零或負的間隔就 `panic`，因為間隔必須為正，是呼叫端寫程式時就該保證的前提；傳進 0 是程式寫錯，不是執行時會發生的狀況，回傳 `error` 也沒有呼叫端處理得了。

## 重點整理

- 先分辨錯誤的來源：程式寫錯（不變條件被破壞）可以 `panic`；使用者輸入、檔案、網路這類執行時狀況一律回傳 `error`。
- 函式庫不能假設呼叫端會 `recover`。`recover` 只攔得住同一個 goroutine 的 panic，別的 goroutine 裡的 panic 會讓整個程式結束。
- 只在確定輸入是寫死的常數時，才提供 `Must` 開頭的版本，並且轉呼回傳 `error` 的版本。
- `Must` 版本用在初始化：寫錯就在啟動時讓程式停下來，比吞掉錯誤或帶著零值繼續跑好。
