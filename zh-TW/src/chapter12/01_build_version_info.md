# `go build` 與用 `-ldflags` 注入版本

## 本集目標

判斷版本資訊該從哪裡來：commit、時間、有沒有未提交的修改與主模組版本，`go build` 已經記在執行檔裡，用 `debug.ReadBuildInfo` 讀；`-ldflags -X` 只留給建置系統才知道的值。

## 情境

`versiond` 是一支小服務：有 `-version` 旗標與 `/version` 端點，啟動時也在日誌印出版本。線上出問題，第一個問題是「現在跑的是哪個 commit」，答案要來自執行檔本身，不是靠人記得。`flag` 與 `go.mod` 前作教過：〈[`flag`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter13/02_flag.html)〉、〈[`go mod init` 與 `go.mod`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/01_go_mod.html)〉。示範都在 repo 外的暫存目錄建置，tag 只存在於暫存的 git repo。

## 常見寫法

版本是一個字串變數，發版的建置指令用 `-X` 把它設成版本號：

```go,ignore
{{#include ../../../examples/chapter12/01_build_version_info/before/main.go:version}}
```

```bash
go build -ldflags "-X main.version=v1.4.0" -o versiond .
```

這行指令寫在 Makefile 或 CI 設定裡，發版的人改它。剛發版的時候，它是對的。完整程式碼在 `examples/chapter12/01_build_version_info/before/`。

## 問題在哪

`mkdemo` 把範例複製到暫存目錄，建一個 git repo：提交並打上 `v1.4.0`，之後又提交一次，最後留一處沒提交的修改。這是 tag 之後有人繼續改、建置指令卻沒人記得更新的狀況。`repo` 是本書 repo 的根目錄，指令從那裡開始：

```bash
export GOTOOLCHAIN=go1.27.0
repo=$PWD demo=/tmp/versiond-demo
mkdemo() { # 用法：mkdemo <原始碼目錄> <模組名稱>
	rm -rf $demo && mkdir -p $demo/bin && cp -r "$1" $demo/src && cd $demo/src || return
	go mod init "$2" && git init -q && git config user.name demo && git config user.email demo@example.com
	git add . && git commit -qm release && git tag -m v1.4.0 v1.4.0
	echo '// fix' >> main.go && git commit -qam fix
	echo '// wip' >> main.go
}
mkdemo $repo/examples/chapter12/01_build_version_info/before example.com/versiond
go build -ldflags "-X main.version=v1.4.0" -o ../bin/versiond .
../bin/versiond -version
go version -m ../bin/versiond | grep -e ': go1' -e $'\tmod' -e ldflags -e vcs
```

本集的輸出都在 macOS（arm64）、Go 1.27.0、git 2.54.0 上取得；`go version -m` 的 `path` 行與其他 `build` 行（`-buildmode`、`-compiler`、`CGO_*`、`GOARCH`、`GOOS` 等）被 `grep` 濾掉了：

執行結果（某一次）：

```text
$ ../bin/versiond -version
v1.4.0
$ go version -m ../bin/versiond | grep -e ': go1' -e $'\tmod' -e ldflags -e vcs
../bin/versiond: go1.27.0
	mod	example.com/versiond	v1.4.1-0.20261030054345-7a0c240b07b4+dirty
	build	-ldflags="-X main.version=v1.4.0"
	build	vcs=git
	build	vcs.revision=7a0c240b07b41e3a3f1b15d168f8056f5af39e0e
	build	vcs.time=2026-10-30T05:43:45Z
	build	vcs.modified=true
```

程式說自己是 `v1.4.0`，Go 記在執行檔裡的是另一回事：commit 是 tag 之後的那一個，`vcs.modified=true`，主模組版本是 `v1.4.1-0…+dirty`。這個執行檔是 v1.4.0 之後的程式碼再加沒提交的修改，`-X` 填的字串沒有任何東西檢查。值班的人看到 `v1.4.0`，會去翻 v1.4.0 的原始碼找問題，翻錯了版本。

## 較好的做法

`go build` 在 git 工作目錄裡建置時（Go 1.18 起）會把 `vcs.revision`、`vcs.time`、`vcs.modified` 記進執行檔；Go 1.24 起，主模組版本也由 tag 與 commit 推出（見 [Go 1.24 發行說明](https://go.dev/doc/go1.24#go-command)）：剛好在 tag 上而且乾淨是 `v1.4.0`，否則是上面那種偽版本，有未提交的修改再加 `+dirty`。程式用 `debug.ReadBuildInfo` 讀，不用人記得更新任何東西。`go version -m` 看的就是同一份資訊，第 7 章〈[PGO](../chapter7/07_pgo.md)〉提過它會多一行 `-pgo=`。

整理成 `Info`。`FromBuildInfo` 收 `*debug.BuildInfo`，不在函式裡自己呼叫 `ReadBuildInfo`：讀的事交給呼叫端，測試才能傳假的。它與 `Info` 是匯出的，因為使用它們的 `main` 在另一個套件；一行格式（`String`）與 JSON 端點（`Handler`）也在這裡：

```go,ignore
{{#include ../../../examples/chapter12/01_build_version_info/after/version/version.go:info}}
```

`main` 讀一次、整理成 `Info`，給 `-version`、`/version` 端點與啟動日誌用。`-X` 只剩一個用途：放建置系統才知道的值，這裡是 CI 的執行編號：

```go,ignore
{{#include ../../../examples/chapter12/01_build_version_info/after/main.go:build}}
```

```go,ignore
{{#include ../../../examples/chapter12/01_build_version_info/after/main.go:main}}
```

同樣的暫存 repo 狀態，改建 `after/`。模組名稱取成與 `after/` 一樣的 import path，`main.go` 的 import 才不必改：

執行結果（某一次）：

```text
$ mkdemo $repo/examples/chapter12/01_build_version_info/after github.com/holmeslin/learning-go-to-senior/examples/chapter12/01_build_version_info/after
$ go build -ldflags "-X main.build=ci-1234" -o ../bin/versiond .
$ ../bin/versiond -version
v1.4.1-0.20261030054350-7538099cf19c+dirty 7538099cf19c-dirty 2026-10-30T05:43:50Z go1.27.0 build ci-1234
```

`go test` 的測試執行檔沒有 VCS 資訊（見〈取捨〉），所以測試不去讀真的 `ReadBuildInfo`，而是用假的 `BuildInfo`。測試只用 `version` 的公開 API：

```go,ignore
{{#include ../../../examples/chapter12/01_build_version_info/after/version/version_test.go:fake}}
```

```go,ignore
{{#include ../../../examples/chapter12/01_build_version_info/after/version/version_test.go:format}}
```

`dirty` 與 `no vcs info` 兩列就是上面兩種建置的樣子；`Example` 同時看一行輸出與端點回的 JSON：

```go,ignore
{{#include ../../../examples/chapter12/01_build_version_info/after/version/example_test.go:example}}
```

## 取捨

**`go run` 與測試執行檔沒有 VCS 資訊。** `go run` 不蓋 VCS 資訊，`-buildvcs=false` 也不蓋；`go test -c` 建出的測試執行檔，主模組版本是 `(devel)`，也沒有 `vcs.*`。在 `after/` 的暫存 repo 裡：

執行結果（某一次）：

```text
$ go run . -version
(devel) (no vcs info) go1.27.0
$ go build -buildvcs=false -o ../bin/versiond . && ../bin/versiond -version
(devel) (no vcs info) go1.27.0
$ go test -c -o ../bin/version.test ./version && go version -m ../bin/version.test | grep -e vcs -e $'\tmod'
	mod	github.com/holmeslin/learning-go-to-senior/examples/chapter12/01_build_version_info/after	(devel)
```

**`-buildvcs` 預設是 `auto`。** 主套件、主模組與目前目錄都在同一個 repo 才會蓋。建置時沒有 `.git`（例如容器的建置脈絡沒帶，本章〈容器映像（multi-stage）〉會談）、或不是 git，就沒有資訊；這時 `-X` 從建置系統帶 commit 是退路。`go install 模組@v1.4.0` 依[模組參考](https://go.dev/ref/mod#go-install)是從模組版本建置，不看目前目錄；主模組版本記成 `v1.4.0`，建置不在 VCS 目錄裡，所以沒有 `vcs.*`，本章〈發布流程與版本號〉會談。

**`-X` 的限制。** [`cmd/link` 的文件](https://pkg.go.dev/cmd/link)說它只對字串變數有效，初始值只能是常數字串。文件沒寫的兩種情況：`const` 設了沒有作用；`int`、`bool` 讓連結失敗，`-ldflags "-X main.n=1"` 設 `var n int` 得到 `main.n: cannot set with -X: not a var of type string (type:int)`。路徑要寫完整：`main` 套件寫 `main.version`，其他套件寫完整 import path；路徑對不上不會報錯，只是沒作用。回到 `before/` 的暫存 repo 試：

執行結果（某一次）：

```text
$ mkdemo $repo/examples/chapter12/01_build_version_info/before example.com/versiond
$ go build -ldflags "-X example.com/versiond.version=v1.4.0" -o ../bin/wrong .
$ ../bin/wrong -version
dev
```

所以 `-X` 的值要用一次真的建置確認，也不要拿它放時間戳（`$(date)`）：每次建置的結果都不同，本章〈可重現的建置〉會談。它適合放只有建置系統才知道、不在 VCS 裡的值，例如 CI 的執行編號、發布通道（`release` 或 `nightly`）；給使用者看的版本號（命令列工具的 `1.4.0`）由人在發布時決定，tag 就是它的來源。

before 在什麼情況下是對的？建置不在 VCS 目錄（原始碼 tarball），或版本號本來就是人在發布時決定、又由建置指令從 tag 帶入（`-X main.version=$(git describe --tags)`），單一來源，不會對不上。

## 重點整理

- 版本資訊從執行檔自己來：`go build` 記下 `vcs.revision`、`vcs.time`、`vcs.modified` 與主模組版本，用 `debug.ReadBuildInfo` 讀，`go version -m` 也看得到。
- 寫死的版本字串或手動維護的 `-X`，只是一串沒人檢查的字，會和實際建置的 commit 對不上。
- `-X` 只留給建置系統才知道的值；它只能設字串變數，路徑對不上不報錯。
- `go run`、`-buildvcs=false`、沒有 `.git` 的建置脈絡與測試執行檔都沒有 VCS 資訊；格式化邏輯收 `*debug.BuildInfo`，測試用假的，不斷言真實的建置內容。
