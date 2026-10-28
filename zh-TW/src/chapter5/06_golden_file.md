# golden file

## 本集目標

輸出很長的函式，判斷預期輸出該寫在測試字串、只檢查幾個子字串，還是放進 `testdata` 的 golden file；知道 `-update` 怎麼用，以及它的風險。

## 情境

財務每個月要一份純文字的銷售報表：依分類排序，每個分類有小計，最後是總計，欄位靠空白對齊。`Render` 負責產生它，輸出十幾行，常常要改：加一欄、換小數位數、調欄寬。`go test` 與 `Example` 的寫法前作教過：〈[`go test`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/09_go_test.html)〉、〈[Example 測試](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/13_example_test.html)〉。這集只談：預期輸出放哪裡。

## 常見寫法

第一種，整份預期輸出寫死在測試的字串裡：

```go,ignore
{{#include ../../../examples/chapter5/06_golden_file/before/report_test.go:exact}}
```

第二種，嫌上面那份太長、一改格式就壞，只檢查幾個重要的字句：

```go,ignore
{{#include ../../../examples/chapter5/06_golden_file/before/report_test.go:parts}}
```

完整程式碼在 `examples/chapter5/06_golden_file/before/`。

## 問題在哪

第一種寫法抓得到格式變動，代價是維護：欄寬從 26 改成 28，字串裡每一行都要手動補空白；失敗時 `%q` 把整份輸出印成兩行長字串，找不出哪一行不一樣。於是有人改成第二種。

第二種的問題是：格式壞了也會過。下面的 `collapse` 把每行的連續空白縮成一個，模擬一次「欄位對齊被改壞」的修改（這是人為模擬，不是 `Render` 真的這樣輸出）。`Example` 印出壞掉的報表，以及子字串檢查缺少的字句：

```go,ignore
{{#include ../../../examples/chapter5/06_golden_file/before/example_test.go:example}}
```

欄位擠成一團，縮排與對齊都沒了，`TestRenderParts` 照過。子字串只能檢查「有沒有」，檢查不了「長什麼樣子」；而報表的價值有一半就在格式。

## 較好的做法

預期輸出放在 `testdata/basic.golden`（`go` 工具會忽略名為 `testdata` 的目錄），測試比對完整輸出：

```text
{{#include ../../../examples/chapter5/06_golden_file/after/testdata/basic.golden}}
```

比對與更新寫成一個小函式，測試只管產生輸出：

```go,ignore
{{#include ../../../examples/chapter5/06_golden_file/after/report_test.go:update}}
```

```go,ignore
{{#include ../../../examples/chapter5/06_golden_file/after/report_test.go:golden}}
```

```go,ignore
{{#include ../../../examples/chapter5/06_golden_file/after/report_test.go:test}}
```

這裡有三個細節。`Render` 的輸入沒有任何不固定的東西：分類在 `Render` 裡先排序，才不受 map 走訪順序影響；產生日期由呼叫端傳入，不在函式裡讀時鐘。測試的輸入每次由函式建立新的值，不共用變數。失敗訊息指出第一個不同的行，以及該怎麼辦。同一個「欄位對齊被改壞」，現在會被抓到（`intact` 是原本的輸出，空字串代表和 golden file 完全相同）：

```go,ignore
{{#include ../../../examples/chapter5/06_golden_file/after/example_test.go:example}}
```

在真的測試失敗時看起來像這樣。把 `Render` 的 `%3d` 改成 `%d` 再跑 `go test -run TestRenderGolden .`，輸出如下；省略了每個測試與套件後面的耗時：

執行結果：

```text
--- FAIL: TestRenderGolden
    --- FAIL: TestRenderGolden/basic
        report_test.go:105: output differs from testdata/basic.golden at line 5:
             got: "  Coffee beans 1kg           3 x    18.00 =     54.00"
            want: "  Coffee beans 1kg             3 x    18.00 =     54.00"
            run go test -update if the change is intended
    --- FAIL: TestRenderGolden/refund
        report_test.go:105: output differs from testdata/refund.golden at line 5:
             got: "  Coffee beans 1kg           2 x    18.00 =     36.00"
            want: "  Coffee beans 1kg             2 x    18.00 =     36.00"
            run go test -update if the change is intended
FAIL
FAIL	github.com/holmeslin/learning-go-to-senior/examples/chapter5/06_golden_file/after
FAIL
```

改格式是你要的，就執行 `go test . -update`：golden file 被重新產生，接著 `git diff` 會列出每一行的變化，commit 進去。reviewer 在 PR 裡看到的不是一大段字串，而是「這次改動讓報表變成這樣」，直接判斷對不對。沒有改任何東西時，`-update` 重新產生的檔案和原來一模一樣，`git diff` 是空的。

## 取捨

- **盲目 `-update` 會把 bug 收進預期輸出。** 上面那次失敗如果是 `Render` 真的寫壞了，執行 `-update` 之後測試轉綠，壞掉的輸出就成了「正確答案」。所以 `-update` 之後一定要看 `git diff`，而且 diff 要在 PR 裡讓別人看；不要把它放進 CI 或 `go generate`，讓測試自己改自己的答案。
- **不固定的輸出要先正規化。** 時間、map 走訪順序、隨機值、絕對路徑，每次不一樣，golden file 就每次失敗。能在來源固定就在來源固定（排序、把時間當參數傳入，時間的測試本章〈測時間與並行：用 `testing/synctest`，還是自己注入時鐘〉會談）；控制不了的，才在比對前把它換成固定字串，例如用 `regexp` 把時間戳記換成 `<time>`。
- **`-update` 是測試自己定義的旗標。** `go test ./... -update` 會在沒有定義它的套件報 `flag provided but not defined: -update`，所以對有 golden file 的套件單獨執行。
- **輸出很短時，寫在測試裡比較好讀。** `TestRenderEmpty` 的預期只有六行，輸入與預期放在同一個畫面，不需要翻檔案：

```go,ignore
{{#include ../../../examples/chapter5/06_golden_file/after/report_test.go:inline}}
```

- **golden file 把輸入與預期分在兩個地方。** 讀測試的人要開另一個檔案才知道預期是什麼，所以 golden file 適合「輸出長、格式本身就是要驗證的東西」：報表、產生的程式碼、序列化後的 JSON。

before 在這些情況下反而是對的：輸出只有一兩行，寫在測試裡最清楚；或輸出裡大部分是不相關的細節（例如一行帶時間戳記的日誌），你只在乎其中一句話，子字串檢查剛好。`Example` 的 `// Output:` 也是 golden file 的一種，輸出短、又想當文件看時用它。

## 重點整理

- 輸出很長、格式本身要驗證時，預期輸出放在 `testdata/*.golden`，測試比對完整輸出。
- 子字串只能檢查「有沒有」，檢查不了「長什麼樣子」，格式壞了照樣通過。
- `-update` 重新產生 golden file；更新之後看 `git diff`，讓 PR 的 reviewer 看到輸出的變化，不要盲目更新。
- 不固定的輸出先在來源固定（排序、傳入時間），不行再正規化，否則 golden file 會時好時壞。
- 輸出只有幾行時，寫在測試裡比較好讀。
