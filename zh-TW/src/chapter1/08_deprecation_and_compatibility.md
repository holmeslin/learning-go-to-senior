# 棄用（deprecation）與相容性

## 本集目標

想淘汰或改名一個已經有人在用的 API 時，判斷該直接刪，還是保留舊名稱、標上 `// Deprecated:` 並轉呼新實作，以及什麼時候才需要開新的主版本號。

## 情境

你維護一個內部共用的 `size` 套件，負責把 `"10MiB"` 這類字串解析成位元組數。v1.2.0 的函式叫 `size.ParseSize`，每次呼叫都得把 `size` 讀兩遍。你想在 v1.3.0 把它改成 `size.Parse`。公司裡有十幾個服務依賴這個套件，各自有自己的發版時間。

文件註解怎麼寫、版本號代表什麼，前作〈[文件註解與 `go doc`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/07_doc_comments.html)〉與〈[`go get` 與相依版本](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/15_go_get.html)〉已經講過。這集只談怎麼淘汰。

## 常見寫法

直覺的做法是直接改名：新名字比較好，舊的就別留著了。v1.3.0 的 `size` 只剩 `Parse`：

```go,ignore
{{#include ../../../examples/chapter1/08_deprecation_and_compatibility/before/size.go:parse}}
```

完整程式碼在 `examples/chapter1/08_deprecation_and_compatibility/before/`。

## 問題在哪

套件本身編譯、`go vet` 都沒問題，壞的是呼叫端。某個服務還寫著舊名稱：

```text
package main

import (
	"fmt"

	"example.com/app/size"
)

func main() {
	n, err := size.ParseSize("10MiB")
	fmt.Println(n, err)
}
```

它一升到 v1.3.0 就無法編譯：

編譯錯誤：

```text
# example.com/app
./main.go:10:17: undefined: size.ParseSize
```

錯誤出現的時機，是每個服務各自升級的那一天，十幾個服務就是十幾次。呼叫端事先沒有任何警告，升級時也只能照錯誤訊息改名。

## 較好的做法

保留舊名稱，內容改成轉呼新函式，並用 `// Deprecated:` 開頭的段落說明該改用什麼：

```go,ignore
{{#include ../../../examples/chapter1/08_deprecation_and_compatibility/after/size.go:deprecated}}
```

`Parse` 的實作和上面 before 的一樣。幾件事要注意：

- `Deprecated:` 要自成一個段落（和前面的說明之間隔一行 `//`），開頭固定這個詞，工具才認得。`go doc` 會顯示它（省略第一行的 import 路徑）：

  執行結果：

  ```text
  $ go doc ParseSize
  package size

  func ParseSize(s string) (int64, error)
      ParseSize 把大小字串解析成位元組數。

      Deprecated: 套件名稱已經是 size，名稱重複了；請改用 Parse。
  ```

  `staticcheck` 這類工具也會根據這個標記，在呼叫端標出警告。

- 舊名稱只轉呼，不複製實作，兩邊永遠是同一份邏輯。測試要鎖住這件事：`ParseSize` 與 `Parse` 對同樣的輸入，結果和錯誤都要一致。
- 呼叫端不用立刻改。他們升級後照常編譯，看到警告再挑自己方便的時間處理。
- 加上 `//go:fix inline`，呼叫端執行 `go fix` 就會自動把 `size.ParseSize(...)` 改成 `size.Parse(...)`（前作〈[`go fix`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/16_go_fix.html)〉介紹過這個指令）。這個指令只適用於轉呼的函式本體是單純一個呼叫的情況。

新的用法本身，一樣寫成外部測試套件裡的 `Example`：

```go,ignore
{{#include ../../../examples/chapter1/08_deprecation_and_compatibility/after/example_test.go:example}}
```

完整程式碼在 `examples/chapter1/08_deprecation_and_compatibility/after/`。

### 什麼時候才需要 `/v2`

棄用只能處理「多一個新名稱」的改動。新舊行為無法並存時，例如同名函式必須改簽名、必須改變既有輸入的結果，才需要把模組路徑改成 `.../size/v2`，讓呼叫端分別升級。這一集的改名不需要它；路徑與版本號怎麼對應，前作〈[`go get` 與相依版本](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/15_go_get.html)〉已經講過。

## 取捨

保留舊 API 不是免費的：

- 舊名稱要一直維護：文件、測試、API 面積都多一份，新進的人會問「這兩個差在哪」。
- 棄用只是提醒，不是強制。不理會的呼叫端會一直留在舊名稱上，所以你得決定什麼時候刪，而在 v1 裡通常等不到那一天：刪除是破壞相容的改動，要等到開 `/v2` 時才能一起做。
- 轉呼的舊函式如果行為有安全問題，不能只標棄用，要直接修掉，或提高主版本號。

直接改名或刪除在這些情況下是對的：

- 套件還沒有外部使用者，或使用者全在同一個 repo、能一次改完。這時多留一個名稱只是在增加負擔（本章〈[匯出名稱與 API 穩定性](04_exported_api_stability.md)〉談過怎麼判斷）。
- 套件還在 v0，你已經明講過 API 隨時會變。

判斷的方法：想像最慢升級的那個呼叫端。如果你沒辦法在同一天通知並協助他改完，就保留舊名稱。

## 重點整理

- 直接刪掉或改名舊 API，呼叫端會在升級的那一天編譯失敗，時機不由他們決定。
- 淘汰的做法是保留舊名稱、轉呼新實作，並用自成一段的 `// Deprecated:` 說明該改用什麼；`go doc` 與靜態分析工具都會顯示。
- 用測試鎖住「舊名稱與新名稱行為一致」；`//go:fix inline` 讓呼叫端用 `go fix` 自動遷移。
- 新舊行為無法並存時才開 `/v2`；單純改名用棄用就夠。
- 沒有外部使用者，或還在 v0 時，直接改名比較划算。
