# 測時間與並行：用 `testing/synctest`，還是自己注入時鐘

## 本集目標

遇到逾時、重試、到期這類依賴時間的程式，判斷測試該用 `testing/synctest`，還是把時鐘注入程式。

## 情境

服務要呼叫一個偶爾很慢的外部元件，所以包了一個 `Do`：每次嘗試最多 30 秒，失敗就退避 1 秒、2 秒再試，最多三次，呼叫端的 `ctx` 結束時要立刻放棄。`testing/synctest` 的用法前作教過：〈[`testing/synctest`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter9/19_synctest.html)〉，這集只談什麼時候用它、什麼時候不用。

```go,ignore
{{#include ../../../examples/chapter5/08_synctest_or_clock/before/retry.go:do}}
```

## 常見寫法

直接用 `time` 套件，測試只寫跑得快的路徑：

```go,ignore
{{#include ../../../examples/chapter5/08_synctest_or_clock/before/retry_test.go:test}}
```

用 `Example` 看一下全部失敗時的錯誤訊息（`Backoff` 設成 0，所以不用等）：

```go,ignore
{{#include ../../../examples/chapter5/08_synctest_or_clock/before/example_test.go:example}}
```

## 問題在哪

最慢的路徑沒有測試。三次嘗試都逾時，要 30 + 1 + 30 + 2 + 30 = 93 秒；沒人願意讓 `go test ./...` 多等一分半，於是 `Sleep` 之後的行為全憑讀程式確認。

這不是理論上的風險。把等待那段 `select` 換成 `time.Sleep(p.Backoff << (i - 1))`：`ctx` 只剩 10 秒，卻要等滿退避時間，「立刻放棄」的承諾就破了。把這個修改套用到 `before/retry.go` 再執行 `go test`，before 的測試照過，因為它們都不經過等待那一段（輸出省略了套件後面的耗時）。

執行結果：

```text
ok  	github.com/holmeslin/learning-go-to-senior/examples/chapter5/08_synctest_or_clock/before
```

被測到的只有「第一次就成功」與「只試一次」，兩個都不碰時間。要補測試又不想真的等，就得選一種做法。

## 較好的做法

做法一：`synctest`，`Do` 一個字都不用改。bubble 裡的時間只在所有 goroutine 都 durably blocked 時才往前跳，所以 93 秒瞬間跑完，而且可以斷言精確的經過時間，不是「大概不超過」：

```go,ignore
{{#include ../../../examples/chapter5/08_synctest_or_clock/after/retry_test.go:timeout}}
```

要「走到某個時間點，看程式停在哪裡」，用前作〈[`testing/synctest`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter9/19_synctest.html)〉教過的 `synctest.Sleep`：

```go,ignore
{{#include ../../../examples/chapter5/08_synctest_or_clock/after/retry_test.go:backoff}}
```

「`ctx` 結束就放棄」那條，正是剛才的 `time.Sleep` 改法會破掉的：

```go,ignore
{{#include ../../../examples/chapter5/08_synctest_or_clock/after/retry_test.go:cancel}}
```

把 `after/retry.go` 的 `select` 換成 `time.Sleep`，只有這個測試失敗，而且說得出差在哪（輸出省略了每個測試與套件後面的耗時）。

執行結果：

```text
--- FAIL: TestDoStopsWhenContextEnds
    retry_test.go:92: Do() = after 3 attempts: not ready, want context.DeadlineExceeded
    retry_test.go:95: Do() returned after 3m0s, want 10s
FAIL
FAIL	github.com/holmeslin/learning-go-to-senior/examples/chapter5/08_synctest_or_clock/after
FAIL
```

**程式在等網路 I/O。** 依 `testing/synctest` 文件，阻塞在 socket 讀寫上的 goroutine 不算 durably blocked，因為它可能被 bubble 外的事件叫醒。所以在 bubble 裡對真的 loopback 連線發請求，bubble 不會閒置，時間也不會因為它在等而往前跳。文件建議改用 `net.Pipe`：它在行程內，兩端都是 bubble 裡的 goroutine，可以被 durably block，連線的 deadline 也走 bubble 的時鐘：

```go,ignore
{{#include ../../../examples/chapter5/08_synctest_or_clock/after/retry_test.go:pipe}}
```

做法二：注入時鐘。時間本身是業務資料時，例如「試用到 12/31 當天結束」，問題不是等，而是「現在是哪一刻」。bubble 的時鐘固定從 2000-01-01 開始、只會往前走；要測 2026 年底與時區邊界，得先睡過二十幾年，也沒辦法把時鐘撥回去。把 `Now` 注入，每個案例直接指定那一刻（建構函式注入的做法見第 3 章〈[不用框架的相依注入](../chapter3/07_di_without_framework.md)〉）：

```go,ignore
{{#include ../../../examples/chapter5/08_synctest_or_clock/after/trial/trial.go:trial}}
```

```go,ignore
{{#include ../../../examples/chapter5/08_synctest_or_clock/after/trial/trial_test.go:trial}}
```

每個案例各自帶一個 `Now`，沒有共用的可變狀態。完整程式碼在 `examples/chapter5/08_synctest_or_clock/after/`，這一段放在自己的 `trial` 套件。

## 取捨

`synctest`：

- 不用改程式。`Do` 的簽章與實作 before 與 after 完全一樣，已經寫好的程式也能補測試。
- 只適用於行程內、會 durably block 的程式。等真的 socket、`sync.Mutex`、系統呼叫都不算；連到別的行程或外部服務的程式，要先換成 `net.Pipe` 這類假連線。
- 依賴 runtime 的特殊行為：bubble 有自己的限制（見前作〈[`testing/synctest`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter9/19_synctest.html)〉），`synctest.Test` 要等 bubble 裡的 goroutine 全部結束才回來，卡住就失敗（怎麼抓洩漏，第 6 章〈[goroutine 洩漏怎麼發生、怎麼在測試裡抓到](../chapter6/09_goroutine_leak.md)〉）。

注入時鐘：

- 要改 API：多一個欄位或參數，正式程式多一個永遠填 `time.Now` 的預設值。
- 測試不依賴 runtime 的特殊行為，任何時間點都能指定，也能倒退。
- 程式要等真的網路 I/O 時比較合適：測試用得起真的 loopback，而「現在幾點」由測試決定。代價是你只控制得了自己讀的那個時鐘，`time.Sleep` 與計時器要另外抽成注入的 `sleep`，不然仍然是真的在等。

反過來，若逾時只是 `ctx` 的 deadline，而 `fn` 是純函式或 `net.Pipe`，加一個 `Clock` 介面只是多一層，`synctest` 比較省事。依賴時間長短而時好時壞的測試，本章〈不穩定（flaky）測試〉會談。

## 重點整理

- 逾時與重試的路徑因為「要等」而沒有測試，改壞了也沒人知道；先把這條路徑測起來，再談工具。
- `synctest.Test` 不必改程式：時間只在全部 goroutine 都 durably blocked 時前進，所以能精確斷言經過的時間。
- 等 socket 的 goroutine 不算 durably blocked；bubble 裡用 `net.Pipe` 這類行程內的假連線，不用 loopback。
- 時間是業務資料（到期日、跨日、時區），或程式要等真的網路 I/O 時，注入時鐘比較合適。
- `synctest` 與注入時鐘各有代價：前者限制程式的形狀，後者改 API；依情境選。
