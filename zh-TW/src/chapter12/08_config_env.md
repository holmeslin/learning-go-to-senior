# 設定與環境變數

## 本集目標

判斷設定該在哪裡讀、怎麼決定優先順序、怎麼驗證：只在進入點讀一次，旗標優先於環境變數、環境變數優先於預設值，所有欄位一次驗完，錯誤在啟動時就全部報出來。

## 情境

一個轉發服務，把收到的事件轉到上游。要調的值有五個：監聽位址、上游網址、呼叫上游的逾時、失敗後的重試次數、要不要輸出除錯日誌。部署的人用環境變數設定，本機開發時想用命令列旗標臨時蓋過去。

`flag` 與結束碼前作教過：〈[`flag`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter13/02_flag.html)〉、〈[結束碼](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter13/04_exit_code.html)〉；字串轉數字與它的 `err`，見〈[`strconv.Atoi` 與 `err`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter1/16_atoi.html)〉。秘密（密碼、金鑰）怎麼讀、怎麼不被印出來，第 11 章〈[秘密管理](../chapter11/05_secret_management.md)〉談過；這集只談不是秘密的設定。

## 常見寫法

哪裡用到設定，就在哪裡呼叫 `os.Getenv`。轉數字的 `err` 用 `_` 丟掉，沒設就換成一個「應該可以」的預設：

```go,ignore
{{#include ../../../examples/chapter12/08_config_env/before/client.go:client}}
```

```go,ignore
{{#include ../../../examples/chapter12/08_config_env/before/server.go:server}}
```

這是刻意示範的寫法，完整程式碼在 `examples/chapter12/08_config_env/before/`，不要照用。

## 問題在哪

下面的 `Example` 模擬一個粗心的部署：逾時寫成 `30s`（程式要的是秒數）、重試次數打錯字、`RELAY_DEBUG=false` 想關掉除錯、上游忘了設。`// Output:` 由 `go test` 逐字比對。`setEnv` 設定變數，結束時還原，原本沒設的還原成沒設：

```go,ignore
{{#include ../../../examples/chapter12/08_config_env/before/example_test.go:setenv}}
```

```go,ignore
{{#include ../../../examples/chapter12/08_config_env/before/example_test.go:example}}
```

四個錯誤沒有一個被回報，服務照樣啟動，而且每一個都比報錯更糟：

- **逾時變成 0，而 `http.Client` 的 0 是不設逾時。** 上游一卡住，請求就永遠掛著。
- **重試次數悄悄變成 0，** 沒有人知道重試沒開。
- **`RELAY_DEBUG=false` 反而打開了除錯，** 因為程式只問「有沒有設」，不看值。
- **上游和埠號落到預設。** 正式環境的服務去連 `localhost:8080`，或監聽一個隨機埠號，要等到第一個請求失敗才發現。

另一個問題是設定散在各處：before 讀取的地方散在同一個套件的不同檔案（`client.go`、`server.go`），真實專案常散在不同套件；想知道服務吃哪些變數，得 grep 全部程式碼；測試要改一個值，只能去動整個行程的環境。

## 較好的做法

**進入點讀一次，驗證完才交出去。** 設定整理成一個 `Config`，欄位是轉換好的型別，限制寫在註解裡，`Load` 驗證過才會交出來：

```go,ignore
{{#include ../../../examples/chapter12/08_config_env/after/config.go:config}}
```

`Load` 在呼叫端給的 `FlagSet` 上註冊旗標，旗標的值只記原始字串；每個欄位依序看旗標、環境變數、預設值，取第一個有的，再轉換並驗證。環境變數設成空字串視為沒設（容器的 `ENV X=` 常這樣），上游這種沒有合理預設的欄位，三個來源都沒給就是錯誤：

```go,ignore
{{#include ../../../examples/chapter12/08_config_env/after/config.go:load}}
```

欄位的格式檢查在 `checkAddr` 與 `checkUpstream`；後者的錯誤不帶網址的內容，原因見〈取捨〉：

```go,ignore
{{#include ../../../examples/chapter12/08_config_env/after/config.go:checks}}
```

錯誤沒有一遇到就回傳，而是收集起來用 `errors.Join` 合併，這是第 2 章〈[多個錯誤要合併回傳，還是只回第一個](../chapter2/05_join_or_first_error.md)〉談過的用法。每一條都寫出欄位名稱與值來自哪裡（旗標、環境變數、預設）。before 悄悄吞掉的那幾個輸入，現在一次全部報出來。下面的 `Example` 沒有 `*testing.T`，不能用 `t.Setenv`，所以用 `setEnv`（與 before 的相同，只多一行註解）設環境變數並還原：

```go,ignore
{{#include ../../../examples/chapter12/08_config_env/after/example_test.go:setenv}}
```

```go,ignore
{{#include ../../../examples/chapter12/08_config_env/after/example_test.go:bad}}
```

**設定往下傳。** 其他元件從參數拿到 `Config`，不再自己讀環境變數：

```go,ignore
{{#include ../../../examples/chapter12/08_config_env/after/config.go:client}}
```

進入點要做的就是下面這幾步：建 `FlagSet`、`Load`、失敗就印出錯誤並結束（結束碼見前作）、成功就把 `Config` 傳下去。這個 `Example` 用 `-timeout=2s -debug` 蓋過環境變數的 `10s`，結果是 2 秒：

```go,ignore
{{#include ../../../examples/chapter12/08_config_env/after/example_test.go:example}}
```

測試用 `t.Setenv` 設環境變數，每次都建自己的 `FlagSet`，不碰全域的 `flag.CommandLine`，所以互不影響；`t.Setenv` 的測試不能 `t.Parallel`。`load` 先把五個變數都設成空字串，結果就不受開發機的環境影響：

```go,ignore
{{#include ../../../examples/chapter12/08_config_env/after/config_test.go:load}}
```

```go,ignore
{{#include ../../../examples/chapter12/08_config_env/after/config_test.go:precedence}}
```

多個錯誤一次回報、錯誤指出來源，以及各欄位的範圍檢查，各有測試。範圍檢查包括逾時要大於零，這正是 before 的 `0` 造成的問題：

```go,ignore
{{#include ../../../examples/chapter12/08_config_env/after/config_test.go:errors}}
```

```go,ignore
{{#include ../../../examples/chapter12/08_config_env/after/config_test.go:validate}}
```

這裡沒有為了測試才匯出的東西：`Load` 與 `NewClient` 是進入點真的會呼叫的 API，環境用 `t.Setenv`、旗標用測試自己的 `FlagSet`。

## 取捨

- **旗標語法錯誤只能一個一個報。** `FlagSet.Parse` 遇到未定義的旗標就停，`-h` 回傳 `flag.ErrHelp`；一次報全部的只有值的轉換與驗證。
- **優先順序要寫在文件裡。** 這裡旗標蓋過環境變數，環境變數蓋過預設值，所以旗標有效時，環境變數裡的錯誤值不會被檢查；每個旗標的說明都標出對應的變數與預設值。
- **預設值不是萬用的。** 有合理預設的（逾時 5 秒）給預設；沒有的（上游網址）寧可啟動失敗。正式環境預設去連 `localhost`，比不能啟動更難發現。
- **設定與秘密分開。** 設定可以印在啟動日誌裡、診斷端點上；秘密不行，用第 11 章〈[秘密管理](../chapter11/05_secret_management.md)〉的做法。網址若夾帶帳號密碼，整串就是秘密，這集的上游驗證因此不把網址寫進錯誤訊息。
- **設定檔與第三方套件。** 欄位多、巢狀、要給非工程師改的時候，才值得用 YAML、TOML 這類設定檔，或 Viper、koanf 這類套件，這集只提名稱。它們換來合併多個來源的功能，付出的是多一個相依，與一套要先搞懂的優先順序。[十二要素應用程式](https://12factor.net/config)的做法是設定放環境變數，同一份映像可以搬到不同環境；欄位多了以後，變數的數量與命名也要管。
- **執行中重新載入的成本。** 收到 `SIGHUP` 或監看檔案之後重讀，要保證各個 goroutine 看到一致的設定、新值驗證失敗時留著舊的，而且監聽位址這類欄位本來就改不了。多數服務啟動時讀一次、改設定就重啟比較單純，前提是關機做得好，第 10 章〈[關機時該等什麼、等多久（graceful shutdown）](../chapter10/06_graceful_shutdown.md)〉談過。
- **before 在什麼時候是對的？** 只有一個變數的小工具或一次性腳本：讀一次，缺了就直接結束，沒必要先建設定 struct。

## 重點整理

- 只在進入點讀設定，其他元件從參數拿 `Config`，欄位在傳下去之前就已經驗證過。
- 優先順序固定為旗標、環境變數、預設值；空字串的環境變數視為沒設；沒有合理預設的欄位缺值就失敗。
- 轉換的 `err` 不丟；所有欄位一次驗完，用 `errors.Join` 回報，並指出每個值的來源。
- 測試用 `t.Setenv` 與自己的 `FlagSet` 斷言優先順序與每一條錯誤；`Example` 自己還原環境變數。
- 設定不是秘密；設定檔與第三方套件看欄位多寡再決定，重新載入的成本通常高於重啟。
