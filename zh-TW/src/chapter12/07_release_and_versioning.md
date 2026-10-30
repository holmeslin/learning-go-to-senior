# 發布流程與版本號

## 本集目標

判斷版本號怎麼給、發錯了怎麼補救：哪些改動只能升次版本、哪些必須改模組路徑發 v2，以及為什麼發布出去的 tag 只能用 `retract` 撤回，不能刪除或移動。

## 情境

內部的 `slugtool` 套件把文章標題轉成網址用的 slug，已經有幾個服務依賴。v1 的 `Slug(title string) string` 遇到空白的標題會回傳空字串，網址變成 `/posts/`，你想讓它回傳錯誤：`Slug(title string) (string, error)`。版本號與 `/v2` 路徑的規則前作講過：〈[`go mod init` 與 `go.mod`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/01_go_mod.html)〉、〈[`go get` 與相依版本](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/15_go_get.html)〉。API 怎麼演進，第 1 章〈[匯出名稱與 API 穩定性](../chapter1/04_exported_api_stability.md)〉與〈[棄用（deprecation）與相容性](../chapter1/08_deprecation_and_compatibility.md)〉談過；這集談發布這一步。

## 在 repo 外發布

示範都在 repo 外的暫存目錄，tag 只存在暫存的 git repo，不推送到任何遠端。真正的 `proxy.golang.org` 從 VCS 的 tag 取得原始碼；這裡用本機目錄當模組代理（`GOPROXY=file://…`），`publish` 把 tag 打包成代理要的檔案。`GOSUMDB=off` 是因為 `example.com` 不在公開的 checksum 資料庫裡；`GOPATH` 指向暫存目錄，不動平常的模組快取。本機的 Go 要先是 1.27.0，從本書 repo 的根目錄開始（`before/slug.go` 是 v1 的 `Slug`，`after/slug.go` 是 v2 的，兩邊都有測試）：

```bash
E=$PWD/examples/chapter12/07_release_and_versioning
cd "$(mktemp -d)" && DEMO=$PWD
export GOTOOLCHAIN=go1.27.0 GOFLAGS=-modcacherw GOSUMDB=off
export GOPATH=$DEMO/gopath GOPROXY=file://$DEMO/proxy
publish() { # publish <模組路徑> <tag>：把 tag 打包成模組代理的四個檔案
	d=$DEMO/proxy/$1/@v && mkdir -p "$d"
	git archive --format=zip --prefix="$1@$2/" -o "$d/$2.zip" "$2"
	git show "$2:go.mod" >"$d/$2.mod"
	printf '{"Version":"%s","Time":"%s"}\n' "$2" "$(git log -1 --format=%cI "$2")" >"$d/$2.info"
	echo "$2" >>"$d/list"
}
```

發 v1.0.0 並用 `go install …@版本` 驗證。`cmd/slugtool` 只是讓模組有個可以安裝的 `main`，印出自己的建置資訊（本章〈`go build` 與用 `-ldflags` 注入版本〉談過版本資訊從哪來）：

```bash
mkdir slugtool && cd slugtool && git init -q && go mod init example.com/slugtool
cp "$E/before/slug.go" . && cp -r "$E/before/cmd" .
git add . && git commit -qm "first release" && git tag v1.0.0
publish example.com/slugtool v1.0.0
go install example.com/slugtool/cmd/slugtool@v1.0.0
go version -m $GOPATH/bin/slugtool | sed -n 2,3p
```

以下輸出在 macOS（arm64）、Go 1.27.0、git 2.54.0 取得。省略：`go install` 的下載訊息，`go version -m` 只留第 2、3 行，`h1:` 之後的雜湊以 `…` 取代：

執行結果（某一次）：

```text
$ go version -m $GOPATH/bin/slugtool | sed -n 2,3p
	path	example.com/slugtool/cmd/slugtool
	mod	example.com/slugtool	v1.0.0	h1:…
```

## 次版本裡的破壞，用 `retract` 撤回

`Slug` 改簽名，呼叫端的 `s := slugtool.Slug(t)` 會編譯失敗（第 1 章〈匯出名稱與 API 穩定性〉貼過這類錯誤）。假設作者趕著修，把 `after/` 當成 v1.1.0 發出去，一個服務已經升上去了。這時刪 tag 沒有用：官方的模組代理與 checksum 資料庫取過某個版本就會一直記住。補救是發新版本：還原 `Slug`，在新版本的 `go.mod` 加 `retract`（本章〈[`go.mod` 與相依升級](06_go_mod_dependency_upgrade.md)〉談過它的用途）。`go` 讀最新版本的 `go.mod`，所以它要寫在比被撤回版本更新的版本裡：

```bash
cp "$E/after/slug.go" . && git commit -qam "Slug 回傳 error" && git tag v1.1.0
publish example.com/slugtool v1.1.0
mkdir ../app && cd ../app
go mod init example.com/app && go get example.com/slugtool@v1.1.0
cd ../slugtool && cp "$E/before/slug.go" .
printf '\nretract v1.1.0 // 破壞相容的修改誤放進次版本\n' >>go.mod
git commit -qam "revert Slug, retract v1.1.0" && git tag v1.1.1
publish example.com/slugtool v1.1.1
cd ../app
go list -m -u example.com/slugtool
go list -m -versions example.com/slugtool
go list -m -versions -retracted example.com/slugtool
```

執行結果（某一次）：

```text
$ go list -m -u example.com/slugtool
example.com/slugtool v1.1.0 (retracted) [v1.1.1]
$ go list -m -versions example.com/slugtool
example.com/slugtool v1.0.0 v1.1.1
$ go list -m -versions -retracted example.com/slugtool
example.com/slugtool v1.0.0 v1.1.0 v1.1.1
```

被撤回的版本不會列在 `-versions`，也不會被 `@latest` 選中；已經依賴它的服務在 `go list -m -u` 看到 `(retracted)`。撤回只是警告，不會讓建置失敗。

## v2 要改模組路徑

真正的解法是 v2。常見的錯誤是只打 tag、沒改 `go.mod` 的模組路徑：

```bash
cd ../slugtool && cp "$E/after/slug.go" . && git commit -qam "Slug 回傳 error（v2）" && git tag v2.0.0
publish example.com/slugtool v2.0.0
cd ../app
go get example.com/slugtool@v2.0.0
```

輸出省略第三行（`->` 開頭，同一則 unzip 訊息的重複），措辭隨代理而異，這是本機目錄的結果：

執行結果（某一次）：

```text
$ go get example.com/slugtool@v2.0.0
go: downloading example.com/slugtool v2.0.0
go: unzip $DEMO/gopath/pkg/mod/cache/download/example.com/slugtool/@v/v2.0.0.zip: example.com/slugtool@v2.0.0: invalid version: should be v0 or v1, not v2
```

`v2.0.0` 已經發出去，不能重打，修正版只能是 v2.0.1。`retract` 屬於 v1 路徑的模組，v2 的 `go.mod` 用 `-dropretract` 拿掉；套件互相 import 的路徑也要改成 `/v2`（這個範例沒有）。沒有 `go.mod` 的舊 repo 才會得到 `v2.0.0+incompatible`，那是模組出現之前的規則，新模組別靠它。v1 與 v2 是兩個模組（`go list -m all` 會同時列出 v1.1.0 與 v2.0.1），可以同時在一個建置裡，下游能分批遷移：

```bash
cd ../slugtool && go mod edit -module example.com/slugtool/v2 -dropretract v1.1.0
git commit -qam "module path /v2" && git tag v2.0.1
publish example.com/slugtool/v2 v2.0.1
cd ../app
go get example.com/slugtool/v2@v2.0.1
go list -m all
```

## 不要移動已發布的 tag

最後一種誘惑：把 `v1.1.0` 移到修好的 commit。清掉模組快取，模擬另一台沒下載過的機器：

```bash
cd ../slugtool && git tag -f v1.1.0 v1.1.1
publish example.com/slugtool v1.1.0
cd ../app && rm -rf "$GOPATH/pkg/mod"
go mod download example.com/slugtool@v1.1.0
```

輸出省略空行與 `SECURITY ERROR` 起的說明，雜湊以 `…` 取代：

執行結果（某一次）：

```text
$ go mod download example.com/slugtool@v1.1.0
verifying example.com/slugtool@v1.1.0/go.mod: checksum mismatch
	downloaded: h1:…
	go.sum:     h1:…
```

擋下來的是已經用過 v1.1.0 的服務自己的 `go.sum`。公開的 `sum.golang.org` 對所有人做同一件事：一個版本只能有一個雜湊，所以 tag 只能往前加。

## 取捨

- **v0 還是 v1。** v0 不承諾相容，API 還在改、只有少數呼叫端時留在 v0 比較誠實；多個團隊依賴、想承諾「次版本不破壞」才發 v1。
- **v2 的成本。** 新路徑、一段時間要維護兩個版本，呼叫端要改 import 與程式碼。能新增或棄用解決就不開 v2（第 1 章〈棄用（deprecation）與相容性〉）；同名函式非改簽名不可才要，另一條路是在 v1 新增 `SlugStrict`。
- **自動化。** 版本號靠人判斷。changelog 與 release 工具（GoReleaser、release-please）只負責紀錄與打包；`gorelease` 比對公開 API 的差異，改動算不算破壞，最後還是人決定。

## 重點整理

- 公開 API 的破壞性修改要改模組路徑發新主版本；只升次版本，呼叫端升級時才發現壞了。
- 代理與 checksum 資料庫永久記住發布過的版本：不刪、不移動、不重用 tag，發錯了發新版本。
- `retract` 寫在更新的版本的 `go.mod`，只是警告；v2 要同時改 `go.mod` 的模組路徑與 tag。
- 用 `go list -m -versions` 與 `go install …@版本` 在乾淨環境驗證發布結果。
