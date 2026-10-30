# `govulncheck` 與相依弱點

## 本集目標

判斷 `govulncheck` 報出的弱點哪些要馬上處理：程式碼呼叫得到的要修，只是相依裡有、沒有碰到的排進一般升級；並知道這個掃描該放在流程的哪裡。

## 情境

服務依賴十幾個模組，只比對版本號的掃描器每週開出一串升級通知，分不出哪一個真的碰得到。`govulncheck` 多做一步：分析原始碼的呼叫關係，只把「你的程式碼呼叫得到有弱點的函式」當成受影響。它查的是 Go 團隊維護的弱點資料庫，請求裡只有已知有弱點的模組路徑，不含你的程式碼。

這集談讀輸出與判斷，升級的流程本身第 12 章〈`go.mod` 與相依升級〉會談；`go.mod` 的基本用法前作教過：〈[`go mod init` 與 `go.mod`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/01_go_mod.html)〉。

## 在暫存模組重現弱點

示範需要一個依賴舊版本的模組，所以建在 repo 之外的空目錄；本書 repo 的 `go.mod` 不會出現有弱點的版本，這個模組做完也整個刪掉。程式只做一件事：用 `golang.org/x/text` 的 `language.ParseAcceptLanguage` 解析 `Accept-Language` 標頭。`GOTOOLCHAIN` 固定工具鏈，因為掃描結果與 Go 版本有關：

```bash
export GOTOOLCHAIN=go1.27.0
mkdir vulndemo && cd vulndemo
go mod init example.com/vulndemo
cat > main.go <<'EOF'
package main

import (
	"fmt"

	"golang.org/x/text/language"
)

func main() {
	tags, _, err := language.ParseAcceptLanguage("zh-TW,zh;q=0.9,en;q=0.8")
	fmt.Println(len(tags), err)
}
EOF
go get golang.org/x/text@v0.3.7
go mod tidy
```

`govulncheck` 一律固定版本用 `go run` 執行，不寫進 `go.mod`。以下輸出在 macOS（arm64）、Go 1.27.0、`govulncheck` v1.9.0 上取得，弱點資料庫的版本是 2026-10-27 更新的（`-format json` 輸出的第一筆 `config.db_last_modified`）；輸出完整，最後一行 `exit status 3` 是 `go run` 轉述 `govulncheck` 的結束碼，`go run` 自己以 1 結束：

執行結果（某一次）：

```text
$ go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...
=== Symbol Results ===

Vulnerability #1: GO-2022-1059
    Denial of service via crafted Accept-Language header in
    golang.org/x/text/language
  More info: https://pkg.go.dev/vuln/GO-2022-1059
  Module: golang.org/x/text
    Found in: golang.org/x/text@v0.3.7
    Fixed in: golang.org/x/text@v0.3.8
    Example traces found:
      #1: main.go:10:46: vulndemo.main calls language.ParseAcceptLanguage

Your code is affected by 1 vulnerability from 1 module.
This scan also found 1 vulnerability in packages you import and 15
vulnerabilities in modules you require, but your code doesn't appear to call
these vulnerabilities.
Use '-show verbose' for more details.
exit status 3
```

## 讀輸出：符號、套件、模組

結論分三層，從最嚴重到最輕：

- **符號（Symbol）**：呼叫關係走得到有弱點的函式。`Example traces found` 那一行指出是 `main.go` 第 10 行呼叫 `language.ParseAcceptLanguage`，要修的人從這裡開始。預設輸出只展開這一層，也只有這一層會讓結束碼不是 0。
- **套件（Package）**：程式碼 import 了有弱點的套件，但沒有呼叫有問題的函式。
- **模組（Module）**：`go.mod` 的依賴圖裡有，程式碼連有弱點的套件都沒 import。

後兩層預設只剩結尾那段計數，加 `-show verbose` 才列出內容。下面只留開頭的掃描範圍、整個 Package Results，與 Module Results 裡的一項（其餘 14 項省略）：

執行結果（某一次）：

```text
$ go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 -show verbose ./...
Govulncheck scanned the following 2 modules and the go1.27.0 standard library:
  example.com/vulndemo
  golang.org/x/text@v0.3.7

=== Package Results ===

Vulnerability #1: GO-2026-6604
    Root.Mkdir(All) can follow junctions out of the root on Windows in os
  More info: https://pkg.go.dev/vuln/GO-2026-6604
  Standard library
    Found in: os@go1.27
    Fixed in: os@go1.27.2
    Platforms: windows

=== Module Results ===

Vulnerability #2: GO-2026-6629
    Panic parsing crafted input in x/text/secure/precis in golang.org/x/text
  More info: https://pkg.go.dev/vuln/GO-2026-6629
  Module: golang.org/x/text
    Found in: golang.org/x/text@v0.3.7
    Fixed in: golang.org/x/text@v0.41.0
```

掃描範圍包含標準庫：那一項套件層級的弱點在 `os`，而且 `Platforms: windows`，部署在 Linux 的服務不受影響。15 項模組層級裡有 13 項也是標準庫（`Standard library`），只有 2 項是 `x/text`。

## 升級後重新掃描

把 `x/text` 升到 v0.43.0（平常會打 `@latest`，這裡固定成實際執行的版本），再掃一次：

執行結果（某一次）：

```text
$ go get golang.org/x/text@v0.43.0
go: upgraded golang.org/x/text v0.3.7 => v0.43.0
$ go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...
=== Symbol Results ===

No vulnerabilities found.

Your code is affected by 0 vulnerabilities.
This scan also found 1 vulnerability in packages you import and 13
vulnerabilities in modules you require, but your code doesn't appear to call
these vulnerabilities.
Use '-show verbose' for more details.
```

符號層級是空的，結束碼 0，這才是「乾淨」。計數沒有歸零：加 `-show verbose` 看，剩下的 14 項全是 `Standard library`，輸出的 `Fixed in` 指向更新的 Go 小版本，所以要靠升級工具鏈，改 `go.mod` 沒用。弱點資料庫天天更新，重跑時編號與數量會不同，判斷看的是這份輸出的結構。

## 取捨

- **排程還是擋部署？** 資料庫每天更新，昨天通過的程式碼今天可能就失敗，與這次變更無關。每次部署都擋，會讓不相關的變更被新公布的弱點卡住。常見的做法是排程每天掃一次，失敗就開 issue 或通知；真的要擋，只擋符號層級，並保留放行的辦法。本書不新增 CI 步驟，CI 的設定第 12 章〈`golangci-lint` 與 CI〉會談。
- **結束碼要看清楚。** 文件寫明有弱點就以非 0 結束，但加 `-format json`、`sarif`、`openvex` 時不論有沒有弱點都是 0（這份暫存模組的 JSON 輸出實測也是 0）。要把 3 與其他錯誤分開，先 `go install` 同一個版本，不用 `go run`。
- **二進位檔模式。** `-mode binary` 掃的是實際部署的檔案，用的是建置當時的 Go 版本與相依。對上面的暫存模組建出的二進位檔加 `-mode binary` 掃描，符號層級一樣報出同一項，但沒有呼叫路徑，只列出符號，「import 的套件」那一項計數也變成 0：

  ```text
      Vulnerable symbols found:
        #1: language.ParseAcceptLanguage
  ```

  文件說明二進位檔沒有呼叫資訊，也可能把在檔案裡但呼叫不到的程式算進來，所以多報的機率比原始碼模式高。
- **升級不了時怎麼評估。** 先看 trace：是哪一行呼叫、參數是不是外部輸入（這個例子的弱點與 `Accept-Language` 標頭有關，標頭正是使用者可控制的）。再看 `Platforms` 與弱點描述，確認自己的環境碰不碰得到。評估完還是升不了，選項是換掉套件，或在 `go.mod` 用 `replace` 指向自己修過的分支。`govulncheck` 沒有靜音的機制，例外要自己記在 issue 裡，寫明原因與重新檢視的日期。
- **「沒呼叫到」不等於安全。** 呼叫圖是靜態分析：透過 `reflect` 的呼叫看不到，會漏報；介面與函式指標的呼叫分析得很保守，會多報。套件與模組層級的項目也是遲早要升級，只是不必今天處理，排進第 12 章〈`go.mod` 與相依升級〉會談的例行升級。
- **只比對版本號的掃描器仍有用。** 它不限於 Go、不需要能編譯，也能涵蓋 `govulncheck` 不分析的部分；缺點是無法區分呼叫得到與否，通知多、能處理的少。

## 重點整理

- `govulncheck` 分析呼叫關係，輸出分符號、套件、模組三層；符號層級才是你的程式碼呼叫得到的，要先處理。
- 判斷靠輸出的結構（`Example traces found`、`Platforms`、`Fixed in`），不靠某個弱點編號；資料庫天天更新，重跑結果會變。
- 「乾淨」是符號層級為空、結束碼 0；標準庫的弱點要靠升級工具鏈，不是升級相依。
- 排程執行比每次部署都擋更穩；JSON 類輸出的結束碼永遠是 0，用 `go run` 時要自己處理結束碼。
- 呼叫圖有限制：`reflect` 看不到、二進位檔沒有呼叫路徑；升級不了就要有記錄過的評估，不是靜音。
