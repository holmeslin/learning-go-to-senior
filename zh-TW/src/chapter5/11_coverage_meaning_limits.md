# 覆蓋率的意義與限制

## 本集目標

看懂覆蓋率數字量的是「有執行」，不是「有驗證」；知道怎麼用 `-coverprofile` 找出沒被測到的分支，把覆蓋率當成找漏洞的工具，而不是要達標的數字。

## 情境

帳單要分給幾個人，`split.Split(total, weights...)` 依權重的比例把金額（單位：分）分成幾份，總和一定等於 `total`；沒有權重、權重不是正數、金額是負數時回傳錯誤。團隊規定覆蓋率要過九成，才能合併。`go test` 的基本用法前作教過：〈[`go test`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/09_go_test.html)〉。這集談覆蓋率。

## 常見寫法

為了讓數字好看，每個分支都呼叫一次，但沒有人去看結果：

```go,ignore
{{#include ../../../examples/chapter5/11_coverage_meaning_limits/before/split_test.go:tests}}
```

完整程式碼在 `examples/chapter5/11_coverage_meaning_limits/before/`。

## 問題在哪

先量覆蓋率。`Example` 也算進覆蓋率，所以用 `-run '^Test'` 只跑測試函式；輸出省略了套件的耗時：

```text
$ go test -count=1 -run '^Test' -cover ./examples/chapter5/11_coverage_meaning_limits/before
ok  	github.com/holmeslin/learning-go-to-senior/examples/chapter5/11_coverage_meaning_limits/before	coverage: 93.8% of statements
```

93.8%，過關。但 `Split` 有兩個錯誤路徑的 bug，下面的 `Example` 印出它們（註解是預期的結果）：

```go,ignore
{{#include ../../../examples/chapter5/11_coverage_meaning_limits/before/example_test.go:example}}
```

權重是負的，該回 `weight must be positive`，卻回了 `no weights`，錯誤的種類不對。金額是負的，該回錯誤，卻靜靜地回了兩份 0，這筆錢就消失了。

第一個 bug 在被執行過的程式裡：`TestSplitBadWeight` 跑過那一行，覆蓋率算它有測到，但結果沒有人檢查。第二個 bug 在沒有被執行過的分支裡，93.8% 裡缺的那 6.2% 就是它。覆蓋率高，不代表這些測試抓得到任何東西；把 `Split` 的演算法整個改壞，只要不 panic，三個測試照樣全過。

## 較好的做法

不要看總數，看是哪幾行沒被跑到。`-coverprofile` 把每個程式區塊的執行次數寫進檔案，`go tool cover -func` 按函式整理：

```text
$ go test -count=1 -run '^Test' -coverprofile=cover.out ./examples/chapter5/11_coverage_meaning_limits/before
$ go tool cover -func=cover.out
github.com/holmeslin/learning-go-to-senior/examples/chapter5/11_coverage_meaning_limits/before/split.go:18:	Split		93.8%
total:														(statements)	93.8%
```

`-func` 只告訴你哪個函式不到 100%。要知道是哪幾行，找 `cover.out` 裡次數是 0 的區塊（欄位依序是檔案與起訖行列、陳述式數、執行次數），或執行 `go tool cover -html=cover.out` 在瀏覽器裡看紅色的程式碼：

```text
$ grep ' 0$' cover.out
github.com/holmeslin/learning-go-to-senior/examples/chapter5/11_coverage_meaning_limits/before/split.go:23.3,24.1 1 0
```

第 23 行的 `return`，正是 `total < 0` 那個分支裡唯一的陳述式。補上測試的重點是**斷言**：錯誤路徑回了什麼錯誤、有沒有同時回傳份額。一般路徑用一張形狀一致的表，錯誤路徑用另一張：

```go,ignore
{{#include ../../../examples/chapter5/11_coverage_meaning_limits/after/split_test.go:shares}}
```

```go,ignore
{{#include ../../../examples/chapter5/11_coverage_meaning_limits/after/split_test.go:errors}}
```

失敗訊息寫出得到什麼、預期什麼。這些測試對著 before 的程式跑，兩個 bug 都會被抓到（我把 after 的測試放進暫存 module，配上 before 的 `split.go`，省略了耗時）：

```text
--- FAIL: TestSplitErrors (0.00s)
    --- FAIL: TestSplitErrors/權重為負 (0.00s)
        split_test.go:58: Split(100, [1 -1]) error = no weights, want weight must be positive
    --- FAIL: TestSplitErrors/權重為_0 (0.00s)
        split_test.go:58: Split(100, [1 0]) error = no weights, want weight must be positive
    --- FAIL: TestSplitErrors/金額為負 (0.00s)
        split_test.go:58: Split(-100, [1 1]) error = <nil>, want total must not be negative
        split_test.go:61: Split(-100, [1 1]) = [0 0], want nil shares on error
FAIL
```

修好 `Split` 之後，`Example` 的輸出變成：

```go,ignore
{{#include ../../../examples/chapter5/11_coverage_meaning_limits/after/example_test.go:example}}
```

注意分工：第一個 bug 是覆蓋率看不出來的（那一行有跑到），是斷言抓到的；第二個 bug 是覆蓋率指出的（那一行沒跑到），是補測試抓到的。覆蓋率負責回答「哪裡我還沒看過」，「看過之後對不對」要靠斷言。

## 取捨

- **覆蓋率只說明「有執行」。** 100% 的意思是每個陳述式都跑過，不是每個行為都驗證過。after 的測試是 100%，before 加上 `Example` 之後也是 100%（`Example` 的輸出雖然有比對，卻只比對三組輸入）。
- **適合當找漏洞的工具與趨勢指標。** 改完程式看一眼哪些新分支是紅的；看覆蓋率隨時間往下掉，代表有人在沒有測試的情況下加程式。這兩種用法不需要規定一個數字。
- **不適合當硬門檻。** 門檻一設，最省力的做法就是 before：呼叫、不斷言、數字過關。需要門檻時，至少讓 reviewer 看得到哪些行是新增而沒被測到的；CI 的設定本書第 12 章〈`golangci-lint` 與 CI〉會談。
- **追到 100% 不一定划算。** 為了跑到一個 `os.Open` 失敗的分支而大費周章，成本可能遠高於它的風險。看紅色的行，判斷哪些值得補。
- **覆蓋率不會告訴你缺的程式碼。** 少寫的檢查沒有行可以標紅。表格該有哪些案例（零、負數、邊界），要靠本章〈[測什麼、不測什麼](01_what_to_test.md)〉那樣的判斷，不是靠數字。
- **整合測試的覆蓋率只提一句：** Go 1.20 起可以用 `go build -cover` 建出會記錄覆蓋率的執行檔，或用 `-coverpkg` 把其他套件的程式算進來，細節請看官方文件〈[Coverage profiling support for integration tests](https://go.dev/doc/build-cover)〉。

before 在什麼情況下是對的？當測試的目的就是「確定它不會 panic」，例如對一個很難構造預期值的解析器餵一批輸入，這類冒煙測試不看結果是刻意的；但它該被當成冒煙測試，不該算進「有驗證」。

## 重點整理

- 覆蓋率量的是有沒有執行，不是有沒有驗證；沒有斷言的測試也能把它衝高。
- `-coverprofile` 加 `go tool cover -func`、`-html`，或找 `cover.out` 裡次數為 0 的區塊，可以看出哪些分支從沒跑過，錯誤路徑最常是其中之一。
- 補測試的重點是斷言：回了什麼錯誤、有沒有同時回傳不該有的值，失敗訊息要寫出得到什麼、預期什麼。
- 覆蓋率適合當找漏洞的工具與趨勢指標，不適合當硬門檻；100% 也不代表沒有 bug。
- `Example` 也算進覆蓋率；缺少的程式碼不會被標成紅色。
