# 包裝錯誤：`%w` 該不該暴露給呼叫端

## 本集目標

用 `%w` 包住底層的錯誤時，判斷這個底層錯誤要不要成為自己 API 的一部分；不要的話，在邊界翻譯成自己的錯誤，或用 `%v` 切斷錯誤鏈。

## 情境

你維護一個 `profile` 套件，負責讀出使用者的個人設定。設定檔現在放在檔案系統上，用 `io/fs` 的 `fs.FS` 讀取。找不到設定檔是常態：新使用者本來就沒有，呼叫端要能分辨這種情況，改用預設值。

`%w` 怎麼包裝、`errors.Is` 與 `errors.As` 怎麼往錯誤鏈裡找，前作〈[`fmt.Errorf` 與 `%w`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter5/04_errorf_wrap.html)〉、〈[`errors.Is`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter5/06_errors_is.html)〉與〈[`errors.As` 與 `errors.AsType`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter5/08_errors_as.html)〉已經講過。這集只談要不要讓底層錯誤透出去。

## 常見寫法

讀檔失敗就加上一點上下文，用 `%w` 包起來往上交。這是前作教的標準做法，看起來也沒有哪裡不對：

```go,ignore
{{#include ../../../examples/chapter2/02_wrap_or_expose/before/profile.go:load}}
```

完整程式碼在 `examples/chapter2/02_wrap_or_expose/before/`。

## 問題在哪

`%w` 的效果是：被包住的錯誤，呼叫端用 `errors.Is` 與 `errors.As` 都找得到。呼叫端不需要你同意，自然就會開始用。底下是一個呼叫端會寫的程式，找不到設定檔時用預設值：

```go,ignore
{{#include ../../../examples/chapter2/02_wrap_or_expose/before/example_test.go:caller}}
```

`// Output:` 由 `go test` 逐字驗證。`Load` 沒有宣告過任何錯誤，呼叫端卻已經能做兩件事：

- 用 `errors.Is(err, fs.ErrNotExist)` 判斷「找不到」。
- 用 `errors.As` 取出 `*fs.PathError`，讀到 `Op` 與 `Path`，也就是檔名 `alice.json`。

這兩件事你都沒有答應過，但現在它們是 `Load` 實際的行為。呼叫端一旦寫成 `errors.Is(err, fs.ErrNotExist)`，`profile` 就必須永遠從 `fs.FS` 讀、而且永遠回傳這個錯誤，否則他們的判斷會失效。底層錯誤成了你的 API，而且沒有人在文件裡寫下它。改了之後呼叫端會怎麼壞，本章〈錯誤是 API 契約的一部分〉會談。

## 較好的做法

在邊界把底層錯誤翻譯成自己的語言。呼叫端需要的判斷是「找不到」，那就公開一個 `ErrNotFound`，只有 `fs.ErrNotExist` 才翻譯成它；其他錯誤沒有人需要分辨，用 `%v` 保留訊息就好：

```go,ignore
{{#include ../../../examples/chapter2/02_wrap_or_expose/after/profile.go:load}}
```

兩種錯誤的差別：

- 找不到：用 `%w` 包住**自己的** `ErrNotFound`，呼叫端的 `errors.Is(err, profile.ErrNotFound)` 成立。
- 其他錯誤（例如權限不足）：`%v` 只把文字放進訊息，不放進錯誤鏈。日誌裡看得到原因，呼叫端卻無法依賴它。

同樣的呼叫，現在長這樣：

```go,ignore
{{#include ../../../examples/chapter2/02_wrap_or_expose/after/example_test.go:example}}
```

測試放在外部套件，只用公開 API，驗證三件事：呼叫端要的判斷仍然成立、底層錯誤不再透出、其他錯誤保留訊息但不進錯誤鏈：

```go,ignore
{{#include ../../../examples/chapter2/02_wrap_or_expose/after/profile_test.go:test}}
```

另外兩個測試（權限錯誤與成功讀取）在 `examples/chapter2/02_wrap_or_expose/after/profile_test.go`。

## 取捨

翻譯不是免費的：

- 每個要讓呼叫端分辨的情況，都要多一個你自己的錯誤，並自己維護對應。上面的 `Load` 把權限不足也收進同一種「其他錯誤」，呼叫端想分辨它就辦不到，要等有人真的需要，再加一個 `ErrPermission`。
- `%v` 切斷之後，`errors.Is(err, context.Canceled)` 這類呼叫端通常真的需要的判斷也一併消失。底層錯誤如果值得讓呼叫端依賴，就刻意用 `%w`，並在文件註解寫明（文件與測試怎麼鎖住這個承諾，本章〈錯誤是 API 契約的一部分〉會談）。重點是「刻意」，而不是順手一包。
- 本章〈哨兵錯誤、自訂型別、不透明錯誤的選擇〉會談 `ErrNotFound` 該是哨兵、型別還是不透明錯誤；這集不處理。

`%w` 直接透出在這些情況下是對的：

- 同一個 module 內部、套件之間互相呼叫。呼叫端和被呼叫端由同一批人維護，改底層時可以一次改完所有判斷，逐層翻譯只是多寫程式（第 1 章〈[匯出名稱與 API 穩定性](../chapter1/04_exported_api_stability.md)〉談過：要守相容的，是你看不到、改不動的呼叫端）。
- 你的函式本來就是薄薄一層，目的就是把底層的行為原樣交給呼叫端，並且在文件裡這樣寫。
- 還沒有外部呼叫端，或者呼叫端全在同一個 repo。

判斷的方法：想一想，如果明天把底層換掉，呼叫端的判斷會不會跟著壞。會的話，這個錯誤就不該透出去。

## 重點整理

- `%w` 讓被包住的錯誤可以被 `errors.Is` 與 `errors.As` 找到，也就是把它變成你 API 的一部分，不論你有沒有打算這麼做。
- 跨出自己的邊界時，把呼叫端需要判斷的情況翻譯成自己的錯誤，其餘用 `%v` 保留訊息、切斷錯誤鏈。
- 翻譯後要用外部測試驗證兩件事：自己的錯誤成立，底層的錯誤不再成立。
- 底層錯誤如果值得讓呼叫端依賴，就刻意用 `%w` 並寫進文件。
- 同一個 module 內部，用 `%w` 透出通常沒有問題。
