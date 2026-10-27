# 在哪一層處理、哪一層只回傳

## 本集目標

遇到錯誤要往上傳時，判斷哪一層該「處理」它（記錄、回應、重試），哪一層只該加上自己的上下文就回傳；讓一個錯誤只被處理一次。

## 情境

你有一個「標記任務完成」的 HTTP 端點，分成 store、service、handler 三層，和第 1 章〈[套件切分的依據（依職責，不依型別）](../chapter1/01_package_by_responsibility.md)〉的 handler 是同一個形狀。資料庫連不上時，store 的 `Save` 會失敗。你希望出事的時候，日誌看得出發生了什麼事。

`if err != nil` 與 `%w` 包裝，前作〈[`if err != nil` 慣例](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter5/03_if_err_nil.html)〉與〈[`fmt.Errorf` 與 `%w`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter5/04_errorf_wrap.html)〉已經講過，這集只談哪一層該做什麼。日誌怎麼設計（格式、層級、欄位），第 9 章會談，這裡只用標準庫的 `log` 印一行。

## 常見寫法

每一層看到錯誤就先記一筆，再回傳給上一層：

```go,ignore
{{#include ../../../examples/chapter2/03_handle_at_boundary/before/taskapi.go:store}}
```

```go,ignore
{{#include ../../../examples/chapter2/03_handle_at_boundary/before/taskapi.go:service}}
```

```go,ignore
{{#include ../../../examples/chapter2/03_handle_at_boundary/before/taskapi.go:handler}}
```

完整程式碼在 `examples/chapter2/03_handle_at_boundary/before/`。

## 問題在哪

資料庫連不上時，送一個請求進去。下面 `Example` 的 `// Output:` 就是日誌的實際輸出（日誌寫到標準輸出，由 `go test` 逐字比對）：

```go,ignore
{{#include ../../../examples/chapter2/03_handle_at_boundary/before/example_test.go:example}}
```

三行日誌講的是同一個失敗：每一層都記了一次，後面兩行還把前一行的內容整段重複。

- 只有一次失敗，日誌卻有三行。如果有兩個請求失敗，就是六行，看日誌的人得自己數、自己對，才知道是兩次，不是六次。
- 最底層的 `store` 記錄時，不知道這個請求是哪個端點，是誰呼叫的；只有 handler 那一行有 `POST /tasks/7/complete`。
- 記錄的那一層，不知道上面的人會怎麼處理。如果呼叫端接著重試，而且成功了，這行日誌就是一則沒有人需要處理的警告。

## 較好的做法

中間層只做一件事：加上自己知道的上下文，把錯誤回傳。記錄與回應，留給邊界一次做完。

```go,ignore
{{#include ../../../examples/chapter2/03_handle_at_boundary/after/taskapi.go:store}}
```

```go,ignore
{{#include ../../../examples/chapter2/03_handle_at_boundary/after/taskapi.go:service}}
```

邊界是錯誤再也不能往上傳的地方：HTTP handler、`main`、worker 的迴圈。這裡的 handler 決定記錄，並回 500：

```go,ignore
{{#include ../../../examples/chapter2/03_handle_at_boundary/after/taskapi.go:handler}}
```

同一個請求，這次的 `// Output:`：

```go,ignore
{{#include ../../../examples/chapter2/03_handle_at_boundary/after/example_test.go:example}}
```

一個失敗，一行日誌。這一行還帶著每一層加上去的上下文：`complete: save task 7: connection refused`，從最外層讀到最內層，就是這個錯誤經過的路。`store` 和 `service` 甚至不需要 logger，沒有人會想在那裡「順手」記一筆。

完整程式碼在 `examples/chapter2/03_handle_at_boundary/after/`。

## 取捨

「只在邊界處理」的前提是錯誤真的能一路傳到邊界。有時候不行：跑在自己 goroutine 裡的 worker，沒有呼叫端接得到它的回傳值。這時 worker 的迴圈本身就是邊界，就地記錄是對的：

```go,ignore
{{#include ../../../examples/chapter2/03_handle_at_boundary/after/taskapi.go:runworker}}
```

```go,ignore
{{#include ../../../examples/chapter2/03_handle_at_boundary/after/example_test.go:worker}}
```

worker 失敗兩次，就記兩行，每一行是一個任務、一個失敗。goroutine 之間怎麼傳遞錯誤，第 6 章會談。

另外兩點要留意：

- 錯誤一經處理就不該再往上傳：記錄之後又回傳，上一層看到錯誤又會記一次，就回到 before 的樣子。要嘛處理（不再回傳），要嘛回傳（交給別人處理）。
- 中間層偶爾也有理由處理錯誤，例如重試、改用備援、把「找不到」當成正常情況；這時它是真的決定了怎麼辦，而不只是記一行就走。

before 在這些情況下反而可以接受：程式很小，只有一兩層，多記幾行無傷大雅；或是臨時除錯，想看錯誤在哪一層冒出來。長期維護的服務，還是讓日誌一個失敗只留一行比較好找。

## 重點整理

- 處理錯誤（記錄、回應、重試、改走備援）與傳遞錯誤是兩件事；一個錯誤只被處理一次，其他層只加上下文再回傳。
- 邊界是錯誤無法再往上傳的地方：HTTP handler、`main`、worker 的迴圈。
- 每一層都記錄，一次失敗就會出現多行，而且最底層記錄時缺少請求資訊，也不知道呼叫端會不會重試。
- 錯誤沒辦法往上傳時（例如跑在自己 goroutine 裡的 worker），就地記錄是對的，那裡就是邊界。
- 記錄之後又回傳，等於把錯誤處理了兩次；二選一。
