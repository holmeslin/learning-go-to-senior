# 錯誤是 API 契約的一部分

## 本集目標

把「函式會回傳哪些錯誤」當成 API 契約來維護：在文件註解寫明、用外部測試鎖住，並在改錯誤之前先判斷，這個改動會不會讓呼叫端的 `errors.Is` 默默失效。

## 情境

你維護一個內部共用的 `store` 套件，`Find(id)` 找不到使用者時回傳哨兵錯誤 `ErrNotFound`，三個服務都用 `errors.Is(err, store.ErrNotFound)` 把它對應成 404。現在要發 v1.1，想讓錯誤訊息好查一點：把 id 放進訊息裡。

`errors.Is` 與 `%w` 的機制，前作〈[`errors.Is`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter5/06_errors_is.html)〉與〈[`fmt.Errorf` 與 `%w`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter5/04_errorf_wrap.html)〉已經講過。本章〈[哨兵錯誤、自訂型別、不透明錯誤的選擇](01_sentinel_custom_opaque.md)〉談過該公開哪一種錯誤，本章〈[包裝錯誤：`%w` 該不該暴露給呼叫端](02_wrap_or_expose.md)〉談過 `%w` 會公開被包住的錯誤；這集談的是公開之後的事：它已經是契約，要像函式簽名一樣維護。

## 常見寫法

v1 找不到時直接回傳哨兵錯誤：

```go,ignore
{{#include ../../../examples/chapter2/07_error_contract/before/v1/store.go:find}}
```

v1.1 為了讓訊息帶上 id，改成用 `fmt.Errorf` 產生新的錯誤。`ErrNotFound` 還留在套件裡，沒有刪：

```go,ignore
{{#include ../../../examples/chapter2/07_error_contract/before/v11/store.go:find}}
```

完整程式碼在 `examples/chapter2/07_error_contract/before/`（`v1` 與 `v11` 是同一個套件的兩個版本，為了放在同一個範例裡才分成兩個目錄）。

## 問題在哪

v1.1 能編譯，`go vet` 沒有意見，`ErrNotFound` 也還在，呼叫端不用改任何一行。下面把同一行呼叫端的判斷分別對 v1 與 v1.1 跑一次（兩次只有 import 的套件不同）。`Example` 的 `// Output:` 就是執行結果，`go test` 會逐字驗證：

```go,ignore
{{#include ../../../examples/chapter2/07_error_contract/before/v11/example_test.go:example}}
```

同樣是找不到 id 42，`errors.Is` 從 `true` 變成 `false`。呼叫端用它判斷「找不到」的那個分支再也不會進去，原本該回 404 的請求會落到「其他錯誤」那條路。

這個壞法難發現，是因為沒有任何東西出聲：套件的簽名沒變，所以編譯器不會報錯；`ErrNotFound` 還是匯出的，所以呼叫端也沒有 `undefined`。對比第 1 章〈[匯出名稱與 API 穩定性](../chapter1/04_exported_api_stability.md)〉裡的改動，呼叫端一編譯就知道；改錯誤則只有在那條錯誤路徑真的被執行到的時候才會露出來，而錯誤路徑正是測試最常漏掉的地方。

## 較好的做法

做兩件事：把會回傳的錯誤寫進文件註解，再用測試把這份清單鎖住。

after 的 `Find` 一樣把 id 放進訊息，但用 `%w` 包住 `ErrNotFound`，並且在文件註解寫明契約（順便新增了一個 `*InvalidIDError`，id 不合法時回傳）：

```go,ignore
{{#include ../../../examples/chapter2/07_error_contract/after/store/store.go:find}}
```

文件註解只列呼叫端能依賴的東西：哪些錯誤、用什麼方式判斷（`errors.Is` 或 `errors.AsType`）。它沒有承諾訊息的文字，所以訊息之後要改就能改。

接著用外部測試套件，從呼叫端的角度驗證同一份清單。測試只用公開 API，檢查的是呼叫端真正會用的 `errors.Is` 與 `errors.AsType` 的結果：

```go,ignore
{{#include ../../../examples/chapter2/07_error_contract/after/store/store_test.go:test}}
```

這個測試鎖住的，就是上一節壞掉的那類改動。我們實際對 after 試了三種改動，每一種都重跑 `go test`；三種都編譯得過，測試都失敗（試完已還原）：

- 把 `%w` 改成 `%v`：`errors.Is(err, ErrNotFound)` 變 `false`。
- 把包住的 `ErrNotFound` 換成新建的 `errors.New("user not found")`：同樣變 `false`，訊息看起來完全一樣。
- 把 `*InvalidIDError` 改成值型別：`Find` 回傳 `InvalidIDError{ID: id}`，`Error` 的接收者也要一起改成值接收者，否則編譯不過：`errors.AsType[*InvalidIDError]` 變 `false`。

`Example` 則讓文件註解的承諾可以直接讀，`go test` 會比對輸出：

```go,ignore
{{#include ../../../examples/chapter2/07_error_contract/after/store/example_test.go:example}}
```

完整程式碼在 `examples/chapter2/07_error_contract/after/`。

## 取捨

還沒有外部使用者時，錯誤契約可以自由調整。套件只有你一個人在用、還沒發布，或所有呼叫端都在同一個 repo、改完能一起跑 `go test ./...`，那就直接改：搜尋 `ErrNotFound` 的所有用法，一起修掉，比先寫契約與測試划算。要守契約的，是你看不到、也改不動的呼叫端。

鎖得越多，以後越難改。文件註解每多列一種錯誤、測試每多鎖一個判斷，就多一份要長期維護的承諾，所以只列呼叫端確實需要區分的錯誤；其他情況用不透明的錯誤，保留改動的空間。

另一半責任在呼叫端：文件註解沒承諾訊息，呼叫端就不該用 `strings.Contains(err.Error(), "not found")` 判斷。before 的 v1.1 也不是全然錯：如果 `ErrNotFound` 本來就沒打算讓呼叫端判斷，改掉它是自由的，問題出在它已經被公開、被依賴，卻沒有人寫下「可以依賴」。

## 重點整理

- 函式回傳哪些錯誤，是對呼叫端的承諾；錯誤值、錯誤型別、有沒有用 `%w` 包住，都是契約的一部分。
- 改錯誤不會有編譯錯誤：哨兵換成新建的錯誤，`errors.Is` 會默默變成 `false`，要等錯誤路徑真的被執行才發現。
- 在文件註解寫明會回傳哪些錯誤、用什麼方式判斷，不承諾訊息文字。
- 用外部測試套件鎖住 `errors.Is` 與 `errors.AsType` 的結果；改錯誤的人在 `go test` 就會看到。
- 還沒有外部使用者時，錯誤契約可以自由調整；只有看不到、改不動的呼叫端才需要守。
