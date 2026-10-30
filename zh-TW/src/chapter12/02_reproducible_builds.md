# 可重現的建置

## 本集目標

判斷要讓「同一份原始碼建出位元組相同的執行檔」得控制哪些輸入：路徑、工具鏈版本、時間戳記；哪些（cgo、VCS 狀態、相依來源）超出 `go build` 的掌握。

## 情境

稽核要確認線上跑的執行檔真的是某個 commit 建出來的：CI 建一次，另一個人在自己的機器再建一次，比雜湊。對不起來時，沒人說得出是程式不同，還是建置環境不同。版本資訊怎麼寫進執行檔、`-ldflags -X` 怎麼用，本章〈`go build` 與用 `-ldflags` 注入版本〉談過；`go.mod` 前作教過：〈[`go mod init` 與 `go.mod`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/01_go_mod.html)〉。

## 兩個目錄，兩個雜湊

示範在 repo 之外的暫存目錄做。程式只印出自己的原始碼路徑，與一個可由 `-X` 設定的字串：

```go
package main

import (
	"fmt"
	"runtime"
)

// buildTime 由建置時的 -ldflags -X 設定，沒有設定就是 unknown。
var buildTime = "unknown"

func main() {
	_, file, _, _ := runtime.Caller(0)
	fmt.Println("source:", file)
	fmt.Println("built:", buildTime)
}
```

把它存成 `demo/a/main.go`，再複製成 `demo/b`。兩個目錄內容逐位元組相同，只差目錄名：

```bash
export GOTOOLCHAIN=go1.27.0
mkdir -p demo/a && cd demo/a && go mod init example.com/repro
# 把上面的程式存成這個目錄的 main.go
cd .. && cp -r a b
```

以下在 macOS（arm64）、Go 1.27.0 取得。輸出裡 `demo` 之前的絕對路徑換成 `$WORK`；`go tool buildid` 的輸出用 `cut` 只留前 20 個字元：

執行結果（某一次）：

```text
$ for d in a b; do (cd $d && go build -o ../bin-$d .); done
$ shasum -a 256 bin-a bin-b
0a6571ee23a45694d259cadff3495197f68a8787b2cdfef4366fc267885066b9  bin-a
8e2c9040e3ade0f8934c7601f341c1f21ba51cc1266131b53a04796eb7d87c85  bin-b
$ for d in a b; do ./bin-$d | head -1; done
source: $WORK/demo/a/main.go
source: $WORK/demo/b/main.go
$ for d in a b; do go tool buildid bin-$d | cut -c1-20; done
1xXM6_N7oO0jkud0ohTO
pe8DOdkFdcBUyuaYGvj5
$ strings bin-a | grep /a/main.go
$WORK/demo/a/main.go
```

雜湊不同，建置 ID 也不同。差別只在路徑：程式印出的 `source:` 是編譯時的絕對路徑，`strings` 在執行檔裡找到同一串。它是給 stack trace 與除錯器用的，panic 訊息、`slog` 的 `AddSource` 帶的也是它。同一個目錄連建兩次，雜湊是相同的（在 `a` 連跑兩次 `go build` 比比看），所以變數是路徑，不是建置過程的隨機性。

## 加上 `-trimpath`

`-trimpath` 把記錄的檔案路徑改成模組路徑，標準庫則是 import path。同樣的兩個目錄，只多這個旗標。`go version -m` 的輸出只留第一行與 `-trimpath`、`CGO_ENABLED`、`GOARM64` 三行：

執行結果（某一次）：

```text
$ for d in a b; do (cd $d && go build -trimpath -o ../bin-$d .); done
$ shasum -a 256 bin-a bin-b
f2646e6f56ea3c80331dfc1068d1d90d17314616926c685d106c535129ace731  bin-a
f2646e6f56ea3c80331dfc1068d1d90d17314616926c685d106c535129ace731  bin-b
$ for d in a b; do ./bin-$d | head -1; done
source: example.com/repro/main.go
source: example.com/repro/main.go
$ go version -m bin-a | grep -E 'go1|trimpath|CGO_ENABLED|GOARM64'
bin-a: go1.27.0
	build	-trimpath=true
	build	CGO_ENABLED=1
	build	GOARM64=v8.0
$ strings bin-a | grep -c /a/main.go
0
```

雜湊相同，`source:` 變成 `example.com/repro/main.go`，執行檔裡找不到目錄名了。相依套件的路徑（模組快取目錄）也一併改寫成「模組路徑@版本」，所以不必要求每個人的 `GOPATH` 一致。

## 工具鏈與時間戳記

**工具鏈。** 執行檔含編譯器、連結器與 runtime，上面 `go version -m` 的 `go1.27.0` 就是它們的版本；別的 Go 版本建出來的不會相同。`go.mod` 的 `go` 行寫到小版本（`go mod init` 在 1.27 寫的就是 `go 1.27.0`），預設的 `GOTOOLCHAIN=auto` 會讓較舊的 `go` 指令切到不低於它的工具鏈，`toolchain go1.27.0` 這一行可以再指定偏好的版本；`GOTOOLCHAIN=go1.27.0` 則讓所有人用同一版，`go.mod` 要求更高的版本時直接失敗，不會悄悄換。另外 `go version -m` 也列出 `CGO_ENABLED`、`GOARM64` 這類建置設定，兩邊的環境變數要一致。PGO 的 `default.pgo` 同理是輸入，第 7 章〈[PGO](../chapter7/07_pgo.md)〉談過要 commit 進 repo。

**時間戳記。** Go 不會自己把建置時間寫進執行檔。常見的問題是自己注入：`-X main.buildTime=$(date …)`。每個目錄與旗標都相同，間隔一秒建兩次：

執行結果（某一次）：

```text
$ for i in 1 2; do (cd a && go build -trimpath -ldflags "-X main.buildTime=$(date -u +%FT%TZ)" -o ../bin-t$i .); sleep 1; done
$ shasum -a 256 bin-t1 bin-t2
4f33edbaba2f88d41f232709a12b2c19dfa49fc5d9de7e9864cb642409bb2909  bin-t1
f617a0e87709b2c703b3633732279820dfac5332db90dc576af355a150b8725e  bin-t2
$ for i in 1 2; do ./bin-t$i | tail -1; done
built: 2026-10-30T05:43:25Z
built: 2026-10-30T05:43:26Z
```

雜湊不同，差別就是那個時間字串。要有時間資訊，用 commit 的時間（`vcs.time` 已經記了）或 `git log -1 --format=%cI` 的值，每次建置都一樣。

## 取捨

- **cgo 與外部連結器。** 啟用 cgo 時，`go build` 會呼叫系統的 C 編譯器，必要時還用外部連結器。它們的版本與系統標頭、函式庫不在 `go.mod` 與 `GOTOOLCHAIN` 的管轄內，相同的 Go 版本在不同機器可能建出不同的執行檔。上面 `go version -m` 的 `CGO_ENABLED=1` 就是預設；要可驗證，用 `CGO_ENABLED=0`（本章〈交叉編譯〉會談）或固定的建置容器（本章〈容器映像（multi-stage）〉會談）。
- **`-buildvcs` 與 dirty 狀態。** 在 git 目錄建置，執行檔會多記 `vcs.revision`、`vcs.time`、`vcs.modified`。從同一個 commit 的兩個乾淨 clone 建，雜湊相同；工作目錄有未提交的變更，連未追蹤的檔案，都會讓 `vcs.modified` 變 `true`，雜湊也跟著變：

  ```bash
  git -C a init -q && git -C a add . && git -C a commit -q -m init
  git clone -q a c && git clone -q a d
  ```

  執行結果（某一次）：

  ```text
  $ for d in c d; do (cd $d && go build -trimpath -o ../bin-$d .); done
  $ shasum -a 256 bin-c bin-d
  29ec9c561aa24e0c6e92930c092b2c6035fa77e184d09f8feabb25123b849db4  bin-c
  29ec9c561aa24e0c6e92930c092b2c6035fa77e184d09f8feabb25123b849db4  bin-d
  $ go version -m bin-c | grep vcs
  	build	vcs=git
  	build	vcs.revision=67f8fd60223f1b108d5f0ae3c544ff28ee744aac
  	build	vcs.time=2026-10-30T05:43:42Z
  	build	vcs.modified=false
  $ echo x > d/notes.txt; (cd d && go build -trimpath -o ../bin-d2 .)
  $ shasum -a 256 bin-d2
  d46d1bb158928a20583b25d6461ba2c1cca11246bff9449d0172625ce24003b7  bin-d2
  $ go version -m bin-d2 | grep vcs.modified
  	build	vcs.modified=true
  $ (cd d && go build -trimpath -buildvcs=false -o ../bin-d3 .)
  $ shasum -a 256 bin-d3
  f2646e6f56ea3c80331dfc1068d1d90d17314616926c685d106c535129ace731  bin-d3
  ```

  `go version -m` 的輸出只留含 `vcs` 的行。`vcs.revision` 與 `vcs.time` 是這次 commit 的，每次都不同。最後一組加 `-buildvcs=false`，雜湊與上面沒有 VCS 資訊的 `-trimpath` 建置相同，代價是執行檔不再說得出自己是哪個 commit。為了可重現關掉它不划算；做法是兩邊都從同一個乾淨的 checkout 建。
- **模組代理與 `go.sum`。** `go.sum` 記每個相依版本內容的雜湊，`go build` 預設 `-mod=readonly` 並驗證，所以相同的 `go.mod` 與 `go.sum` 要嘛拿到記錄過的那份原始碼，要嘛報錯，不會悄悄換成別的內容；預設的代理也讓版本被刪後仍拿得到。保護的只有相依的原始碼：不涵蓋工具鏈、C 函式庫與環境變數。示範的模組沒有相依，所以沒有 `go.sum`。
- **工具鏈本身。** Go 1.21 起，官方發行版的工具鏈本身可以重現（見〈[Perfectly Reproducible, Verified Go Toolchains](https://go.dev/blog/rebuild)〉），別人可以驗證你用的 `go1.27.0` 就是官方那一份。
- **不一定要做。** 只在單一台機器、同一個目錄建置時，不加 `-trimpath` 照樣可以重現；絕對路徑還讓 stack trace 能在編輯器直接跳到檔案，加了之後除錯器要自己設路徑對應。需要「別人能驗證」或跨機器比對時才值得。

## 重點整理

- 路徑、Go 版本、自己注入的時間戳記是可以自己控制的輸入；兩個目錄的雜湊不同，原因是路徑被寫進執行檔，`-trimpath` 處理它，證明的方法是兩次建置的雜湊相同。
- 用 `go.mod` 的 `go 1.27.0` 與 `GOTOOLCHAIN` 固定工具鏈；不要在建置時注入時間，需要就用 commit 的時間。
- cgo 與 VCS 狀態超出這兩招：要驗證就 `CGO_ENABLED=0`、從乾淨的 checkout 建，`-buildvcs=false` 會丟掉可追溯性。
