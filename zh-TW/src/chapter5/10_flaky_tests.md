# 不穩定（flaky）測試

## 本集目標

看到時好時壞的測試，先判斷是不是測試之間共用了狀態；知道 `-race`、`-shuffle`、`-count`、`-failfast` 各自抓什麼，以及團隊遇到 flaky 測試時該怎麼處理。

## 情境

你的服務有一個命令註冊表：`Register` 把名稱和處理函式配成一對，`Run` 照名稱執行。為了方便，註冊表是套件層級的變數，和 `http.DefaultServeMux` 一樣，誰都可以直接 `Register`。測試很自然地寫成兩個：一個測註冊，一個測執行。`go test` 與 `-race` 的用法前作教過：〈[`go test`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/09_go_test.html)〉、〈[data race 與 `-race`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter9/03_data_race.html)〉。這集談的是另一類 flaky：測試之間的順序相依。

## 常見寫法

```go,ignore
{{#include ../../../examples/chapter5/10_flaky_tests/before/registry.go:registry}}
```

```go,ignore
{{#include ../../../examples/chapter5/10_flaky_tests/before/registry_test.go:test}}
```

完整程式碼在 `examples/chapter5/10_flaky_tests/before/`。`TestRun` 沒有自己註冊 `echo`，它用的是前一個測試留下的東西。預設的 `go test`、`go test -race` 都通過，CI 也一直是綠的。

## 問題在哪

測試的執行順序一換，或同一個行程裡跑第二遍，就會失敗。套件的文件註解寫明了重現的指令，這裡用 `-shuffle=2`，讓 `go test` 打亂測試的順序，數字是亂數的種子，同一個種子永遠是同一個順序：

```text
-test.shuffle 2
--- FAIL: TestRun
    registry_test.go:22: Run("echo") error = run "echo": unknown command, want nil
FAIL
FAIL	.../10_flaky_tests/before
FAIL
```

`-count=2` 讓每個測試在同一個行程裡跑兩遍，第二遍 `echo` 已經在表裡了：

```text
--- FAIL: TestRegister
    registry_test.go:15: Register("echo") error = register "echo": already registered, want nil
FAIL
FAIL	.../10_flaky_tests/before
FAIL
```

這兩段是自己跑出來再貼的，省略了每個測試後面的耗時，以及最後一行的套件路徑前綴與耗時。`-shuffle=1` 剛好排成和預設相同的順序，所以通過：種子要試過才知道。單獨執行 `go test -run TestRun .` 同樣失敗，這就是「整包跑都過，單跑一個就掛」的來源。

換成 `-shuffle=on`，種子由系統時鐘決定，`go test` 會把它印在第一行（`-test.shuffle` 後面那串數字）。CI 上遇到失敗，把這個數字抄下來帶回本機，就能照同一個順序重跑。這讓「偶爾紅一次」變成「給我這個種子，我穩定重現」。

## 較好的做法

讓每個測試建立自己的狀態。註冊表改成一個型別，`New` 每次回傳一份新的：

```go,ignore
{{#include ../../../examples/chapter5/10_flaky_tests/after/registry.go:registry}}
```

```go,ignore
{{#include ../../../examples/chapter5/10_flaky_tests/after/registry_test.go:test}}
```

`TestRun` 自己註冊自己要用的 `echo`，不管誰先跑、跑幾遍，都從空的註冊表開始。`Example` 同樣自己建立註冊表，輸出由 `go test` 逐字驗證：

```go,ignore
{{#include ../../../examples/chapter5/10_flaky_tests/after/example_test.go:example}}
```

這份 after 在 `go test -shuffle=on`（連跑 30 次）、`-shuffle=1` 到 `-shuffle=20`、`-count=3` 與 `-race -shuffle=on -count=5` 下都通過。程式本身只多了一個 `Registry` 型別，沒有任何魔法。

## 取捨

四個旗標各抓不同的事，不能互相取代：

- **`-race`** 抓資料競爭：兩個 goroutine 同時存取同一塊記憶體、至少一個在寫。上面的 before 在 `-race` 下通過，因為測試是一個接一個跑的，沒有競爭，只有順序相依。
- **`-shuffle`** 抓順序相依：某個測試靠前一個測試留下的東西才會過。
- **`-count`** 抓殘留狀態：測試改了全域的東西卻沒還原，第二遍就看得出來。`-count=2` 是最便宜的檢查，在本機提交前跑一次就夠。
- **`-failfast`** 不抓問題，只縮短除錯時間：第一個失敗之後不再開始新的測試。搭配固定的 `-shuffle=N` 很順手；代價是你看不到其他測試是不是也壞了。

遇到 flaky 測試，團隊的規則是：

1. **重現。** 用 `-shuffle=on` 的種子、`-count`、`-run` 把它變成穩定失敗。重現不了的才需要懷疑是環境或時間。
2. **修，或隔離。** 能修就修：讓測試自己建立狀態。一時修不了，用 `t.Skip` 暫時隔離，原因寫上 issue 編號，並真的開那張 issue。
3. **不要重跑到過。** CI 上自動重試會讓失敗變成雜訊，之後真的壞掉時沒人相信紅燈。

依賴時間長短的 flaky（逾時、重試、睡一下等結果）是另一個來源，不屬於順序相依，做法見本章〈測時間與並行：用 `testing/synctest`，還是自己注入時鐘〉。

after 也不是「所有東西都不能共用」。建起來很貴、而且測試只讀不寫的 fixture 值得共用，例如整份載入一次的測試資料。條件是唯讀：只要有一個測試會改它，就回到 before 的問題。after 測試裡的 `echo` 就是這樣的例子，它是沒有狀態的函式。

before 的寫法在這些情況下反而是對的：整個流程本來就是一連串依序的步驟（註冊、執行、解除註冊），那就寫成一個測試，用 `t.Run` 明講順序，讓依賴寫在程式裡，而不是靠檔案裡誰排前面。如果程式非得有套件層級的預設實例，測試至少要在 `t.Cleanup` 裡把它還原。

## 重點整理

- 測試之間共用可變狀態，會讓順序一換或跑第二遍就失敗；每個測試建立自己的狀態。
- `-shuffle=N` 的種子與 `-count=2` 可以把「偶爾紅」變成穩定失敗；`-shuffle=on` 會印出種子，CI 失敗時記下它。
- `-race` 抓資料競爭、`-shuffle` 抓順序相依、`-count` 抓殘留狀態、`-failfast` 只是縮短除錯時間。
- 處理規則：重現、修正或隔離並開 issue，不重跑到過。
- 唯讀的共用 fixture 沒問題；只要有人會改，就要拆開。
