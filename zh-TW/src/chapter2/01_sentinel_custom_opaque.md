# 哨兵錯誤、自訂型別、不透明錯誤的選擇

## 本集目標

為一個錯誤選擇種類時，先問「呼叫端需要根據它做什麼決定」：不需要分辨就用不透明錯誤，只需要知道「是不是這種情況」就用哨兵錯誤，需要拿到額外資料就用自訂型別。

## 情境

你在寫一個 `wallet` 套件，提供 `Withdraw(id, amount)` 提款。它會失敗的原因有三種：帳戶不存在、餘額不足、`Wallet` 已經關閉（或金額不合法）。呼叫端是一個 HTTP 服務：帳戶不存在要回 404，餘額不足要告訴使用者還差多少錢，其他的失敗一律回 500。

哨兵錯誤、自訂錯誤型別與 `errors.Is`、`errors.AsType` 的語法，前作〈[哨兵錯誤](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter5/05_sentinel_errors.html)〉、〈[自訂錯誤型別](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter5/07_custom_error_type.html)〉與〈[`errors.As` 與 `errors.AsType`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter5/08_errors_as.html)〉已經講過，這集只談怎麼選。

## 常見寫法

每一種失敗都定義一個哨兵錯誤，因為「這樣呼叫端想判斷哪一種都可以」：

```go,ignore
{{#include ../../../examples/chapter2/01_sentinel_custom_opaque/before/wallet.go:errors}}
```

```go,ignore
{{#include ../../../examples/chapter2/01_sentinel_custom_opaque/before/wallet.go:withdraw}}
```

呼叫端用 `errors.Is` 逐一判斷。餘額不足要告訴使用者差多少，哨兵錯誤帶不了資料，只好把數字寫進訊息，再由呼叫端從訊息裡解析出來。下面的 `Example` 把三種失敗各跑一次：

```go,ignore
{{#include ../../../examples/chapter2/01_sentinel_custom_opaque/before/example_test.go:example}}
```

完整程式碼在 `examples/chapter2/01_sentinel_custom_opaque/before/`。

## 問題在哪

看上面的 `decide`，有兩個問題。

第一，`ErrClosed` 和 `ErrInvalidAmount` 那一行，做的事和 `default` 完全一樣，都是 500。這個呼叫端從來不會因為「Wallet 已經關閉」而改變做法，但這兩個名稱已經是匯出的 API：套件要一直保留它們，之後每次改動都得想想有沒有人在用。

第二，`shortfall` 的格式字串和 `Withdraw` 裡組訊息的格式，是兩處各寫一份、靠約定對上的。資料明明是程式算出來的數字，卻被壓成文字，再讓呼叫端把它解析回來；解析失敗時，呼叫端只能退回一個沒有數字的訊息（`decide` 裡的 `422 餘額不足`）。

四個哨兵錯誤看起來很周到，但它們沒有回答「呼叫端要做什麼決定」：該分辨的（餘額差多少）分辨不了，不需要分辨的卻都公開了。

## 較好的做法

依呼叫端的決定，每一種失敗各選一種錯誤：

- **查無帳戶**：呼叫端只需要知道「是不是這種情況」，不需要任何資料 → 哨兵錯誤 `ErrNotFound`。
- **餘額不足**：呼叫端需要 `Need`、`Have` 兩個數字 → 自訂型別 `*InsufficientFundsError`，數字是欄位，不必解析文字。
- **Wallet 已關閉、金額不合法**：呼叫端沒有不同的做法 → 不透明錯誤：回傳 `errors.New` 或 `fmt.Errorf` 的結果，不匯出任何名稱。

```go,ignore
{{#include ../../../examples/chapter2/01_sentinel_custom_opaque/after/wallet.go:errors}}
```

```go,ignore
{{#include ../../../examples/chapter2/01_sentinel_custom_opaque/after/wallet.go:withdraw}}
```

文件註解把承諾寫清楚：哪兩種失敗可以用 `errors.Is`、`errors.AsType` 分辨，其他只保證「不是 nil」。呼叫端的 `decide` 少了一個分支，也不再解析文字；輸出和 before 一樣，由 `go test` 逐字比對：

```go,ignore
{{#include ../../../examples/chapter2/01_sentinel_custom_opaque/after/example_test.go:example}}
```

after 的測試放在外部測試套件，三種錯誤各用呼叫端實際會用的方式檢查（`errors.Is`、`errors.AsType`），不透明錯誤的測試則確認它不會被誤認成前兩種。完整程式碼在 `examples/chapter2/01_sentinel_custom_opaque/after/`。

## 取捨

不透明錯誤是起點，不是終點。一開始先回傳不透明錯誤，等呼叫端真的需要分辨時再升級成哨兵錯誤或自訂型別，通常比一開始就公開便宜：

- 公開一個名稱容易，之後想收回來，就得先問有誰在依賴它（本章〈錯誤是 API 契約的一部分〉會談）。
- 等需求真的出現，你才知道呼叫端要的是什麼：只要知道「是不是」，還是要資料。一開始猜，常常猜錯種類。

另外兩種也各有代價：

- 哨兵錯誤是全域的名稱，沒辦法帶資料；多定義一個，API 就多一個要維護的名稱。
- 自訂型別要多寫型別與 `Error` 方法，欄位也成為公開 API；而且呼叫端要寫 `errors.AsType` 的泛型型別參數，寫法比 `errors.Is` 重。

before 在這些情況下反而是對的：

- 呼叫端真的需要依 `ErrClosed` 做不同的事，例如重新建立 Wallet 再重試；這時它就該是公開的哨兵錯誤。
- 數字很少被用到，使用者只看一句訊息就夠：只回傳含數字的訊息，也比硬做一個型別簡單。

判斷的方法：寫出呼叫端的 `if` 或 `switch`。每個分支要做的事不同，才值得有對應的種類；做的事一樣的，就是同一種不透明錯誤。

## 重點整理

- 選錯誤種類的依據是「呼叫端需要根據它做什麼決定」，不是「這個失敗叫什麼名字」。
- 不需要分辨：不透明錯誤。只需要知道是不是這種情況：哨兵錯誤。需要拿到資料：自訂型別。
- 把資料壓進訊息再讓呼叫端解析，是該用自訂型別的訊號；每個失敗都匯出一個哨兵錯誤，是把沒人需要的名稱變成 API。
- 先用不透明錯誤，等呼叫端真的需要再升級；公開出去的名稱，之後很難收回。
- 在文件註解寫明哪些錯誤可以分辨，其餘只保證「不是 nil」。
