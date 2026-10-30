# bulkhead

## 本集目標

服務有好幾個下游時，判斷要不要依下游分開並行上限與 HTTP client，讓一個下游卡住時，其他下游的功能仍然能完成。

## 情境

商品頁的服務，每個請求要呼叫兩個下游：搜尋（`search`）偶爾很慢，價格（`pricing`）一向很快，兩個功能互不相干。為了不讓 goroutine 與連線無限增長，服務用一組固定大小的 worker 控制對外的並行請求（範例用 2 個，真實的數字會大得多）。

HTTP client 與 `httptest` 前作教過：〈[HTTP client](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter14/06_http_client.html)〉、〈[`httptest`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter14/09_httptest.html)〉。一次批次的並行上限，第 6 章〈[`errgroup` 的並行上限](../chapter6/03_errgroup_limit.md)〉談過；這集談的是下游不只一個時，上限怎麼分。

## 常見寫法

所有下游共用同一組 worker 與同一個 client：

```go,ignore
{{#include ../../../examples/chapter10/08_bulkhead/before/gateway.go:gateway}}
```

只有一份資源，設定簡單，下游都健康時也沒有任何毛病。完整程式碼在 `examples/chapter10/08_bulkhead/before/`。

## 問題在哪

測試用函式充當 `http.RoundTripper`，不開 port，也不用 `time.Sleep`。`blocked` 是卡住的下游，只有 `release` 關閉或請求的 context 結束才會回來：

```go,ignore
{{#include ../../../examples/chapter10/08_bulkhead/before/gateway_test.go:stubs}}
```

時間用 `testing/synctest`（用法見第 5 章〈[測時間與並行：用 `testing/synctest`，還是自己注入時鐘](../chapter5/08_synctest_or_clock.md)〉）。`synctest.Wait` 等到 bubble 裡所有 goroutine 都卡住，這時 `search` 的兩個請求已經占滿兩個 worker。bubble 裡的時間是假的，經過的時間是確定的，所以可以斷言恰好 1 秒：

```go,ignore
{{#include ../../../examples/chapter10/08_bulkhead/before/gateway_test.go:starved}}
```

`pricing` 自己沒有任何問題，它的下游一次都沒被呼叫到，請求卻在呼叫端的 deadline 到了才失敗。壞的是 `search`，受害的是 `pricing`。呼叫端沒設 deadline 的話，這些請求會一直等到 `search` 恢復。共用的若是沒有依下游分開的連線池（例如兩個功能共用同一個資料庫的連線池），結構也一樣：資源不分，一個下游的故障就是整個服務的故障。

## 較好的做法

每個下游有自己的名額與自己的 client：

```go,ignore
{{#include ../../../examples/chapter10/08_bulkhead/after/gateway.go:gateway}}
```

`New` 為每個下游建立容量為 `Limit` 的名額，`Limit` 小於 1 會被拒絕（0 會讓請求永遠排不進去）。`Get` 只從自己下游的名額取位置；滿了就等，等待受 `ctx` 限制。`Client` 由呼叫端傳入，每個下游一個，所以逾時、TLS 設定與連線池都能各自調整。建立 client 時要留意閒置連線的上限：

```go,ignore
{{#include ../../../examples/chapter10/08_bulkhead/after/gateway.go:client}}
```

`Transport` 預設每個 host 只留 2 條閒置連線（`http.DefaultMaxIdleConnsPerHost`）。下游的 `Limit` 比 2 大時，尖峰過後多出來的連線被關掉，下一次尖峰又要重新建立。`NewClient` 讓閒置上限跟著 `Limit`。

測試用的 stub 與 before 一模一樣，情境也一樣：`search` 卡住並占滿自己的兩個名額。

```go,ignore
{{#include ../../../examples/chapter10/08_bulkhead/after/gateway_test.go:isolated}}
```

三個斷言各有用途：`pricing` 在假時間 0 秒內完成；`search` 的第三個請求排不進去，下游同時只看到 2 個，它等到 deadline；而這個測試在共用資源的寫法下會失敗。把 `New` 裡所有下游的名額換成同一個 channel，`pricing` 的兩個斷言（結果、經過的時間）都失敗；把 `Limit` 改成 100，`search` 同時進行的數量變成 3，第三個斷言失敗。

用真的 `httptest.Server` 走一遍，確認每個下游走自己的 client：

```go,ignore
{{#include ../../../examples/chapter10/08_bulkhead/after/example_test.go:example}}
```

## 取捨

- **閒置的名額借不出去。** 共用 2 個 worker 時，`search` 沒流量，`pricing` 可以用滿；分開後各 2 個，總共要 4 個，而 `pricing` 的請求再多，也用不到 `search` 閒置的名額。切得越細，每一份都要留尖峰的餘裕，加總就越大。按下游或功能切到夠用就好，不要為每個 endpoint 各切一份。
- **上限怎麼定。** 從該下游的容量與你預期的尖峰流量出發，不是平均分。副本會相乘：4 個副本各設 4，下游看到的是 16，算法和第 6 章〈[`errgroup` 的並行上限](../chapter6/03_errgroup_limit.md)〉的一樣。各下游上限加總，也是這個行程最多同時占用的 goroutine 與連線數。
- **和連線池的關係。** `http.Transport` 的連線池本來就依 host 分開，共用一個 client 並不會讓兩個 host 搶同一批連線；真正共用的是 worker 的名額，以及 `MaxIdleConns` 這類全域設定。限制連線數（`MaxConnsPerHost`）也不等於限制同時請求，HTTP/2 的一條連線會承載很多請求，所以並行上限要自己管。範例的 `Downstream{Client: NewClient(4), Limit: 4}` 把兩個數字設成一樣，是刻意的選擇：`Client` 的閒置連線上限管連線池，`Limit` 管並行名額，是兩個不同的旋鈕，不是同一個數字寫兩次。資料庫也是同樣的道理：`database/sql` 的連線池就是資料庫專屬的 bulkhead，怎麼設見第 4 章〈[連線池要設多大、設錯會怎樣](../chapter4/04_pool_size.md)〉。
- **和斷路器、限流的分工。** bulkhead 限制「同時占用多少」，讓故障留在一個區域；本章〈[斷路器](05_circuit_breaker.md)〉談過的斷路器偵測到下游壞了就停止呼叫，請求不再占名額；本章〈[`golang.org/x/time/rate` 限流](04_rate_limit.md)〉談過的限流管「每秒幾個」。bulkhead 不會讓卡住的下游變好，它的名額滿了之後，那個下游的呼叫端仍然在等；等多久靠 deadline，要不要乾脆拒絕，本章〈[負載卸載與降級](07_load_shedding.md)〉談過。三者常常一起用。
- **名額是靜態的。** 下游變慢或變快，上限都不會自己調整。要依延遲自動調整，要用自適應的並行控制，這裡只提名稱。

before 在這些情況下反而可以接受：下游只有一個，或所有下游其實依賴同一個後端、同生共死，切開也隔離不了什麼；或流量小、所有呼叫都有 deadline，而且沒有已知會變慢的下游。這時多切一份，只是多一組要調的數字。

## 重點整理

- 所有下游共用一組 worker 或連線資源時，一個下游卡住就會占滿它，不相關的功能跟著停擺。
- bulkhead 依下游各給一份並行上限與一個 client，故障只影響那個下游自己的功能。
- 證明隔離要有「共用資源就會失敗」的斷言：用卡住的假下游占滿一個下游，另一個下游仍在假時間 0 秒內完成。
- 代價是閒置的名額：總量是各下游上限的加總，也要乘上副本數；上限從下游的容量與尖峰流量估。
- bulkhead 管同時幾個，不偵測故障也不限速；它和斷路器、限流分工，不互相取代。
