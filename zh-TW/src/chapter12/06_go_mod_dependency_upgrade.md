# `go.mod` 與相依升級

## 本集目標

判斷相依該怎麼升：一次升一個、每次都檢查，比 `go get -u ./...` 一口氣全升更容易找出問題在哪；並知道 `go.mod` 裡 `go`、`toolchain`、`tool`、`retract`、`exclude`、`replace` 各管什麼。

## 情境

服務的 `go.mod` 一年沒動，同事開了一個 PR：「相依都更新了」，內容是跑一次 `go get -u ./...` 的結果。`go get`、`go mod tidy` 與語意化版本前作教過：〈[`go get` 與相依版本](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/15_go_get.html)〉；`go.mod` 的基本內容見〈[`go mod init` 與 `go.mod`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/01_go_mod.html)〉。因弱點通報而升級時，怎麼判讀輸出見第 11 章〈[`govulncheck` 與相依弱點](../chapter11/04_govulncheck.md)〉。這集談的是平常升級的做法。

## 在暫存模組看升級

示範建在 repo 之外的空目錄，做完整個刪掉，本書的 `go.mod` 不會被動到。模組有兩個真的相依，起點刻意選舊版：`golang.org/x/text` v0.3.7（第 11 章用的那一版）與 `golang.org/x/sync` v0.1.0，`go` 指令設成 1.22。`GOTOOLCHAIN` 固定工具鏈，免得結果隨本機的 Go 版本變：

```bash
export GOTOOLCHAIN=go1.27.0
mkdir upgdemo && cd upgdemo && go mod init example.com/upgdemo
go mod edit -go=1.22
cat > main.go <<'EOF'
package main

import (
	"fmt"

	"golang.org/x/sync/errgroup"
	"golang.org/x/text/language"
)

func main() {
	var g errgroup.Group
	_, _, err := language.ParseAcceptLanguage("zh-TW,zh;q=0.9")
	fmt.Println(g.Wait(), err)
}
EOF
go get golang.org/x/text@v0.3.7 golang.org/x/sync@v0.1.0 && go mod tidy
```

`go list -m -u all` 列出每個模組的目前版本，方括號裡是可升的最新版。以下輸出都在 macOS（arm64）、Go 1.27.0 上取得，模組快取已有這些版本，所以沒有 `go: downloading …` 行（快取是空的就會多出來）；版本號是執行當天的最新版，重跑會不同：

執行結果（某一次）：

```text
$ go list -m -u all
example.com/upgdemo
golang.org/x/sync v0.1.0 [v0.24.0]
golang.org/x/text v0.3.7 [v0.43.0]
golang.org/x/tools v0.0.0-20180917221912-90fa682c2a6e [v0.52.0]
```

`golang.org/x/tools` 沒有被 import，為什麼在清單裡，下面的 `go mod why` 回答。

## 一次全升，還是一次升一個

先看 `go get -u ./...`。為了之後能比較，先 `cp -r . ../upgdemo-all` 複製一份，在複製出來的目錄執行：

執行結果（某一次）：

```text
$ (cd ../upgdemo-all && go get -u ./...)
go: upgraded go 1.22 => 1.26.0
go: upgraded golang.org/x/sync v0.1.0 => v0.24.0
go: upgraded golang.org/x/text v0.3.7 => v0.43.0
```

一個指令改了三件事，其中 `go` 從 1.22 變成 1.26.0：最低 Go 版本被相依悄悄拉高，CI 與同事的工具鏈都得跟上。兩個相依還看得出來；幾十個相依、`go.sum` 多出上百行時，就很難說是哪一個造成。

改成一次一個，每個之後都建置（真實的專案在這裡接 `go test ./...`），在原本的目錄進行：

執行結果（某一次）：

```text
$ go get golang.org/x/sync@latest && go build ./... && echo "build ok"
go: upgraded go 1.22 => 1.26.0
go: upgraded golang.org/x/sync v0.1.0 => v0.24.0
build ok
$ go get golang.org/x/text@latest && go build ./... && echo "build ok"
go: upgraded golang.org/x/text v0.3.7 => v0.43.0
build ok
```

這次輸出直接指出是 `golang.org/x/sync` 把 `go` 拉到 1.26.0。每升一個就提交一次，`git bisect` 與還原都有明確的單位。只想要修訂版本時用 `go get -u=patch ./...`，不升次版本。

## 為什麼在清單裡：`go mod why` 與 `go mod tidy`

`go mod why` 回答「主模組為什麼需要它」：

執行結果（某一次）：

```text
$ go mod why golang.org/x/text/language
# golang.org/x/text/language
example.com/upgdemo
golang.org/x/text/language
$ go mod why -m golang.org/x/tools
# golang.org/x/tools
(main module does not need module golang.org/x/tools)
$ go mod edit -require=golang.org/x/net@v0.1.0 && go mod tidy -v
unused golang.org/x/net
```

第一個是直接 import。第二個說這個模組不是建置所需：`go list -m` 的 `all` 列的是整張相依圖，包含只被別人的 `go.mod` 提到的模組，所以能不能刪看 `go mod why -m`，不是看清單。`go mod tidy` 依實際的 `import` 整理，`-v` 會列出被拿掉的模組；上面最後一個指令先模擬重構後留下沒人用的 `require`，再交給 `go mod tidy -v`，它拿掉了 `golang.org/x/net`。

## `go.mod` 裡其他的指令

升級會碰到 `go` 與 `toolchain` 兩行：`go` 是最低需要的 Go 版本，本機的工具鏈比它舊時，`go` 指令預設會自動下載較新的；`toolchain` 是建議的工具鏈，只在本機的工具鏈比它舊時才起作用。`tool` 把 `go tool` 要執行的工具固定在 `go.mod`，版本跟著模組走：

執行結果（某一次）：

```text
$ go get -tool golang.org/x/tools/cmd/stringer@latest
go: upgraded golang.org/x/mod v0.41.0 => v0.42.0
go: upgraded golang.org/x/tools v0.51.0 => v0.52.0
```

`go.mod` 因此多了一行 `tool golang.org/x/tools/cmd/stringer`，之後用 `go tool stringer` 執行。本書的 lint 工具也是這樣固定版本，本章〈`golangci-lint` 與 CI〉會談。其餘三個指令改變的是「用哪個版本」：

```gomod
exclude golang.org/x/text v0.40.0
replace golang.org/x/text => ../text-fork
retract v1.0.1 // 發布時誤放了測試用的金鑰
```

- `exclude`：不使用某個版本；有人要求它時，Go 改用下一個較高的版本。它與 `replace` 只在主模組的 `go.mod` 生效，相依的 `go.mod` 裡的會被忽略。
- `replace`：把模組換成別處的內容，常見是本機目錄或自己修過的分支。先 `mkdir ../text-fork && (cd ../text-fork && go mod init golang.org/x/text)` 建一個空模組，`go list -m` 的 `=>` 就是被換掉的證據：

  執行結果（某一次）：

  ```text
  $ go mod edit -replace golang.org/x/text=../text-fork
  $ go list -m golang.org/x/text
  golang.org/x/text v0.43.0 => ../text-fork
  ```
- `retract`：寫在**被發布的模組**自己的 `go.mod`，宣告某些版本不該用，`go list -m -u` 會標出。怎麼發布與撤回，本章〈發布流程與版本號〉會談。

## 取捨

- **節奏比方法重要。** 每週或每月固定升一輪，每次只有幾個小變動；一年升一次，每個相依都跨好幾個次版本，出事時要查的範圍大得多。弱點通報是例外，依第 11 章的判斷插隊處理。
- **自動化。** Dependabot 與 Renovate 會替每個相依各開一個 PR，等於把「一次一個」交給機器；前提是 CI 夠可靠，測試薄弱時只是更快合併沒人檢查的升級。
- **`go get -u ./...` 在什麼情況下是對的。** 新專案、測試完整的小工具、或專門升級的分支。它也不是完全隔離的相反：一次一個時，A 要求更高版本的 B，B 也會跟著升，所以每一步仍要看輸出。
- **`replace` 不該留在發布的模組。** 它只對自己有效：使用者拿到的是原版，你測過的與他們用的不同；`go install 模組@版本` 遇到有 `replace` 或 `exclude` 的 `go.mod` 會直接拒絕。為了等上游修正而暫時用 fork，附上 issue 連結與移除期限，發布前拿掉。
- **主版本升級是另一種成本。** `go get -u` 不會跨主版本。`v2` 以上的模組路徑多一段 `/v2`，要改每個 import，兩個主版本也可以同時在相依圖裡。先讀升級指南，獨立一個 PR 做，不要混在例行升級裡。
- **`go list -m -u` 只說有新版。** 值不值得升要看變更記錄；有沒有弱點交給 `govulncheck`。

## 重點整理

- 升級要一次一個、每次建置與測試；`go get -u ./...` 會同時改動多個相依與 `go` 版本，出問題時分不出是哪一個。
- `go list -m -u all` 看可升的版本，`go mod why` 看為什麼需要，`go mod tidy` 清掉沒用的 `require`。
- `go` 是最低 Go 版本，`toolchain` 是建議的工具鏈，`tool` 固定 `go tool` 的工具；`exclude`、`replace` 只對主模組生效，`retract` 由發布者寫在自己的 `go.mod`。
- `replace` 不留在發布的模組裡；升級要有固定節奏，主版本升級單獨處理。
