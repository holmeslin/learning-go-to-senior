# 多個錯誤要合併回傳，還是只回第一個

## 本集目標

遇到好幾個彼此獨立的問題時，判斷該用 `errors.Join` 全部收集起來，還是回傳第一個就停；收集之後，呼叫端要怎麼檢查、怎麼顯示。

## 情境

你在寫註冊表單的驗證：姓名、email、年齡三個欄位，送進來的是使用者剛打好的字串。使用者會看到驗證的結果，並根據它修改欄位。

`errors.Join` 怎麼把錯誤合併、`errors.Is` 與 `errors.AsType` 怎麼在裡面找，前作〈[`errors.Join`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter5/09_errors_join.html)〉與〈[`errors.As` 與 `errors.AsType`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter5/08_errors_as.html)〉已經講過。這集談的是什麼時候該用。

## 常見寫法

`if err != nil { return err }` 寫慣了，驗證也照這個節奏：每個檢查失敗就回傳。

```go,ignore
{{#include ../../../examples/chapter2/05_join_or_first_error/before/form.go:validate}}
```

完整程式碼在 `examples/chapter2/05_join_or_first_error/before/`。

## 問題在哪

姓名、email、年齡三個欄位全都有問題的表單，使用者要送三次才能看完所有錯誤：

```go,ignore
{{#include ../../../examples/chapter2/05_join_or_first_error/before/example_test.go:example}}
```

三次呼叫的結果就是 `// Output:` 的三行：每次只看到一個，改完才看到下一個。這些檢查彼此獨立，姓名有沒有填，跟 email 有沒有 `@` 無關，驗證函式明明一次就能全部檢查完，卻只講了第一個。

## 較好的做法

每個欄位的檢查各自獨立，錯誤收集起來，最後用 `errors.Join` 合併。先定義一個說明「哪個欄位出了什麼問題」的型別：

```go,ignore
{{#include ../../../examples/chapter2/05_join_or_first_error/after/form.go:fielderror}}
```

再讓 `Validate` 收集每個欄位的錯誤：

```go,ignore
{{#include ../../../examples/chapter2/05_join_or_first_error/after/form.go:validate}}
```

`errors.Join(errs...)` 在 `errs` 是空的時候回傳 `nil`，呼叫端照樣用 `if err != nil` 判斷。全部有問題的表單，一次就看到所有問題：

```go,ignore
{{#include ../../../examples/chapter2/05_join_or_first_error/after/example_test.go:example}}
```

### 呼叫端怎麼檢查與顯示

合併之後，呼叫端能做三件事：

```go,ignore
{{#include ../../../examples/chapter2/05_join_or_first_error/after/example_test.go:inspect}}
```

- `errors.Is` 會找遍整棵錯誤樹，任何一個欄位的錯誤符合就是 `true`。
- `errors.AsType` 只回傳**第一個**符合型別的錯誤。想知道每個欄位，它不夠用。
- 要逐一處理，就用 `Unwrap() []error` 展開。`errors.Join` 的結果有這個方法，標準庫沒有為它另外提供函式，所以直接做型別斷言。

直接 `fmt.Println(err)` 則是每個錯誤一行，給人看、給日誌看都行。完整程式碼在 `examples/chapter2/05_join_or_first_error/after/`，測試鎖住了上面幾點：每個欄位都被回報、`errors.Is` 仍然找得到 `ErrRequired`，也找得到年齡解析失敗時的 `ErrNotNumber`。

### 有依賴的檢查，還是第一個就停

不是所有檢查都能一起收集。年齡的範圍檢查要用到解析出來的數字：

```go,ignore
{{#include ../../../examples/chapter2/05_join_or_first_error/after/form.go:checkage}}
```

`"abc"` 解析失敗時，`n` 沒有意義，再比較範圍只會多出一個誤導的錯誤，所以 `checkAge` 在這裡直接回傳。回傳的是這個套件自己的 `ErrNotNumber`，不是 `strconv` 的錯誤：`strconv.ErrSyntax` 不該成為表單驗證的 API（理由見本章〈[包裝錯誤：`%w` 該不該暴露給呼叫端](02_wrap_or_expose.md)〉），測試也確認 `errors.Is(err, strconv.ErrSyntax)` 是 `false`。整份表單是欄位之間收集，但同一個欄位裡，前一步是後一步的前提時，第一個錯誤就停。測試 `TestValidateAgeStopsAtFirstError` 鎖住這一點。

## 取捨

合併回傳不是免費的：

- 呼叫端要多做一點事。想逐一處理就得自己展開 `Unwrap() []error`，`errors.AsType` 又只會給第一個，用起來比單一錯誤麻煩。
- 錯誤訊息是多行的。同一行一筆的日誌格式裡，這會很難讀，也很難用文字比對。
- 收集本身也要成本。每個檢查都要查資料庫或呼叫遠端時，「全部跑完再回報」就是全部都付出代價。要驗證的是一大批資料時，也許該設個上限，收集到一定數量就停。

回傳第一個錯誤在這些情況下是對的：

- 後面的步驟依賴前面的結果，像上面的 `checkAge`，或「開檔、讀取、解析」這種流水線。前面失敗之後，後面的錯誤只是連鎖反應，收集起來是雜訊。
- 呼叫端不會同時處理好幾個問題。例如要不要重試、要不要中止，一個錯誤就足以決定。
- 檢查的成本高，或一個失敗就讓後面都沒有意義。

判斷的方法：這些問題彼此獨立，而且使用者（或呼叫端）能一次修好嗎？能，就收集；不能，就在第一個錯誤停下來。

## 重點整理

- 只回傳第一個錯誤，會讓使用者改一個、送出、再看下一個；檢查彼此獨立時，用 `errors.Join` 一次收集。
- 合併後的錯誤仍可用 `errors.Is` 找到任何一個；`errors.AsType` 只給第一個符合的；要逐一處理，用 `Unwrap() []error` 展開。
- 用有欄位資訊的錯誤型別（如 `FieldError`），呼叫端才知道每個問題屬於哪裡。
- 後面的檢查依賴前面的結果時，第一個錯誤就該停；一份輸入可以是「欄位之間收集、欄位內部 fail-fast」。
- 收集會讓錯誤訊息變成多行，檢查成本高或數量沒有上限時，要想清楚值不值得。
