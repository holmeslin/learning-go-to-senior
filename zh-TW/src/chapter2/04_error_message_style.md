# 錯誤訊息怎麼寫

## 本集目標

讓多層包裝起來的錯誤鏈，讀起來像一句話：小寫開頭、不加句號、每一層只說自己在做什麼、不寫「failed to」。也能分清楚哪些話該給終端使用者看，哪些是給開發者的錯誤鏈。

## 情境

伺服器啟動時要讀設定檔：`server.New` 呼叫 `config.Load`，`config.Load` 呼叫標準庫的 `fs.ReadFile`。每一層出錯時都用 `fmt.Errorf` 加一段上下文再往上回傳。凌晨三點設定檔不見了，值班的人只看得到日誌裡的這一行。

`fmt.Errorf` 與 `%w` 怎麼包裝，前作〈[`fmt.Errorf` 與 `%w`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter5/04_errorf_wrap.html)〉講過；錯誤字串小寫、不加句號的慣例，前作〈[`errors.New`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter5/02_errors_new.html)〉也提過。這集談的是：每一層該寫什麼。

## 常見寫法

每一層都寫成一句完整的英文句子，用「Failed to」開頭，有時再補一個「Error:」：

```go,ignore
{{#include ../../../examples/chapter2/04_error_message_style/before/config/config.go:load}}
```

```go,ignore
{{#include ../../../examples/chapter2/04_error_message_style/before/server/server.go:new}}
```

完整程式碼在 `examples/chapter2/04_error_message_style/before/`。

## 問題在哪

下面這段 `Example` 把設定檔不存在、埠號寫錯兩種情況的完整錯誤鏈印出來，輸出就是 `// Output:` 註解，由 `go test` 逐字比對：

```go,ignore
{{#include ../../../examples/chapter2/04_error_message_style/before/server/example_test.go:example}}
```

這兩條錯誤鏈有幾個問題：

- **大寫與標點散在句子中間。** 錯誤會被串在別的訊息後面，「Failed to create server: Failed to load config: Error: …」的大寫與冒號斷不出句子，讀起來是一串標題。
- **「Failed to」與「Error:」沒有提供資訊。** 能走到這條錯誤鏈，操作就一定失敗了。三層各寫一次，只是把真正有用的字擠到最後面。
- **同一件事說兩次。** `fs.ReadFile` 回傳的錯誤已經是 `open app.conf: file does not exist`，包含動作與路徑；`Load` 又補了一句 `failed to open file app.conf`。
- **沒有說「這一層在做什麼」。** `Load` 的兩種失敗只靠後面的字區分，「Failed to load config」在兩個地方各寫一次。

## 較好的做法

每一層只用一個動詞片語說自己正在做的事，接冒號，再接底下那一層的錯誤：

```go,ignore
{{#include ../../../examples/chapter2/04_error_message_style/after/config/config.go:load}}
```

```go,ignore
{{#include ../../../examples/chapter2/04_error_message_style/after/server/server.go:new}}
```

規則只有四條：

1. 小寫開頭，結尾不加標點。
2. 每一層寫「我在做什麼」（`load config`、`create server`），不重述底下已經說過的事。這裡 `fs.ReadFile` 的錯誤已經帶著動作和路徑，`Load` 就不再寫檔名。
3. 不寫「failed to」「error:」。回傳的是 `error`，已經說明失敗了。
4. 用冒號串接，由外往內讀，像一句話：「建立伺服器：載入設定：開啟 app.conf：檔案不存在」。

同樣的兩種情況，印出來是這樣（一樣由 `Example` 的 `// Output:` 驗證）：

```go,ignore
{{#include ../../../examples/chapter2/04_error_message_style/after/server/example_test.go:example}}
```

after 的測試另外確認，錯誤鏈沒有因為改寫而斷掉：`errors.Is(err, fs.ErrNotExist)` 仍然是 `true`。完整程式碼在 `examples/chapter2/04_error_message_style/after/`。

## 取捨

**錯誤鏈是給開發者的，不是給終端使用者的。** `create server: load config: parse port: strconv.Atoi: parsing "abc": invalid syntax` 對維運的人有用，對一般使用者沒有。最外層（`main`、HTTP handler）才決定要對使用者說什麼，錯誤鏈整串留給日誌：

```go,ignore
{{#include ../../../examples/chapter2/04_error_message_style/after/server/example_test.go:user}}
```

在哪一層做這件事，本章〈在哪一層處理、哪一層只回傳〉會談。給使用者的訊息不受「英文、小寫」的慣例限制，可以用中文，也可以用完整句子和標點，因為它是最後一句，不是別人的前綴。

before 的寫法也不是處處都錯：

- 如果整個團隊的程式碼都寫「failed to」，一致比你個人的偏好重要。要改，整個 codebase 一起改。
- 「不重述底下」有前提：底下的錯誤真的帶著那段資訊。如果下層只回 `errors.New("not found")`，這一層就該補上是什麼找不到。
- 錯誤字串是寫給人看的，不要拿它當條件判斷（`strings.Contains(err.Error(), ...)`）。呼叫端要判斷，用哨兵錯誤或型別；錯誤字串改了之後會讓誰壞掉，本章〈錯誤是 API 契約的一部分〉會談。

## 重點整理

- 錯誤字串小寫開頭、不加句號；錯誤會被串在別的訊息後面，所以不是一句獨立的話。
- 每一層只寫自己在做什麼（動詞片語），底下已經說過的路徑、動作不再重寫。
- 不寫「failed to」「error:」，回傳 `error` 本身就代表失敗。
- 把完整的錯誤鏈印出來看一次：串起來像不像一句話？可以用 `Example` 鎖住，訊息就不會悄悄退化。
- 錯誤鏈給開發者看，最外層再翻成使用者看得懂的話。
