# 健康檢查

## 本集目標

設計健康檢查端點時，判斷哪些檢查屬於 liveness、哪些屬於 readiness：下游慢或壞的時候，該讓流量繞開，還是讓行程被重啟。

## 情境

一個對外的 HTTP 服務，前面有負載平衡器與編排平台，會定期打服務的健康端點。端點回得出 200，代表行程還在、可以收流量；連續失敗，平台就把實例移出流量，或直接重啟它。同一個資料庫被所有實例共用。

HTTP handler 與 `httptest` 前作教過：〈[第一個 HTTP 伺服器](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter14/01_first_server.html)〉、〈[`httptest`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter14/09_httptest.html)〉。這集只談端點該檢查什麼，不談探針的設定語法。

## 常見寫法

只開一個 `/health`，裡面 ping 資料庫：能 ping 通就 200，否則 503。liveness 與 readiness 共用它：

```go,ignore
{{#include ../../../examples/chapter9/07_health_check/before/health.go:handler}}
```

資料庫正常時，這個寫法沒有任何毛病，也很好讀。完整程式碼在 `examples/chapter9/07_health_check/before/`。

## 問題在哪

`db.Ping()` 沒有逾時（`*sql.DB` 的 `Ping` 用的是 `context.Background()`）。資料庫一慢，`/health` 就跟著卡住，而不是回 503：

```go,ignore
{{#include ../../../examples/chapter9/07_health_check/before/example_test.go:example}}
```

`stuckDB` 的 `Ping` 用 channel 真的阻塞，沒有 `time.Sleep`：請求進到 `Ping` 之後，只有 `release` 能讓它回來，所以資料庫卡住的期間不會有任何回應。探針那一側等不到回應，就當作失敗；每打一次，服務端多卡一個 goroutine，這些 goroutine 的去向，第 6 章〈[goroutine 洩漏怎麼發生、怎麼在測試裡抓到](../chapter6/09_goroutine_leak.md)〉談過。

接著是連鎖反應。所有實例共用同一個資料庫，也共用同一個 `/health`：資料庫變慢的那一刻，每個實例的探針一起失敗，一起被當成死掉的行程而重啟。重啟本身救不了資料庫，剛起來的實例還會一起重建連線，反而讓資料庫更慢。一個下游的問題，被放大成整個服務的中斷。

## 較好的做法

把兩個問題拆開問。`/livez` 只回答「行程還活著嗎」，不碰任何下游，答案是 200；`/readyz` 回答「現在能不能接流量」，才去檢查下游。實例暫時不 ready，只是被移出流量，不會被重啟。

`/readyz` 的每一項檢查有自己的名稱與逾時，同時執行，結果彙整成一份回應：

```go,ignore
{{#include ../../../examples/chapter9/07_health_check/after/health.go:ready}}
```

`run` 另外開 goroutine 執行 `Probe`，自己等結果或 `ctx` 結束。這樣就算 `Probe` 忘了看 `ctx`，`/readyz` 也會在逾時回 503，不會跟著卡住。結果 channel 有緩衝，`Probe` 晚點才回來也不會卡在送結果。

測試用假的檢查，下面這三個 helper 會在每個測試與 `Example` 裡用到：

```go,ignore
{{#include ../../../examples/chapter9/07_health_check/after/health_test.go:checks}}
```

時間用 `testing/synctest`（用法見第 5 章〈[測時間與並行：用 `testing/synctest`，還是自己注入時鐘](../chapter5/08_synctest_or_clock.md)〉），逾時在 bubble 裡瞬間跑完，經過的時間是精確的。下游失敗與逾時時，`/readyz` 回 503 並列出失敗的檢查；三項檢查是同時跑的，所以經過的時間是最慢的失敗項 2 秒，不是加總：

```go,ignore
{{#include ../../../examples/chapter9/07_health_check/after/health_test.go:readyz}}
```

檢查卡住而且不理 `ctx` 的情況，也要有測試：

```go,ignore
{{#include ../../../examples/chapter9/07_health_check/after/health_test.go:stuck}}
```

同樣的下游全壞，`/livez` 仍然回 200，而且在 bubble 裡經過的時間是 0，代表它沒有等任何下游：

```go,ignore
{{#include ../../../examples/chapter9/07_health_check/after/health_test.go:livez}}
```

兩個端點的回應，可以用 `Example` 直接看：

```go,ignore
{{#include ../../../examples/chapter9/07_health_check/after/example_test.go:example}}
```

## 取捨

**readiness 該檢查哪些下游？** 只檢查「沒有它就無法服務」的硬相依，例如每個請求都要用的資料庫。有降級路徑的下游（推薦系統掛了還是能回預設內容）不該讓實例退出流量，否則它一壞，所有實例同時 503，等於整個服務停擺。共用的下游壞掉時，所有實例一起 not ready 是無法避免的；要處理的是有備援或降級的部分。

**liveness 該檢查什麼？** 預設什麼都不檢查。只有行程自己可能陷入「還活著但永遠不會好」的狀態（例如內部的死鎖偵測），而且重啟確實能解決時，才加內部的檢查；下游的狀態不屬於這裡。

**頻率與快取。** 探針的頻率由平台決定，每個實例每次探測都會真的打一次下游。實例多、探針密的時候，健康檢查本身就是下游的一份負載。檢查慢或有成本（例如要借用一條連線池的連線），可以由背景 goroutine 定期檢查、`/readyz` 只回最近一次的結果；代價是狀態會落後一個檢查週期，快取的時效要比探針判斷失敗的時間短。上面的 `run` 對卡住的 `Probe` 會留下 goroutine，檢查頻繁而下游一直卡住時，背景定期檢查（一次只留一個進行中的檢查）能把數量壓在固定的上限。

**逾時要比探針的逾時短。** 每項檢查的逾時要小於探針等待回應的時間，否則探針先放棄，看到的是無回應而不是 503 與失敗清單。整條請求鏈的逾時怎麼分配，第 10 章〈逾時預算〉會談；資料層的版本見第 4 章〈[查詢逾時該設在哪一層](../chapter4/03_query_timeout_layer.md)〉。

**回應內容。** `/readyz` 列出失敗項方便排查，但端點若對外公開，內容會洩漏內部架構；健康端點通常只開給負載平衡器與平台，或只回狀態碼。

**before 什麼時候夠用？** 單一實例、沒有編排平台自動重啟、資料庫是唯一的相依，而且只有人在看這個端點時，單一 `/health` 沒有連鎖反應可言。實例一多、有自動重啟，就該拆開。

關機時 readiness 怎麼配合（先變成 not ready、再停止接受連線），第 10 章〈關機時該等什麼、等多久（graceful shutdown）〉會談。

## 重點整理

- liveness 只回答「行程還活著嗎」，不檢查下游；下游的狀態放在 readiness。
- 同一個端點兼任兩者，下游一慢，所有實例同時被重啟，問題被放大。
- readiness 的每項檢查要有名稱與逾時，同時執行並彙整；失敗時回 503 並列出失敗的檢查。
- 只檢查沒有它就無法服務的下游；探針越密、檢查越貴，越要考慮背景檢查加快取。
- 健康檢查的測試用假的檢查與 `testing/synctest`，逾時與經過的時間都是確定的，不需要 `time.Sleep`。
