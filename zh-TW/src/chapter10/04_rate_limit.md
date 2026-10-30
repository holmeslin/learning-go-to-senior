# `golang.org/x/time/rate` 限流

## 本集目標

需要把速率壓在「每秒幾個」以內時，判斷用 `rate.Limiter` 的 `Wait` 還是 `Allow`，並看出自己用 `time.Sleep` 或計數器寫的限流，在並行與突發流量下哪裡不準。

## 情境

通知服務有兩個方向要限流。往外，它呼叫簡訊供應商的 API，合約寫明每秒最多 5 則（數字是舉例），超過的會被拒絕。往內，它的 HTTP API 開放給很多 client，任何一個 client 狂打，都不該拖累其他人。

`httptest` 與 `testing/synctest` 前作教過：〈[`httptest`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter14/09_httptest.html)〉、〈[`testing/synctest`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter9/19_synctest.html)〉；測時間為什麼選它，見第 5 章〈[測時間與並行：用 `testing/synctest`，還是自己注入時鐘](../chapter5/08_synctest_or_clock.md)〉。這集只談「每秒幾個」。

## 常見寫法

兩種自製的做法：每次呼叫前睡一個間隔，或用每秒歸零的計數器：

```go,ignore
{{#include ../../../examples/chapter10/04_rate_limit/before/limiter.go:limiters}}
```

單一呼叫端慢慢送的時候，這兩個都夠用，也都很短。完整程式碼在 `examples/chapter10/04_rate_limit/before/`。

## 問題在哪

`synctest` 的 bubble 裡時間是假的，睡眠與經過的時間都是確定的，下面的測試因此可以斷言精確的數字。這些測試斷言的是「問題真的發生」：

```go,ignore
{{#include ../../../examples/chapter10/04_rate_limit/before/limiter_test.go:problems}}
```

**睡眠版。** 20 個 goroutine 同時呼叫 `Wait`，同時睡 200 毫秒、同時醒來，第一秒內放行了 20 個，上限是 5。睡眠只是讓每個呼叫晚一點，沒有任何地方記得「這一秒已經放行幾個」。它也沒有突發的概念：閒置很久之後的第一個呼叫，同樣要睡滿 200 毫秒。

**計數器版。** 有鎖，並行下的計數是對的，問題在視窗的邊界。bubble 的時鐘從整數秒開始（午夜 UTC 2000-01-01），0.9 秒放行 5 個，1.0 秒換了視窗、計數歸零，又放行 5 個：100 毫秒內過了 10 個。負載剛好落在邊界兩側時，下游實際收到的是上限的兩倍。

## 較好的做法

`rate.Limiter` 是 token bucket：桶子最多裝 `burst` 個 token，以每秒 `limit` 個的速度補，每個呼叫先取一個 token 才能做。閒置時桶子會裝滿，所以突發可以一次過；長期下來速率不會超過 `limit`。`Limiter` 可以讓多個 goroutine 同時使用。

**往外：呼叫下游前 `Wait`。** 有 token 就立刻回來，沒有就阻塞到有；ctx 被取消，或預期的等待時間超過 deadline，就回錯誤：

```go,ignore
{{#include ../../../examples/chapter10/04_rate_limit/after/limiter.go:throttle}}
```

10 個 goroutine 同時送，`burst` 是 5：前 5 個在 0 秒過，之後每 200 毫秒一個，最後一個在 1 秒。拿掉 `Wait` 那一步，10 個呼叫全在 0 秒，測試就失敗，所以這個測試證明了並行的呼叫共用同一份額度：

```go,ignore
{{#include ../../../examples/chapter10/04_rate_limit/after/limiter_test.go:wait}}
```

`Wait` 在 deadline 之前一定等不到 token 時，不會等到逾時才失敗，而是立刻回錯誤，下游也不會被呼叫。deadline 怎麼從呼叫端往下分配，本章〈[逾時預算](01_timeout_budget.md)〉談過：

```go,ignore
{{#include ../../../examples/chapter10/04_rate_limit/after/limiter_test.go:deadline}}
```

**往內：伺服器端 `Allow`，不過就回 429。** 伺服器端不該 `Wait`：讓請求排隊會占住連線與 goroutine，超載時拒絕誰，本章〈負載卸載與降級〉會談。`Allow` 不阻塞，沒有 token 就回 `false`。每個 client 各有一個 limiter，所以一個 client 用完額度，不影響其他人：

```go,ignore
{{#include ../../../examples/chapter10/04_rate_limit/after/limiter.go:perclient}}
```

`Allow` 只回 `bool`，不會說還要等多久，`Retry-After` 要自己算：還缺的 token 數除以補充速度，進位到整數秒（標頭的最小單位是秒，所以最小是 1）。client 收到之後怎麼等，本章〈[重試、退避與 jitter](02_retry_backoff_jitter.md)〉談過。下面的測試都用這兩個 helper：

```go,ignore
{{#include ../../../examples/chapter10/04_rate_limit/after/limiter_test.go:helpers}}
```

每秒補 1 個、最多累積 3 個：alice 連送 5 個，前 3 個通過、後 2 個回 429；同一時間 bob 的請求照常通過。拿掉 client 的區分（全部共用一個 limiter），bob 的請求會被擋，這個測試就失敗。等 1 秒（假時間）之後補回剛好 1 個：

```go,ignore
{{#include ../../../examples/chapter10/04_rate_limit/after/limiter_test.go:clients}}
```

每 10 秒補 1 個時，`Retry-After` 隨時間遞減：剛被擋是 10，過 4 秒再試是 6，再過 6 秒就通過：

```go,ignore
{{#include ../../../examples/chapter10/04_rate_limit/after/limiter_test.go:retryafter}}
```

回應的樣子，用 `Example` 看（每小時才補 1 個，所以 `Retry-After` 是 3600，不必等任何時間）：

```go,ignore
{{#include ../../../examples/chapter10/04_rate_limit/after/example_test.go:example}}
```

`Reserve` 是第三種：先預約、拿到要等多久，自己決定怎麼等，適合「慢下來但不丟」；要在 deadline 或取消時放棄，用 `Wait` 比較省事。拒絕後呼叫 `Cancel` 歸還預約，文件只保證「盡可能」還原，並行時不見得完全還原，所以伺服器端用 `Allow`。

## 取捨

- **多個副本時，單機限流不準。** 每個副本各自有一份額度，3 個副本各設 100，整體上限就是 300；副本數隨擴縮變動，整體上限也跟著變。要精確的全域限制，得有共用的計數（Redis 之類）或放在閘道，這些只提名稱，這集不談。
- **依 client 的 limiter 要清理。** 範例的 map 只增不減，client 一多就吃記憶體；標頭是 client 自己填的，換一個 ID 就拿到新額度。正式環境用認證過的身分當鍵，並定期清掉閒置的 limiter：桶子已經裝滿的 limiter 和新建的沒有差別，清掉不會放寬限制。依 client 記錄被拒絕的次數時，client ID 不要當指標的屬性，原因見第 9 章〈[指標（metrics）設計與高基數問題](../chapter9/03_metrics_cardinality.md)〉。
- **限流和並行上限常要一起設。** 並行上限管「同時幾個」，限流管「每秒幾個」；下游一筆要 3 秒、限每秒 10 個，同時就會有 30 個。往外呼叫時，`Wait` 加上第 6 章〈[`errgroup` 的並行上限](../chapter6/03_errgroup_limit.md)〉的 `SetLimit`，兩個上限各管一件事。
- **擋掉的也可能是合法流量。** `burst` 太小，正常的瞬間高峰也會被拒絕；太大，保護就形同虛設。數字從下游的合約與量測來。
- **依賴。** 依本書「標準庫與 `golang.org/x` 優先」的原則，限流用 `golang.org/x/time/rate`，不自己寫；它是本書唯一用到 `golang.org/x/time` 的一集。

before 在這些情況下反而夠用：只有一個呼叫端依序送、不在乎突發的腳本，睡一個間隔最簡單；登入失敗次數這類只要大致擋住的限制，計數器的邊界誤差可以接受。

## 重點整理

- 睡一個間隔，並行的呼叫會同時醒來；每秒歸零的計數器，在視窗邊界可以放行兩倍。兩者都沒有突發的概念。
- `rate.Limiter` 是 token bucket：`burst` 決定一次能過多少，`limit` 決定長期速率；多個 goroutine 共用同一個。
- 往外用 `Wait`，deadline 之前等不到就立刻回錯誤；伺服器端用 `Allow`，不過就回 429 並帶 `Retry-After`。
- 依 client 分開 limiter，一個 client 用完額度不影響其他人；map 要清理，鍵要來自可信的身分。
- 單機限流不管副本數與並行數：多個副本要共用計數，並行上限要另外設。
