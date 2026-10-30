# 輸入驗證

## 本集目標

判斷 HTTP 請求的驗證該放在哪裡、做到多完整：在 handler 入口限制本文大小、拒絕未知欄位與多餘的內容，把原始輸入轉成驗證過的型別，之後的程式只接受這個型別，並一次回報所有欄位的錯誤。

## 情境

會議室預約服務有一支 `POST /bookings`，收 JSON：`name`、`email`、`room`、`guests`。網頁表單已經檢查過欄位，所以後端只檢查必填。但後端的網址誰都能直接送請求，用 `curl` 就能繞過網頁；前端的檢查是為了方便使用者，不是防線。

這集只談請求本文本身的驗證：大小、格式、範圍、未知欄位。輸入被拿去組 SQL、當檔案路徑、當對外請求的網址，各有各的攻擊方式，本章〈SQL injection 與參數化查詢〉、〈路徑來自使用者輸入時，用 `os.Root` 還是自己檢查〉、〈SSRF 與對外請求〉會談。`MaxBytesReader` 與 JSON 的基本用法前作〈[JSON API](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter14/04_json_api.html)〉教過；handler 的測試方式見第 5 章〈[要啟動真的 server，還是只測 handler](../chapter5/09_server_or_handler.md)〉。

## 常見寫法

直接解碼進 struct，檢查必填，然後交給下一層：

```go,ignore
{{#include ../../../examples/chapter11/01_input_validation/before/handler.go:handler}}
```

`before` 套件的文件註解寫明這是刻意示範的漏洞，不要照用。完整程式碼在 `examples/chapter11/01_input_validation/before/`。

## 問題在哪

下面八筆輸入，before 的 handler 全部接受（回 201，並把資料交給 `save`）：

| 輸入 | 內容 | 問題 |
| --- | --- | --- |
| `oversized_body` | 姓名是 2 MiB 的字串 | 整份讀進記憶體，同時送很多份就是拒絕服務 |
| `unknown_field` | 多了 `"admin":true` | 被默默丟掉；日後 struct 加了同名欄位，這筆輸入就突然生效 |
| `trailing_value` | 一個物件後面又接另一個 | `Decode` 只讀第一個值；代理、日誌與 handler 對這份本文的解讀可能不同 |
| `guests_out_of_range` | 100000 人 | 型別對，範圍不對 |
| `unknown_room` | 房間 `x9` | 不存在的房間 |
| `bad_email` | `not-an-email` | 只檢查了「非空」 |
| `blank_name` | 三個空白 | 非空不等於有內容 |
| `control_character` | 姓名含 `\r\n` | 之後進 email 標頭或日誌，會多出一行 |

表格就是這段程式。`Attacks` 回傳同一份輸入，before 的 `Example` 與 after 的測試都用它：

```go,ignore
{{#include ../../../examples/chapter11/01_input_validation/before/handler.go:attacks}}
```

```go,ignore
{{#include ../../../examples/chapter11/01_input_validation/before/example_test.go:example}}
```

被拒絕的請求也有問題。解碼失敗時 handler 把 `err.Error()` 原樣回給呼叫端：

```go,ignore
{{#include ../../../examples/chapter11/01_input_validation/before/example_test.go:leak}}
```

呼叫端因此知道伺服器裡有個叫 `Request` 的型別、欄位對應哪個名稱。單看沒什麼，但這是攻擊者拼湊內部樣貌的材料。錯誤鏈給開發者、最外層才決定對使用者說什麼，第 2 章〈[錯誤訊息怎麼寫](../chapter2/04_error_message_style.md)〉談過。

## 較好的做法

驗證做在信任邊界，分三層：本文的形狀、欄位的值、通過之後的型別。

第一層在 `decode`：`http.MaxBytesReader` 限制大小，`DisallowUnknownFields` 拒絕未知欄位，解碼完再讀一次，確認後面沒有別的值。錯誤用 `problem` 翻成固定的訊息，不回 `err.Error()`；`NewHandler` 把三層接起來：

```go,ignore
{{#include ../../../examples/chapter11/01_input_validation/after/handler.go:handler}}
```

第二、三層在 `Validate`。它收原始的 `Request`，回傳 `Booking`：欄位不匯出，只有 `Validate` 建得出它，所以 `save` 收到 `Booking` 就不必再檢查。順序是先正規化（去掉前後空白），再檢查正規化後的值，存的也是那一個值。`blank_name` 就是順序錯了的後果：檢查原始字串，三個空白通過了。

```go,ignore
{{#include ../../../examples/chapter11/01_input_validation/after/booking.go:types}}
```

```go,ignore
{{#include ../../../examples/chapter11/01_input_validation/after/booking.go:booking}}
```

每個欄位的檢查彼此獨立，所以全部收集起來，用 `errors.Join` 合併，一次回報；理由與呼叫端怎麼展開，第 2 章〈[多個錯誤要合併回傳，還是只回第一個](../chapter2/05_join_or_first_error.md)〉談過。本文層則是第一個問題就停：本文壞了，欄位沒有意義。

測試用同一份表格。before 當對照組（同一筆輸入被接受、到了 `save`），after 要回指定的狀態碼與欄位、`save` 不被呼叫、回應不含 `json:`、`struct` 之類的內部細節：

```go,ignore
{{#include ../../../examples/chapter11/01_input_validation/after/booking_test.go:helpers}}
```

```go,ignore
{{#include ../../../examples/chapter11/01_input_validation/after/booking_test.go:attacks}}
```

這個測試用的 `before.Attacks` 與 `MaxBodyBytes` 是為了測試才匯出的：前者讓兩個套件共用一份表格，後者讓測試確認超大的本文沒有被讀完。多個欄位同時有問題時，一次回報全部：

```go,ignore
{{#include ../../../examples/chapter11/01_input_validation/after/booking_test.go:all}}
```

```go,ignore
{{#include ../../../examples/chapter11/01_input_validation/after/booking_test.go:bigbody}}
```

把 `MaxBytesReader` 換成 `r.Body`、拿掉 `DisallowUnknownFields`、拿掉最後那次讀取、拿掉任何一個欄位的檢查、拿掉 `TrimSpace`，或讓 `problem` 回 `err.Error()`，上面的測試都會失敗。回應長這樣：

```go,ignore
{{#include ../../../examples/chapter11/01_input_validation/after/example_test.go:example}}
```

## 取捨

- 驗證程式碼要維護：每加一個欄位，就多一條規則、一個測試。規則寫在 `Validate` 一處，比散在各個 handler 好找；範圍（20 人、50 字）是業務決定，改的時候兩邊一起改。驗證函式庫（例如 `go-playground/validator`）用 struct tag 宣告規則，省下樣板，換來規則藏在字串裡、跨欄位的規則還是要自己寫；本集不用，需要時再評估。
- 白名單比黑名單可靠：`room` 只收已知的三個值，沒人需要猜「還有哪些壞字元」。姓名不行，人名什麼字都有，所以只拒絕控制字元（一個 Unicode 定義好的類別）與過長的字串。黑名單會漏，所以能寫成白名單的就寫白名單。
- 錯誤訊息的詳細程度是取捨：「must be between 1 and 20」讓合法的使用者改得動，代價是公開了規則；它不含 Go 的型別名稱、內部路徑與使用者送來的字串，所以可以公開。要不要連規則都不說，看這支 API 的對象是誰。
- 4 KiB 的上限是依這個請求估的（合法的本文不到 1 KiB），每支 API 要各自算；上傳檔案的端點需要另一個上限，不能共用。
- 只看 `Content-Length` 不夠：chunked 傳輸沒有這個標頭。`MaxBytesReader` 在讀的時候數，超過就停，並盡可能要求伺服器在回應後關閉連線。
- 重複的鍵擋不住：`encoding/json` 取最後一個值，這份驗證看不到被蓋掉的那個。需要拒絕時，前作〈[`encoding/json/v2`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter12/09_json_v2.html)〉的解碼預設就拒絕。
- 程式這一層擋不住的：大量的小請求。限制本文大小只是少吃一點記憶體；請求的頻率，第 10 章〈[`golang.org/x/time/rate` 限流](../chapter10/04_rate_limit.md)〉談過。

before 在什麼時候可以接受？請求只在信任的內部服務之間傳，雙方是同一個團隊、同一個型別定義時，嚴格的驗證換來的保護有限。只要請求會跨過信任邊界，就該用 after。

## 重點整理

- 前端的檢查只是方便，驗證要在信任邊界做完整：大小、未知欄位、本文裡只有一個值、值的範圍。
- `MaxBytesReader` 限制本文大小，上限依這個請求估，不共用。
- 把原始輸入轉成驗證過的型別，欄位不匯出，之後的程式不必再檢查；先正規化，再檢查，存檢查過的那個值。
- 能列舉的用白名單；錯誤一次回報所有欄位，訊息固定，不帶 `err.Error()` 與使用者的輸入。
- 驗證要有表格測試，並用 before 當對照，確認同一筆輸入真的被擋下。
