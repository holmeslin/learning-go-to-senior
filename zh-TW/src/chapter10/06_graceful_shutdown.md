# 關機時該等什麼、等多久（graceful shutdown）

## 本集目標

決定關機要依什麼順序、等哪些東西、每一步最多等多久：先讓 readiness 失敗，再等進行中的請求，最後停背景工作，全部加起來不超過部署平台給的寬限期。

## 情境

一個事件收集服務：收進來的請求要寫進儲存才回應，背景另有一個工作，把累積的事件批次送去分析系統。每次部署，平台先送 `SIGTERM`，寬限期（例如 30 秒）過了還沒結束，就用 `SIGKILL` 強制結束。

`Shutdown` 與 `signal.NotifyContext` 前作教過：〈[graceful shutdown](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter14/08_graceful_shutdown.html)〉、〈[`signal.NotifyContext`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter13/07_signal_notify_context.html)〉。這集只談順序與上限。

## 常見寫法

兩種都很常見。一種是 ctx 結束就回傳，讓 `main` 結束行程；另一種是加上 `Shutdown`，只等 HTTP 伺服器：

```go,ignore
{{#include ../../../examples/chapter10/06_graceful_shutdown/before/graceful.go:exit}}
```

```go,ignore
{{#include ../../../examples/chapter10/06_graceful_shutdown/before/graceful.go:shutdown}}
```

## 問題在哪

下面兩個 `Example` 用同一組工具：`/slow` 卡在 `gate` 上，背景工作卡在一筆很久的工作中，兩者都由測試決定何時放行。

```go,ignore
{{#include ../../../examples/chapter10/06_graceful_shutdown/before/example_test.go:helpers}}
```

直接結束時，行程一退出，作業系統就關掉所有連線。測試裡用 `Config.Close` 代替這件事：

```go,ignore
{{#include ../../../examples/chapter10/06_graceful_shutdown/before/example_test.go:exit}}
```

進行中的請求被切斷，使用者拿到的是連線錯誤，不知道事情有沒有做完；背景工作開始了，沒有做完，也沒有人通知它。

加上 `Shutdown` 之後，請求保住了，缺口還在：

```go,ignore
{{#include ../../../examples/chapter10/06_graceful_shutdown/before/example_test.go:shutdown}}
```

`Shutdown` 只認得 `http.Server` 自己的連線，不認得你開的 goroutine，所以 `ShutdownOnly` 一回傳，背景工作還停在半路（誰開 goroutine、誰負責等，第 6 章〈[goroutine 的生命週期歸誰管](../chapter6/01_goroutine_ownership.md)〉談過）。還有兩個程式碼看不出來的問題：

- 沒有順序。Kubernetes 文件〈[Termination of Pods](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-termination)〉寫明，kubelet 開始關閉 Pod 的同時，control plane 也在評估要不要把它從 EndpointSlice 移除，兩件事是並行的；負載平衡器要過一陣子才會發現，這段時間還會有請求進來。readiness 沒有先失敗，新請求會一直進來，`Shutdown` 要等的東西沒有盡頭。
- 上限和平台無關。寫死的 10 秒可能比寬限期長，那就等不到結束，被 `SIGKILL`；也可能短到做不完正常的請求。

## 較好的做法

關機是一串有順序的步驟，每一步有自己的上限，而且全部掛在同一個寬限期底下：

1. 讓 readiness 失敗（第 9 章〈[健康檢查](../chapter9/07_health_check.md)〉的 `/readyz`），等 `Drain`，給負載平衡器時間發現。這段時間仍接受請求。
2. 呼叫 `Shutdown`：停止接受新連線，等進行中的請求，最多等 `Requests`。超過就 `Close`，切斷剩下的連線。
3. 請求都結束了，才通知背景工作停止（請求可能還要用到它），並等它回傳，最多等 `Workers`。
4. 以上每一步的 context 都從 `Grace` 衍生，所以總時間不會超過 `Grace`。

```go,ignore
{{#include ../../../examples/chapter10/06_graceful_shutdown/after/graceful.go:run}}
```

幾個細節：`Server` 由呼叫端建立，逾時等設定留在 `main`；`Run` 收 `net.Listener` 而不是位址，是為了讓測試換成記憶體裡的連線；正式環境傳真的 listener，沒有其他為了測試才開的 API。背景工作的 context 用 `WithoutCancel` 切開，才不會在 `SIGTERM` 當下就停。`Grace` 的計時從收到訊號起算，用的是新的 context，不是已經結束的 `ctx`。`main` 只負責接 signal 與組裝，這一段只放在 `main` 套件（CI 只編譯，不執行）：

```go,ignore
{{#include ../../../examples/chapter10/06_graceful_shutdown/after/demo/main.go:main}}
```

啟動它，在另一個終端機執行 `curl localhost:8080/slow`，回應出來之前按 Ctrl+C。5 秒的 `Drain` 內，`curl -i localhost:8080/readyz` 回 503，`/slow` 照常做完；`Drain` 過後新的連線被拒絕。

測試用 `testing/synctest`，時間是假的、確定的，所以可以斷言精確的經過時間。bubble 裡的 socket 不算 durably blocked，因此改用 `net.Pipe` 做的假網路（原因見第 5 章〈[測時間與並行：用 `testing/synctest`，還是自己注入時鐘](../chapter5/08_synctest_or_clock.md)〉）。測試不送 signal，用取消 context 代表收到訊號：

```go,ignore
{{#include ../../../examples/chapter10/06_graceful_shutdown/after/graceful_test.go:net}}
```

```go,ignore
{{#include ../../../examples/chapter10/06_graceful_shutdown/after/graceful_test.go:launch}}
```

第一個測試依序確認每一步：`Drain` 內 `/readyz` 回 503、新請求仍被接受；`Drain` 之後新連線被拒、進行中的請求還沒結束、背景工作還沒被通知；請求做完才通知背景工作，`Run` 等它收尾才回傳。總時間的斷言是範圍（5 到 6 秒）：`Server.Shutdown` 輪詢閒置連線的間隔從 1 毫秒起每次加倍、上限 500 毫秒，每次還加上最多一成的隨機 jitter，請求結束到被發現之間差多少不固定；其他的經過時間斷言都是精確值。

```go,ignore
{{#include ../../../examples/chapter10/06_graceful_shutdown/after/graceful_test.go:order}}
```

請求卡住不放時，`Run` 在 `Drain` 加 `Requests` 恰好 15 秒後強制結束，那個請求拿到連線錯誤：

```go,ignore
{{#include ../../../examples/chapter10/06_graceful_shutdown/after/graceful_test.go:stuck}}
```

請求與背景工作都卡住時，各步驟的上限加起來有 25 秒，但整段只等到 `Grace` 的 12 秒：

```go,ignore
{{#include ../../../examples/chapter10/06_graceful_shutdown/after/graceful_test.go:grace}}
```

把 `Shutdown` 換成 `Close`、不等背景工作、不讓 readiness 失敗、讓步驟上限不掛在 `Grace` 底下，各有測試會失敗。在真的 socket 上，行為一樣，這個 `Example` 用 `httptest` 提供 listener：

```go,ignore
{{#include ../../../examples/chapter10/06_graceful_shutdown/after/example_test.go:example}}
```

## 取捨

**寬限期不是你決定的。** 它來自部署平台（例如 Kubernetes 的 `terminationGracePeriodSeconds`，預設 30 秒）。`Grace` 要比它短，留時間給行程退出；`Drain` 要比探針的間隔乘上連續失敗次數長。容器與部署的設定，第 12 章〈容器映像（multi-stage）〉會談。

**關機會拉長部署。** 每個實例每次部署都要多等 `Drain` 與最慢的請求。滾動更新的實例越多，總時間越長；上限設得越保守，越浪費。

**`Shutdown` 管不到的東西。** 被 hijack 的連線（例如 WebSocket）與 handler 裡自己 `go` 出去的 goroutine，它都不等。前者用 `RegisterOnShutdown` 通知並自己關閉，後者要有擁有者。`Shutdown` 也不取消請求的 context，超過 `Requests` 才由 `Close` 切斷，所以長時間的 handler 仍要自己設上限。

**佇列裡的工作，做完還是交還。** 範例的 `Work` 在 ctx 結束時回傳。記憶體裡的佇列隨行程消失，所以回傳之前要決定：做完（受 `Workers` 上限限制），或交還給外部的佇列重送。交還表示同一筆工作可能被處理兩次，要冪等，本章〈[冪等性](03_idempotency.md)〉談過。

**用盡上限不是成功。** `Run` 回傳的錯誤代表有請求被切斷或背景工作沒停下來，應該記錄下來，並在部署時留意它的頻率。整套服務的觀測與關機怎麼搭配，第 16 章〈可觀測性與韌性〉會談。

**before 什麼時候夠用？** 請求都很短、可以安全重送，沒有背景工作，平台又會先把流量導開，只用 `Shutdown` 就夠；連背景工作都不需要時，連 `Drain` 都可以省。

## 重點整理

- 關機有順序：readiness 先失敗、`Shutdown` 等請求、請求結束後才停背景工作。
- 每一步有上限，且都從同一個 `Grace` 衍生，總時間不會超過平台的寬限期。
- `Shutdown` 不等 hijack 的連線與自己開的 goroutine，背景工作要有人負責等。
- 超過上限就強制結束，並把它當成要記錄的事件。
- 測試用 context 觸發、`synctest` 的假時間斷言順序與經過的時間，不送 signal。
