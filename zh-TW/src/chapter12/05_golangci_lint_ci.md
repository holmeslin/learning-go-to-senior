# `golangci-lint` 與 CI

## 本集目標

判斷 CI 該擋什麼、lint 該開哪些規則：看得懂本書自己的設定，也知道每一條排除的理由。

## 情境

本書的 CI 每次推到 main 都會跑 `checkcode`、`go vet`、`go test -race` 與 benchmark 一次，全部通過才部署。這些步驟一直是綠的，但沒有人檢查「被忽略的錯誤」這一類問題。現在要加 lint，限制有兩個：`before/` 是刻意寫壞的，不能算進去；lint 工具不能把兩百多個間接相依帶進根目錄的 `go.mod`。`gofmt` 與 `go vet` 前作教過：〈[`gofmt` 與 `go vet`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/08_gofmt_vet.html)〉。

下面的輸出來自 macOS（arm64），Go 1.27.0、golangci-lint 2.15.0，都在 repo 根目錄執行。

## 常見寫法

兩種。一是 CI 只跑 `go test`。二是想一次抓乾淨，把規則全開，設定存成 `all.yml`：

```yaml
version: "2"
linters:
  default: all
  exclusions:
    paths:
      - /before/
```

## 問題在哪

只跑 `go test` 的 CI 對這棵樹是綠的。同一棵樹（本集修改之前的樣子）換成 golangci-lint 的預設規則（`errcheck`、`govet`、`ineffassign`、`staticcheck`、`unused`），設定 `min.yml` 就是上面那份拿掉 `default: all` 那一行。預設每個 linter 最多回報 50 筆、同樣的訊息最多 3 筆，不放開的話第一次只看到 56 筆，所以加上兩個旗標，只留最後 5 行，省略前面逐筆的訊息：

```bash
go tool -modfile=tools/lint/go.mod golangci-lint run -c min.yml --max-issues-per-linter=0 --max-same-issues=0 ./... | tail -5
```

執行結果（某一次）：

```text
138 issues:
* errcheck: 132
* ineffassign: 2
* staticcheck: 3
* unused: 1
```

相反的做法是全開。省略其餘 41 個 linter 的行，只留最大的六個：

```bash
go tool -modfile=tools/lint/go.mod golangci-lint run -c all.yml --max-issues-per-linter=0 --max-same-issues=0 ./... | grep -E '^[0-9]+ issues|^\* (depguard|exhaustruct|nlreturn|paralleltest|varnamelen|wsl):'
```

執行結果（某一次）：

```text
4311 issues:
* depguard: 312
* exhaustruct: 195
* nlreturn: 470
* paralleltest: 335
* varnamelen: 327
* wsl: 1350
```

多數是風格偏好，而且和本書的寫法衝突：`depguard` 預設不許 import 專案自己的套件，`exhaustruct` 要求 struct literal 填滿所有欄位，`paralleltest` 要求每個測試呼叫 `t.Parallel`，但用 `t.Setenv` 的測試不能平行，`varnamelen` 嫌 `w` 太短，`wsl` 與 `nlreturn` 管空行。四千多筆沒有人讀得完，下場通常是整個關掉，或是到處補 `//nolint`。

## 較好的做法

從預設的規則集合開始，不加規則；每一條排除都對應一種「看了也沒有人會改」的情況。完整的 `.golangci.yml`：

```yaml
{{#include ../../../.golangci.yml}}
```

- **`before/`。** 刻意寫壞的示範，整個排除。
- **`errcheck` 的函式排除。** 寫進 `http.ResponseWriter` 失敗代表對方斷線，伺服器沒有別的事可做；`bufio.Writer` 記住第一個錯誤，由 `Flush` 回報（第 3 章〈[接受介面、回傳 struct](../chapter3/02_accept_interfaces_return_structs.md)〉的 `Logger` 就是這樣設計）；`os.Setenv` 的名稱是常數。
- **`defer` 的清理與 `Encode`。** `defer rows.Close()`、`defer tx.Rollback()` 沒有地方回報錯誤；`json.NewEncoder(w).Encode` 的狀態碼已經送出。
- **測試檔的 `errcheck`。** 讓斷路器跳開、`go get(...)` 在背景占名額，這些呼叫的結果由後面的斷言檢查。
- **`build-tags`。** 沒有 `integration` 時，整合測試檔不會被分析（實測在帶 tag 的檔案加一個未使用的變數，只有帶 tag 才報錯）。這和 `go vet` 要跑兩次是同一個道理。
- **`max-issues-per-linter: 0`。** 不然修掉一批、又冒出被截掉的另一批。

排除之後還剩 10 筆，逐筆處理。`r.Redeem(...)`、`syscall.Write` 與 `s.Server.Close()` 這四處忽略回傳值，改成 `_ =`，把「刻意不看」寫進程式；`entry` 結構的 `payload` 欄位只是補到 128 位元組，改成 `_ [120]byte`。剩下五筆是示範用的寫法，用 `//nolint` 並寫明理由，例如第 11 章〈[秘密管理](../chapter11/05_secret_management.md)〉故意用錯的動詞：

```go,ignore
{{#include ../../../examples/chapter11/05_secret_management/after/settings_test.go:gaps}}
```

這一批沒有哪一筆會讓範例答錯。每個排除都是在說「這個寫法在本書是對的」，要負責的是設定檔，不是 `//nolint` 的數量。修正後同一個指令回報（結束碼 0）：

```bash
go tool -modfile=tools/lint/go.mod golangci-lint run ./...
```

執行結果（某一次）：

```text
0 issues.
```

工具的版本鎖在 `tools/lint/go.mod`，用 `tool` 指令；它是獨立的模組，根目錄 `./...` 不會包含它，根目錄的 `go.mod` 也不會多出相依。`tool` 指令本身，本章〈`go.mod` 與相依升級〉會談：

```gomod
{{#include ../../../tools/lint/go.mod:1:5}}
```

CI 的步驟加在所有測試之後、建置書之前，指令和本機完全一樣；因為只有推到 main 才跑，lint 失敗會擋住部署：

```yaml
{{#include ../../../.github/workflows/deploy.yml:lint}}
```

CI 的其他工作怎麼分工：

| 工作 | 時機 | 理由 |
| --- | --- | --- |
| `go vet ./...` 與 `go vet -tags integration ./...` | 每次 | 有些檔案只在其中一種設定下編譯；測試只跑帶 tag 的那次，見第 5 章〈[整合測試與 build tags](../chapter5/05_integration_tests_build_tags.md)〉 |
| `go test -race -tags integration ./...` | 每次 | 競爭與整合測試都在這一次 |
| `go test -run='^$' -bench=. -benchtime=1x ./...` | 每次 | 只確保 benchmark 跑得完，見第 7 章〈[把效能回歸擋在 CI 之外](../chapter7/08_perf_regression_ci.md)〉 |
| `golangci-lint` | 每次 | 便宜、結果確定 |
| `CGO_ENABLED=0 GOOS=… go vet` | 每次，有多平台才加 | 本章〈[交叉編譯](03_cross_compile.md)〉談過，本書的 CI 不跑 |
| fuzzing | 排程 | 時間越長越有用，不該拖慢每次推送 |
| 覆蓋率 | 每次，只報告 | 不當門檻 |
| `govulncheck` | 排程 | 資料庫天天更新，與這次變更無關 |

下面三項本書的 CI 都不跑，只是起點。排程的工作放在另一個 workflow 檔，只在預設分支上執行：

```yaml
on:
  schedule:
    - cron: "17 3 * * *" # 每天一次，UTC
  workflow_dispatch: {}
```

- **fuzzing。** `go test -run='^$' -fuzz=FuzzRoundTrip -fuzztime=10m ./examples/chapter5/07_fuzz_roundtrip/after`。`-fuzz` 一次只能對一個套件（`go test -fuzz=Fuzz ./...` 回 `cannot use -fuzz flag with multiple packages`），找到的輸入存成 artifact 或開 issue，提交進 `testdata/fuzz` 後就成了每次都跑的回歸測試，見第 5 章〈[什麼時候值得寫 fuzz 測試](../chapter5/07_fuzz_roundtrip.md)〉。
- **覆蓋率。** `go test -coverprofile=cover.out ./...` 之後，把 `go tool cover -func=cover.out | tail -1` 寫進 `$GITHUB_STEP_SUMMARY`，讓 reviewer 看得到；理由見第 5 章〈[覆蓋率的意義與限制](../chapter5/11_coverage_meaning_limits.md)〉。
- **`govulncheck`。** 排程跑 `go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...`，失敗就通知；怎麼讀輸出見第 11 章〈[`govulncheck` 與相依弱點](../chapter11/04_govulncheck.md)〉。

## 取捨

- **規則有維護成本。** 每多一條規則就多一種要看懂的訊息與誤報；升級 golangci-lint 也可能多出新訊息，所以升級單獨一個 PR，先看新增了什麼。
- **排除規則會放過真的問題。** `defer` 那條也放過寫檔案後的 `defer f.Close()`，而寫入的 `Close` 錯誤要回報；本書沒有這種寫法，但你的專案可能有，要靠 review。
- **`//nolint` 一律寫理由、範圍最小。** 要強制可以開 `nolintlint`，本書沒開。
- **lint 看不到的。** 它和 `go vet` 一樣，一次只分析一組 tag 與一個平台；也看不出設計對不對，不取代 code review，第 13 章〈Go 程式碼的 review 檢查清單〉會談。
- **只跑 `go test` 什麼時候是對的？** 小專案、很少有人改的程式，維護一份設定的成本比它抓到的問題還高。全開則適合只跑一次，從中挑出值得開的規則，不要放進 CI。

## 重點整理

- lint 從預設規則開始，每一條排除或 `//nolint` 都要有理由；全開的結果沒有人讀得完。
- 工具版本用獨立模組的 `tool` 指令鎖定，根目錄用 `-modfile` 執行，CI 步驟放在測試之後。
- 每次 CI 跑確定、便宜的檢查；fuzzing 與 `govulncheck` 排程，覆蓋率只報告。
- 帶與不帶 `integration` tag 的檔案都要檢查，所以 `go vet` 跑兩次、lint 帶 tag。
