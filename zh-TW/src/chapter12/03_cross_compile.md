# 交叉編譯

## 本集目標

判斷程式能不能在一台機器上建出所有目標平台：用 `CGO_ENABLED=0` 與 `GOOS`／`GOARCH` 建置，只有某些平台才有的程式碼用 build tag 隔開，並在 CI 用 `GOOS=… go vet` 提早發現編不過的平台。

## 情境

`probe` 是一支小工具，印出主機名稱與行程能同時開啟的檔案數。開發機是 macOS（arm64），部署在 Linux 的 amd64 與 arm64，偶爾有人在 Windows 上跑。希望在一台機器上用一串指令建出全部。

cgo 與 `CGO_ENABLED` 的預設值前作〈[cgo 簡介](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter11/09_cgo.html)〉教過，build tags 與 `GOOS` 檔名規則見〈[build tags](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter11/06_build_tags.html)〉。這集談的是判斷：什麼會讓交叉編譯壞掉、壞的時候長什麼樣子、怎麼提早發現。把執行檔放進容器，本章〈容器映像（multi-stage）〉會談。

## 常見寫法

主機名稱呼叫 C 的 `gethostname`；檔案數上限直接呼叫 `syscall.Getrlimit`，檔案沒有任何 build tag：

```go,ignore
{{#include ../../../examples/chapter12/03_cross_compile/before/hostname.go:hostname}}
```

```go,ignore
{{#include ../../../examples/chapter12/03_cross_compile/before/limit.go:limit}}
```

這是刻意示範的寫法，完整程式碼在 `examples/chapter12/03_cross_compile/before/`，不要照用。在 macOS 上 `go run` 一切正常。

## 問題在哪

以下輸出都在 macOS（arm64）、Go 1.27.0 上、從 repo 根目錄取得，`go build` 失敗時結束碼是 1。先建 Linux（amd64）：

執行結果（某一次）：

```text
$ GOOS=linux GOARCH=amd64 go build -o /dev/null ./examples/chapter12/03_cross_compile/before
# github.com/holmeslin/learning-go-to-senior/examples/chapter12/03_cross_compile/before
examples/chapter12/03_cross_compile/before/main.go:13:36: undefined: hostname
```

錯誤裡沒有「cgo」這個字：交叉編譯時 `CGO_ENABLED` 預設是 0，`import "C"` 的 `hostname.go` 整個被排除，於是 `hostname` 沒有定義。本機編譯時 cgo 是開的，所以這個問題只會在別的機器或發布時才出現。硬把 cgo 打開也沒用：

執行結果（某一次）：

```text
$ CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o /dev/null ./examples/chapter12/03_cross_compile/before
# runtime/cgo
gcc_clearenv.c:16:2: error: implicit declaration of function 'clearenv' [-Werror,-Wimplicit-function-declaration]
# runtime/cgo
gcc_amd64.S:27:8: error: unknown token in expression
```

輸出很長，上面只留兩個 `# runtime/cgo` 區塊各自的第一行錯誤，其餘的 `gcc_amd64.S` 組譯錯誤與 `note:` 都省略，區塊出現的順序每次也可能不同。壞的是 Go 自己的 `runtime/cgo`：本機的 clang 以 macOS（arm64）為目標，拿 macOS 的標頭去編 Linux（amd64）的 C 與組語。要建 Linux 的 cgo 程式，得另外準備目標平台的 C 編譯器（用 `CC` 指定）與函式庫。

換成 Windows 的錯誤是另一種：

執行結果（某一次）：

```text
$ GOOS=windows GOARCH=amd64 go build -o /dev/null ./examples/chapter12/03_cross_compile/before
# github.com/holmeslin/learning-go-to-senior/examples/chapter12/03_cross_compile/before
examples/chapter12/03_cross_compile/before/limit.go:10:18: undefined: syscall.Rlimit
examples/chapter12/03_cross_compile/before/limit.go:11:20: undefined: syscall.Getrlimit
examples/chapter12/03_cross_compile/before/limit.go:11:38: undefined: syscall.RLIMIT_NOFILE
examples/chapter12/03_cross_compile/before/main.go:13:36: undefined: hostname
```

`limit.go` 沒有 tag，每個平台都編，但 Windows 的 `syscall` 沒有 `Getrlimit`。兩種壞法的共通點是：在開發機上編得過，到了別的平台才發現。

## 較好的做法

先把平台差異隔開。主機名稱改用標準庫的 `os.Hostname`，不經過 cgo；檔案數上限一個平台一份實作，用 build tag 互補，每個平台恰好編進其中一份：

```go,ignore
{{#include ../../../examples/chapter12/03_cross_compile/after/probe/limit_unix.go:unix}}
```

```go,ignore
{{#include ../../../examples/chapter12/03_cross_compile/after/probe/limit_other.go:other}}
```

`//go:build unix` 涵蓋 Linux、macOS 與各種 BSD。檔名規則只認 `GOOS` 與 `GOARCH` 的名字，`unix` 不是 `GOOS`，所以 `limit_unix.go` 的檔名不會自動帶條件，tag 一定要寫。呼叫端不必知道平台，沒有上限的平台回 `errors.ErrUnsupported`，由 `Collect` 判斷這不算錯誤：

```go,ignore
{{#include ../../../examples/chapter12/03_cross_compile/after/probe/probe.go:collect}}
```

測試也跟著平台分檔，放在外部測試套件，只用匯出的 `FileLimit` 與 `Collect`；`FileLimit` 本來就是這個套件的 API，不是為了測試才匯出。本機平台跑自己那一份：

```go,ignore
{{#include ../../../examples/chapter12/03_cross_compile/after/probe/limit_unix_test.go:test}}
```

格式化與平台無關，用 `Example` 鎖住：

```go,ignore
{{#include ../../../examples/chapter12/03_cross_compile/after/probe/probe_test.go:example}}
```

建置時關掉 cgo，逐一指定平台，並用 `file` 確認產出。以下輸出在同一台機器上取得；`file` 的 Linux 兩行省略了 `Go BuildID=…`、`BuildID[sha1]=…` 與結尾的 `with debug_info, not stripped`，也省略了暫存目錄的路徑：

```bash
out=$(mktemp -d)
for t in linux/amd64 linux/arm64 darwin/arm64 windows/amd64; do
	CGO_ENABLED=0 GOOS=${t%/*} GOARCH=${t#*/} go build -o "$out/probe-${t%/*}-${t#*/}" ./examples/chapter12/03_cross_compile/after/cmd/probe
done
(cd "$out" && file probe-*)
```

執行結果（某一次）：

```text
probe-darwin-arm64:  Mach-O 64-bit executable arm64
probe-linux-amd64:   ELF 64-bit LSB executable, x86-64, version 1 (SYSV), statically linked
probe-linux-arm64:   ELF 64-bit LSB executable, ARM aarch64, version 1 (SYSV), statically linked
probe-windows-amd64: PE32+ executable (console) x86-64, for MS Windows
```

四個平台一次建完，不需要任何目標平台的 C 編譯器。CI 要擋的是「某個平台編不過」，所以對每個目標平台跑一次 `go vet`；`go vet` 連測試檔一起型別檢查，另一個平台的 `limit_other_test.go` 也在範圍內：

```bash
for t in linux/amd64 linux/arm64 darwin/arm64 windows/amd64; do
	CGO_ENABLED=0 GOOS=${t%/*} GOARCH=${t#*/} go vet ./examples/chapter12/03_cross_compile/after/...
done
```

四個平台都沒有輸出、結束碼 0。把 `CGO_ENABLED=0` 也放進去，是讓「有東西偷偷依賴 cgo」直接在 CI 變成失敗。整體的 CI 怎麼分工，本章〈`golangci-lint` 與 CI〉會談。

## 取捨

- **純 Go 還是 cgo。** 能用純 Go 的相依，交叉編譯就只剩 `GOOS`／`GOARCH` 兩個變數。本書的資料層用 `modernc.org/sqlite`，就是為了這點：第 4 章〈測試資料層：fake 與真實資料庫〉談過它是純 Go、不必安裝任何東西；可以直接驗證：第 4 章的 `04_pool_size/after` 用 `CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build` 與 `GOOS=windows GOARCH=amd64` 都沒有輸出、結束碼 0。代價是它把 SQLite 的 C 程式轉成 Go，版本與效能和用 C 函式庫的 driver 不一定一致，要換之前自己量。
- **`net` 與 `os/user` 的純 Go 實作。** 沒有 cgo 時這兩個套件改用純 Go：`net` 的解析器自己讀 `/etc/resolv.conf` 發 DNS 請求（本來就是 Unix 上的首選，cgo 只在特定設定下才接手），`os/user` 只解析 `/etc/passwd` 與 `/etc/group`。差別在系統設定用到純 Go 版沒實作的功能時（例如 `nsswitch.conf` 的部分項目，或帳號不在那兩個檔案裡），就查不到。要保留 cgo、只強制這兩個套件用純 Go，可以加 `-tags netgo,osusergo`。
- **cgo 有時是對的。** 沒有純 Go 替代品的 C 函式庫，就得用 cgo；那就在目標平台上建置，或準備交叉 C 編譯器，並接受 CI 要多跑一個環境。cgo 呼叫也占一個執行緒，第 8 章〈阻塞、syscall 與排程〉談過。
- **build tag 的成本。** 每多一種實作就多一份要測的程式碼；`go vet` 只證明編得過，不證明跑得對，`limit_other_test.go` 在 Windows 上才真的執行。能用 `runtime.GOOS` 判斷的差異不必拆檔，但那要求兩邊的程式碼在所有平台都編得過；這裡 `syscall.Getrlimit` 在 Windows 不存在，只能靠 tag。

## 重點整理

- 本機編得過不代表別的平台編得過：交叉編譯時 cgo 預設關閉，含 `import "C"` 的檔案被排除，錯誤只會說某個名稱沒定義。
- 用 `CGO_ENABLED=0` 加 `GOOS`／`GOARCH` 在一台機器建出所有平台；要用 cgo 就得替目標平台準備 C 編譯器。
- 平台才有的程式碼用互補的 build tag（`unix` 與 `!unix`）分檔，呼叫端不分平台；測試跟著 tag 分檔。
- CI 對每個目標平台跑 `CGO_ENABLED=0 GOOS=… GOARCH=… go vet`，提早發現編不過的平台與偷偷依賴 cgo 的相依。
