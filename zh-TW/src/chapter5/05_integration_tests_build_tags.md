# 整合測試與 build tags

## 本集目標

判斷需要真實資源的測試要怎麼和單元測試分開；分開之後，知道 CI 要帶什麼參數，才不會讓沒人跑的測試悄悄壞掉。

## 情境

筆記服務的 `notes.Store` 把筆記存在一個 SQLite 檔案裡。`Normalize`（去掉前後空白、拒絕空筆記）是純函式，測試很單純。`Store` 要開真的檔案才測得到，這是整合測試。這裡的 SQLite 只是代替真實專案裡需要的資料庫或外部服務；它是純 Go、不用安裝，範例才能直接跑。

build tags 的機制前作教過：〈[build tags](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter11/06_build_tags.html)〉；`go test` 的基本用法見〈[`go test`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/09_go_test.html)〉。這集只談怎麼用它們分開測試。

## 常見寫法

整合測試就寫在單元測試旁邊，同一個檔案、同一個 `go test`：

```go,ignore
{{#include ../../../examples/chapter5/05_integration_tests_build_tags/before/notes/notes_test.go:tests}}
```

完整程式碼在 `examples/chapter5/05_integration_tests_build_tags/before/`。

## 問題在哪

**第一個問題：每個人的 `go test ./...` 都會跑它。** 光是改了 `Normalize`，也得開檔案、建表。SQLite 在這裡很快，看不出痛處；它代表的那類資源才是問題：每個測試都要連線、建表、清資料，所以慢；沒有那個服務、帳號或網路的機器上，`go test ./...` 一開始就是紅的，所以依賴環境。

於是有人給整合測試加上 `//go:build integration`，搬到自己的檔案。這解決了第一個問題，卻帶來第二個：**預設的 `go vet` 與 `go test` 不編譯有 tag 的檔案，沒人帶 tag 跑，它壞了也沒人知道。**

下面的檔案就是搬出來的整合測試。之後 `Open` 為了接 `context` 多了一個參數，單元測試與呼叫端都跟著改了，這個檔案沒人碰，因為沒有任何指令會編譯它：

```text
//go:build integration

package notes_test

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter5/05_integration_tests_build_tags/before/notes"
)

func TestStoreAddList(t *testing.T) {
	store, err := notes.Open(filepath.Join(t.TempDir(), "notes.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()
	if _, err := store.Add(t.Context(), "buy milk"); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	got, err := store.List(t.Context())
	if err != nil || !slices.Equal(got, []string{"buy milk"}) {
		t.Errorf("List() = %q, %v; want [buy milk]", got, err)
	}
}
```

要自己重現，把這個檔案放進 `before/notes/`，刪掉 `notes_test.go` 裡原本的 `TestStoreAddList`（它搬到了這裡），在 `before/` 目錄執行。預設的指令全部通過（省略了耗時）：

執行結果：

```text
$ go vet ./... && go test ./...
ok  	github.com/holmeslin/learning-go-to-senior/examples/chapter5/05_integration_tests_build_tags/before/notes
```

帶 tag 才看到錯誤（省略了開頭兩行 `#` 開頭的套件標頭）：

編譯錯誤：

```text
$ go vet -tags integration ./...
vet: notes/store_integration_test.go:14:65: not enough arguments in call to notes.Open
	have (string)
	want (context.Context, string)
```

第一個指令全綠，整合測試其實已經壞了，直到有人哪天帶 tag 跑才會發現。編輯器也一樣：除非另外設定 build flags，它不會載入這個檔案。

## 較好的做法

分開之後，補上另一半：**CI 帶 tag 跑 vet 與測試。** 整合測試放在 `store_integration_test.go`，第一行是 `//go:build integration`，後面空一行才寫 `package`；單元測試留在 `notes_test.go`：

```go,ignore
{{#include ../../../examples/chapter5/05_integration_tests_build_tags/after/notes/store_integration_test.go:test}}
```

這個測試放在外部測試套件，每次用 `t.TempDir()` 的新檔案，所以不依賴別的測試、也不依賴任何外部服務；它驗證的是「關掉再開，筆記還在」，只有真的檔案測得出來。完整程式碼在 `examples/chapter5/05_integration_tests_build_tags/after/`。

本機與 CI 的差別，只在 `-tags integration`。在 `examples/chapter5/05_integration_tests_build_tags/` 底下，同一個套件不帶與帶 tag 各列一次會跑的測試（`-list` 只列名稱、不執行；`ok` 那一行省略了耗時與套件路徑前綴）：

執行結果：

```text
$ go test -list . ./after/notes
TestNormalize
TestNormalizeRejectsEmpty
ok  	.../after/notes
$ go test -tags integration -list . ./after/notes
TestNormalize
TestNormalizeRejectsEmpty
TestStorePersistsAcrossReopen
TestStoreRejectsEmptyNote
ok  	.../after/notes
```

本書自己的 CI（`.github/workflows/deploy.yml`）就是這樣跑的，所以書裡每個加了 `integration` tag 的測試都真的執行過：

```bash
go vet ./... && go vet -tags integration ./... && go test -race -tags integration ./...
```

`go vet` 預設與帶 tag 各跑一次，是因為有些檔案只在其中一種設定下才編譯；測試只跑帶 tag 的那一次，它已經包含不帶 tag 的測試。CI 的設定本身，第 12 章〈[`golangci-lint` 與 CI](../chapter12/05_golangci_lint_ci.md)〉談過。

## 取捨

分開測試有三種常見做法：

| 做法 | 預設 | 檔案有沒有被編譯 | 代價 |
| --- | --- | --- | --- |
| build tag | 不跑 | 預設不編譯，`vet` 與編輯器也看不到 | 一定要有人帶 tag 跑，否則就像上面那樣壞掉 |
| 環境變數加 `t.Skip` | 不跑 | 一直編譯，`vet` 看得到 | 要記得設變數；忘了設，測試只是「跳過」，看起來像通過 |
| `testing.Short()` | 全跑 | 一直編譯 | 方向相反：要加 `-short` 才跳過，慢的測試預設還是會跑 |

第二種做法的好處：同一個壞掉的測試，如果改成不加 tag、用 `NOTES_INTEGRATION` 變數決定要不要 `t.Skip`，那個檔案預設就會被編譯，所以 `go vet ./...` 會立刻報同一個編譯錯誤。代價是 CI 還是得設變數；少設一次，整合測試整批變成 `SKIP`，而 `go test` 仍然回報通過。

選擇的依據：

- 要靠 CI 帶設定才會跑的測試，用 tag 把「預設不跑」寫在檔案上，最不容易被誤跑；前提是 CI 帶 tag。
- 想讓編譯錯誤在每個人的 `go vet` 就被抓到，用環境變數加 `t.Skip`。
- 測試只是比較慢、不需要外部資源（例如大量輸入的計算），用 `testing.Short()` 讓想快的人自己加 `-short`。

用 tag 還有一個限制：加了 tag 的檔案裡的輔助函式，沒加 tag 的檔案不能用，否則預設設定下就編譯不過；反過來沒問題。tag 也不是免費的：每多一個 tag，要驗證的組合就多一倍（前作〈build tags〉也提醒過）；本書只用 `integration` 一個。

before 在這些情況下反而是對的：

- 資源便宜、完全在行程內，就不必分開。第 4 章〈[測試資料層：fake 與真實資料庫](../chapter4/07_fake_or_real_database.md)〉的 SQLite 測試沒加 tag：純 Go、不必安裝任何東西，分開反而讓它沒人跑。本集的範例也是 SQLite，只是用它代替「要先起服務的資料庫」；判準是成本與環境，不是「是不是真的資料庫」。
- 整合測試只有一兩個，又沒有 CI 可以帶 tag 時，環境變數加 `t.Skip` 比 tag 容易維持。

整合測試要啟動真的 HTTP server 時，要用 `httptest.NewServer` 還是測 handler，本章〈要啟動真的 server，還是只測 handler〉會談。

## 重點整理

- 需要真實資源、跑得慢的測試要和單元測試分開，讓每個人的 `go test ./...` 又快又不依賴環境。
- 加了 `//go:build integration` 的檔案，預設的 `go vet` 與 `go test` 不會編譯它；沒人帶 tag 跑，它壞了也沒人知道。
- 分開之後，CI 必須帶 `-tags integration` 跑 `go vet` 與 `go test`，tag 才算真的在用。
- build tag、環境變數加 `t.Skip`、`testing.Short()` 各有代價：tag 讓檔案不被編譯，環境變數會把「沒跑」顯示成「跳過」，`-short` 預設全跑。
- 便宜、在行程內的資源不必分開；判準是成本與環境。
