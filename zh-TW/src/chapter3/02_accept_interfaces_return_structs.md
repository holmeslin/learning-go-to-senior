# 接受介面、回傳 struct

## 本集目標

設計函式簽名時，判斷兩端各該用什麼：參數接受介面，讓呼叫端傳入任何符合的實作；回傳值用具體型別，讓呼叫端拿得到完整的行為。並知道什麼時候回傳介面才是對的。

## 情境

你寫了一個 `audit` 套件，把稽核事件寫到檔案。寫入先經過緩衝，所以除了 `Log` 之外還有 `Flush`：寫完要呼叫它，緩衝區裡的紀錄才會真的送出去。別的服務與它們的測試都會用這個套件。

介面與隱式實作，前作〈[`interface`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter4/01_interface.html)〉與〈[隱式實作](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter4/02_implicit_implementation.html)〉已經講過。這集只談函式簽名的兩端；介面該宣告在哪個套件，本章〈在使用端定義介面〉談過，這集不重複。

## 常見寫法

建構函式回傳介面，實作藏在未匯出的型別後面，呼叫端只看得到 `Log`。參數寫成 `*os.File`，因為這個套件本來就是寫到檔案：

```go,ignore
{{#include ../../../examples/chapter3/02_accept_interfaces_return_structs/before/audit.go:logger}}
```

完整程式碼在 `examples/chapter3/02_accept_interfaces_return_structs/before/`。

## 問題在哪

先看呼叫端。它要在結束前 `Flush`，也想在測試裡用 `bytes.Buffer` 收輸出（`audit` 就是 `before`）：

```text
package main

import (
	"bytes"
	"fmt"
	"os"

	"example.com/app/audit"
)

func main() {
	l := audit.New(os.Stdout)
	l.Log("login")
	fmt.Println(l.Flush())

	var buf bytes.Buffer
	audit.New(&buf)
}
```

編譯錯誤：

```text
./main.go:14:16: l.Flush undefined (type audit.Logger has no field or method Flush)
./main.go:17:12: cannot use &buf (value of type *bytes.Buffer) as *os.File value in argument to audit.New
```

兩個錯誤是同一個毛病的兩端：

- 回傳端：`New` 回傳 `Logger`，介面裡只有 `Log`。`Flush` 明明在背後的型別上，呼叫端卻碰不到，而 `bufio.Writer` 的文件要求寫完要 `Flush`，資料才會送出。唯一的出路是把 `Flush` 加進 `Logger`，但介面多一個方法，每個實作過 `Logger` 的人（例如測試裡的替身）都會編譯失敗，第 1 章〈[匯出名稱與 API 穩定性](../chapter1/04_exported_api_stability.md)〉示範過。
- 參數端：收 `*os.File`，就只能傳檔案。緩衝區、網路連線、測試用的 writer 都傳不進去。

回傳介面還有更隱蔽的問題。參數只收檔案，「不記錄」就沒有別的表達方式，只好約定 `f` 為 nil 時回傳 nil，呼叫端也照著檢查。`Example` 用 `recover` 攔下 panic，才印得出結果：

```go,ignore
{{#include ../../../examples/chapter3/02_accept_interfaces_return_structs/before/example_test.go:typednil}}
```

`l == nil` 是 `false`。`New` 裡的 `l` 是 `*bufLogger`，`return l` 時被轉成 `Logger`，介面裡裝著「型別 `*bufLogger`、值為 nil 指標」，所以不等於 nil（原理見前作〈[nil 介面陷阱](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter4/11_nil_interface_trap.html)〉）。`if l != nil` 擋不住，`Log` 在 nil 指標上讀 `l.w`，就 panic。「先宣告具體型別的變數、依條件指派、最後統一 `return`」是建構函式裡常見的寫法；只要回傳型別是介面，編譯器不會提醒，`go vet` 也照樣通過。

## 較好的做法

回傳具體型別，參數接受介面：

```go,ignore
{{#include ../../../examples/chapter3/02_accept_interfaces_return_structs/after/audit.go:logger}}
```

三個差別：

- 回傳 `*Logger`：`Flush` 呼叫端拿得到。日後再加匯出的方法，只是多一個方法，不會讓既有的呼叫端編譯失敗，也就是第 1 章說的「只新增、不修改」。
- 函式裡沒有任何「具體型別轉成介面」的地方，nil 指標不會變成不等於 nil 的介面。
- 參數收 `io.Writer`，一個方法的小介面（基本原則見前作〈[小介面設計](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter4/10_small_interfaces.html)〉）。呼叫端傳什麼都行，不想記錄就傳 `io.Discard`，不用再靠 nil 表示。

外部測試用三種不同的 `io.Writer` 呼叫同一個 `New`，證明參數接受任何符合的實作：

```go,ignore
{{#include ../../../examples/chapter3/02_accept_interfaces_return_structs/after/audit_test.go:test}}
```

before 要靠 nil 表示的「不記錄」，現在是：

```go,ignore
{{#include ../../../examples/chapter3/02_accept_interfaces_return_structs/after/example_test.go:discard}}
```

這個形狀和標準庫一樣：`bufio.NewWriter` 收 `io.Writer`、回傳 `*bufio.Writer`，所以呼叫端拿得到 `Flush`。完整程式碼與測試在 `examples/chapter3/02_accept_interfaces_return_structs/after/`，測試另外涵蓋：`Flush` 之前寫入不會送出、寫入失敗時 `Flush` 回傳錯誤。

## 取捨

回傳具體型別有代價：

- 匯出的型別就是 API。欄位與方法一旦匯出就要維持；`Logger` 把欄位 `w` 留成不匯出，日後換掉緩衝的做法，呼叫端不受影響。第 1 章〈[什麼時候讓零值可用，什麼時候該強制用建構函式](../chapter1/06_zero_value_or_constructor.md)〉也是這個做法：匯出型別、不匯出欄位。
- 呼叫端想在測試裡換掉 `*audit.Logger`，得在自己的套件宣告一個小介面，只放用到的方法，`*audit.Logger` 自動符合（本章〈在使用端定義介面〉談過；測試替身怎麼選，第 5 章會談）。回傳 `*Logger` 之後，原本那個只有一種實作的 `Logger` 介面也用不到了，介面何時是多餘的，本章〈介面何時是多餘的〉會談。
- 參數收介面，也收得進有型別的 nil：呼叫端傳 `(*os.File)(nil)`，`New` 裡的 `w == nil` 判斷不出來，所以文件寫明 `w` 不可為 nil。

回傳介面在這些情況下才是對的：

- `error`：任何函式都可能失敗，原因與型別各不相同，用 `error` 這個介面才能走同一條路（前作〈[`error` 是介面](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter5/01_error_interface.html)〉）。回傳自訂錯誤型別時要小心上面的陷阱：沒有錯誤就直接 `return nil`，不要回傳一個值是 nil 的 `*MyError` 變數。
- 實作要到執行時才決定：`net.Dial` 依 `network` 參數連到 TCP、UDP 或 Unix socket，簽名只能是 `net.Conn`；`image.Decode` 依資料的格式選解碼器，回傳 `image.Image`。呼叫端事先不知道是哪一種，也不該依賴。

所以 before 的問題不在「回傳介面」本身，而在 `audit` 只有一種實作，卻把呼叫端需要的 `Flush` 擋在介面後面。判斷方法：呼叫端會不會需要具體型別上的方法？會，就回傳具體型別；實作是呼叫端不該、也無法預先知道的，才回傳介面。

## 重點整理

- 建構函式回傳具體型別。回傳介面會藏起背後型別的方法，要讓呼叫端用到就得加進介面，而介面加方法會讓每個實作者編譯失敗。
- 把具體型別的變數轉成介面回傳，nil 指標會變成「不等於 nil 的介面」；回傳具體型別，就沒有這個轉換。
- 參數接受小介面（像 `io.Writer`）：呼叫端能傳檔案、緩衝區、`io.Discard` 或測試用的 writer，也不必發明 nil 的約定。
- 回傳介面只在 `error`、或實作要到執行時才決定（像 `net.Dial`）時才合理；判斷依據是呼叫端需不需要具體型別上的方法。
