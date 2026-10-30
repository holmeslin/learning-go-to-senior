# 秘密管理

## 本集目標

判斷秘密該放哪、怎麼讀、怎麼不被意外印出來：從環境變數或掛載的檔案讀，讀不到就在啟動時失敗；用型別讓 `fmt`、`slog` 與 JSON 輸出自動遮蔽。

## 情境

一個報表服務要連資料庫（帳號與密碼），還要呼叫第三方報表 API（一把金鑰）。啟動時載入設定，連不上資料庫時印出錯誤；另外有一個診斷用的端點，把目前的設定以 JSON 回傳，方便值班的人確認服務讀到什麼。

這集只談秘密的存放、讀取與不外洩。日誌的遮蔽策略（依鍵名遮蔽、不記還是遮蔽），本章〈日誌裡的敏感資料〉會談；設定怎麼載入，第 12 章〈設定與環境變數〉會談。`fmt.Stringer` 前作〈[`fmt.Stringer`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter4/08_stringer.html)〉講過，`encoding/json` 與 `log/slog` 見〈[`encoding/json`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter12/08_json.html)〉、〈[`log/slog`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter12/10_slog.html)〉；型別實作 `LogValuer`，第 9 章〈[結構化日誌：`log/slog` 的欄位與 handler 怎麼選](../chapter9/01_structured_logging.md)〉談過。

## 常見寫法

金鑰寫成常數，密碼也直接放在設定裡；設定 struct 的欄位全是 `string`，連線失敗時把連線字串寫進錯誤訊息，方便對照到底連去哪：

```go,ignore
{{#include ../../../examples/chapter11/05_secret_management/before/settings.go:config}}
```

```go,ignore
{{#include ../../../examples/chapter11/05_secret_management/before/settings.go:connect}}
```

這是刻意示範的漏洞，完整程式碼在 `examples/chapter11/05_secret_management/before/`，不要照用。

## 問題在哪

下面的 `Example` 把設定印出來、讓連線失敗、再輸出成診斷端點會回傳的 JSON。`// Output:` 由 `go test` 逐字比對：

```go,ignore
{{#include ../../../examples/chapter11/05_secret_management/before/example_test.go:example}}
```

三條路徑都把秘密原樣帶出去：一行 `%v` 或 `%+v`、一條錯誤訊息、一個 `json.Marshal`，而這三種輸出最後都會進日誌、錯誤追蹤系統或聊天室的貼文。再加上常數本身：

- **進了版本控制就收不回來。** 之後再刪掉，歷史裡還在；程式編譯進執行檔，`strings` 一類的工具也讀得到。處理方式是當作已經外洩、換掉秘密（輪替），不是改一行程式。
- **測試裡用看起來像真的金鑰，一樣有風險。** 掃描工具可能把它當成真的秘密，擋下 push 或發出警報；複製貼上的人也不會想到要換掉。本書的範例都用明顯是假的值，例如 `example-not-a-real-key`。

## 較好的做法

**型別讓輸出自動遮蔽。** 秘密包成 `Secret`，實作四個方法，分別管 `fmt` 的 `%v`／`%s`、`%#v`、`slog`、JSON。欄位是小寫，套件外只能用 `Reveal` 取值，誤把它當 `string` 會編譯失敗：

```go,ignore
{{#include ../../../examples/chapter11/05_secret_management/after/secret.go:secret}}
```

`LogValue` 在 `slog` 把值交給 handler 之前就換成遮蔽後的字串，不用靠 handler 怎麼格式化 `Any`。連線字串裡有密碼，所以 `DSN()` 回傳的也是 `Secret`；連線失敗的錯誤只寫位址，只有真的連線的那一行才呼叫 `Reveal`：

```go,ignore
{{#include ../../../examples/chapter11/05_secret_management/after/settings.go:connect}}
```

**從環境變數或檔案讀，讀不到就啟動失敗。** 檔案的做法是容器平台把秘密掛成檔案，路徑放在 `_FILE` 結尾的變數裡；檔案不會被子行程繼承，也不會出現在 `docker inspect`。兩個秘密一起檢查，缺的全部回報：

```go,ignore
{{#include ../../../examples/chapter11/05_secret_management/after/settings.go:load}}
```

這是第 2 章〈[多個錯誤要合併回傳，還是只回第一個](../chapter2/05_join_or_first_error.md)〉的 `errors.Join`：值班的人一次看到缺哪幾個。同樣的三條路徑，這次的輸出，以及讀不到秘密時的錯誤：

```go,ignore
{{#include ../../../examples/chapter11/05_secret_management/after/example_test.go:example}}
```

```go,ignore
{{#include ../../../examples/chapter11/05_secret_management/after/example_test.go:missing}}
```

測試用 `t.Setenv` 設好假的秘密，對 `%v`、`%+v`、`%#v`、`%s`、`%q` 與各種容器（struct、指標、slice、map）斷言輸出不含秘密，而且含有遮蔽標記，避免輸出是空的也算通過：

```go,ignore
{{#include ../../../examples/chapter11/05_secret_management/after/settings_test.go:load}}
```

```go,ignore
{{#include ../../../examples/chapter11/05_secret_management/after/settings_test.go:fmt}}
```

讀不到秘密、錯誤訊息只寫變數名稱、連線失敗不帶密碼，也各有測試，`slog` 的 text 與 JSON handler、`json.Marshal` 同樣如此：

```go,ignore
{{#include ../../../examples/chapter11/05_secret_management/after/settings_test.go:failfast}}
```

```go,ignore
{{#include ../../../examples/chapter11/05_secret_management/after/settings_test.go:connect}}
```

測試把每個方法分別改成回傳原值，都會失敗。`Reveal` 是正式程式交給 client 時也要用的 API，沒有為了測試才匯出的東西。

## 取捨

- **遮蔽型別擋的是意外，不是刻意。** `Reveal()` 一呼叫就拿得到值，反射或記憶體傾印（core dump）也拿得到。它的價值是把「取值」縮成少數幾個可搜尋的呼叫處。
- **型別沒顧到的角落。** 錯的動詞（`%d`）會印出原值，`go vet` 抓得到寫死的 `%d`；欄位若是小寫，`fmt` 看不到它的方法，會把裡面的字串印出來，所以含秘密的欄位要大寫。這兩種情況下面用測試鎖住。
- **環境變數也會外洩。** 子行程預設繼承全部環境變數；Linux 上同一個使用者與 root 讀得到 `/proc/<pid>/environ`；容器的環境變數 `docker inspect` 看得到。診斷端點輸出設定時型別擋得住，但第 9 章〈[`/debug/pprof` 與診斷端點](../chapter9/06_debug_pprof.md)〉的 `cmdline` 會公開命令列參數，所以秘密不放在參數上。檔案比環境變數好一些，仍然要設定檔案權限。
- **下層的錯誤可能自己帶著連線字串。** 包裝時只能管自己加的那一段；套件的錯誤帶了，只能在邊界換成不帶細節的訊息，原因留在受控的地方。
- **輪替比存放更難。** 上面的 `Load` 只在啟動時讀一次，換秘密要重新啟動；要不重啟就換，得重新讀檔案，也要讓新舊兩把在一段時間內都有效。HashiCorp Vault、AWS Secrets Manager、Google Secret Manager 這類秘密管理服務處理發放、輪替與稽核，這集只提名稱。
- **before 在什麼時候是對的？** 沒有任何真實秘密的示範程式與一次性的腳本；以及本來就不是秘密的值（公開的 client ID）。包成 `Secret` 要多一個型別，也要在交給第三方套件的地方呼叫 `Reveal`。

這個測試直接斷言 `%d` 與小寫欄位會露出原值：

```go,ignore
{{#include ../../../examples/chapter11/05_secret_management/after/settings_test.go:gaps}}
```

## 重點整理

- 秘密不進原始碼與版本控制；進去過就視為外洩，換掉。測試與範例用明顯是假的值。
- 從環境變數或掛載的檔案讀，讀不到、是空的就啟動失敗，錯誤只寫變數名稱。
- 秘密包成型別，實作 `String`、`GoString`、`LogValue`、`MarshalJSON`；由連線字串這類秘密衍生的值也要包。
- 錯誤訊息只寫位址與動作，不寫連線字串；測試斷言輸出不含秘密，並把每個方法改壞確認測試會失敗。
- 型別擋意外，擋不住 `Reveal`、錯的動詞與小寫欄位；環境變數、命令列參數與輪替要在其他層處理。
