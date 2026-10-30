# 冪等性

## 本集目標

判斷什麼樣的操作可以安全地重試：不是冪等的操作（建立訂單、扣款），要怎麼用冪等鍵讓重複的請求只生效一次、回傳第一次的結果。

## 情境

一個付款服務提供 `POST /charges`，結帳流程呼叫它扣款。呼叫端遇到逾時就重試，本章〈[重試、退避與 jitter](02_retry_backoff_jitter.md)〉談過該重試哪些錯誤、怎麼退避。但逾時還有一個問題那集沒有回答：逾時代表 client 沒收到回應，伺服器可能還沒收到請求，也可能已經扣完款、只是回應沒送到。client 分不出這兩種情況。

HTTP handler 與 `httptest` 前作教過：〈[JSON API](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter14/04_json_api.html)〉、〈[`httptest`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter14/09_httptest.html)〉。

## 常見寫法

handler 解析請求、扣一次款、回 201 與扣款紀錄。帳本是記憶體版，只為了讓範例能跑：

```go,ignore
{{#include ../../../examples/chapter10/03_idempotency/before/payment.go:handler}}
```

單獨看，這個 handler 沒有毛病：每個請求就是一筆扣款。完整程式碼在 `examples/chapter10/03_idempotency/before/`。

## 問題在哪

重試的前提是「做兩次和做一次一樣」，`POST /charges` 不符合。下面的 client 用自訂的 `RoundTripper` 讓第一個請求照常送達、回應被丟掉，這正是逾時的樣子，然後照標準做法重試：

```go,ignore
{{#include ../../../examples/chapter10/03_idempotency/before/example_test.go:example}}
```

client 的每個動作都合理：逾時、重試、第二次拿到 201。結果帳本裡有兩筆 `A-1`，客人被扣了兩次款。重試愈積極，這種事愈常發生；網路與伺服器都沒有出錯，錯在這個操作本身不能重做。

## 較好的做法

讓 client 為「一個動作」產生一個冪等鍵（Idempotency-Key），這個動作的每一次嘗試都帶同一個鍵；換一個新動作才換鍵。伺服器用鍵記住第一次的結果：

```go,ignore
{{#include ../../../examples/chapter10/03_idempotency/after/store.go:store}}
```

`recorded` 記下狀態碼、標頭與內容，同時實作 `http.ResponseWriter`，用來接住 `next` 的輸出。`claim` 在同一把鎖裡查詢並登記，所以同一個鍵同時進來好幾個請求，只有一個會拿到 `first`。`Wrap` 把它接到 handler 前面：

```go,ignore
{{#include ../../../examples/chapter10/03_idempotency/after/store.go:wrap}}
```

幾個細節。第一個請求執行 `next`，結果記進 `entry`，最後關閉 `done`；後到的請求（不管是第二次嘗試，還是第一個還在跑時同時進來的）等 `done`，然後拿到同一份回應，並帶 `Idempotent-Replayed: true`。`run` 用 `defer` 保證 `done` 一定會關：`next` 中途 panic 的話，記成 500，後到的請求不會永遠等下去。等待的請求自己的 `ctx` 結束就離開，不影響別人。鍵相同、內容不同，表示 client 用錯了鍵，回 422，不執行也不回放。

測試用的 `post` 送一個帶鍵的請求：

```go,ignore
{{#include ../../../examples/chapter10/03_idempotency/after/payment_test.go:post}}
```

同樣的重試，這次只扣一次款，回的是第一次的結果：

```go,ignore
{{#include ../../../examples/chapter10/03_idempotency/after/example_test.go:example}}
```

並行的重複請求是最容易寫錯的地方。測試用 `testing/synctest`（用法見第 5 章〈[測時間與並行：用 `testing/synctest`，還是自己注入時鐘](../chapter5/08_synctest_or_clock.md)〉）：handler 真的阻塞在 channel 上，三個同鍵的請求同時送出，`synctest.Wait` 等到全部停住，才檢查「放行之前，進到 handler 的只有一個」，再放行並確認三個請求拿到同一份回應：

```go,ignore
{{#include ../../../examples/chapter10/03_idempotency/after/payment_test.go:concurrent}}
```

這個斷言證明得了並行去重：`claim` 若沒有共用 `entry`（每個請求都當第一個），三個請求都會進到 handler 並卡住，放行前的執行次數是 3，測試失敗。同一個目錄裡另有測試涵蓋不同鍵各自獨立、同鍵不同內容回 422、沒帶鍵回 400、handler panic 後重試不會卡住；`go test -race` 通過。

## 取捨

**儲存位置。** 這個 `Store` 只適合示範：重啟就消失、多個副本互相看不到、`entries` 只增不減。正式環境把鍵存進資料庫，靠唯一限制（`PRIMARY KEY` 或 `UNIQUE`）搶占：`INSERT` 成功的請求是第一個，其餘的撞到限制，就改去讀第一次的結果。鍵與扣款寫在同一個交易裡（第 4 章〈[交易邊界放在哪裡](../chapter4/02_transaction_boundary.md)〉），兩者才會一起成功或一起失敗；如果扣款是呼叫外部的金流，把同一個鍵一併傳給對方，否則兩者之間還是有重複的縫隙。

**保存期限。** 鍵不能永遠留著，也不能比 client 的重試窗口短：期限要涵蓋自動重試、排程重送，以及人工補送的最長間隔。過期之後同一個鍵會被當成新請求；期限由業務決定，沒有通用的數字。

**同鍵不同內容。** 這裡用內容的雜湊比對，不一致就回 422。只比鍵不比內容，client 的程式錯誤（兩個動作共用一個鍵）會被悄悄當成重試，第二個動作永遠不會執行。鍵也要限定在同一個 client 或帳號之內，否則別人的鍵可能撞到你的。

**執行中與失敗。** 同鍵的請求撞上還在執行的第一個，這裡選擇等它、回同一份結果；也可以直接回 409 Conflict 叫 client 稍後再試，伺服器不必佔住連線。第一次的結果不論成敗都記，連 5xx 也是，所以第一次若因暫時性錯誤失敗，重試只會拿到同一個失敗。要讓失敗可以重做，得先確定失敗沒有留下副作用，這取決於操作本身，這集不處理。

**HTTP 方法的冪等語意。** 依 RFC 9110 第 9.2.2 節，`GET`、`PUT`、`DELETE` 與其他安全方法本來就是冪等的：多次相同請求對伺服器的預期效果和一次相同。`POST` 不保證，所以需要冪等鍵。冪等指的是效果，不是回應：重複的 `DELETE` 第二次可以回 404，仍然是冪等。`PUT` 若寫成「在原值上加一」就不冪等了，方法本身不能代替設計。

**before 什麼時候夠用？** 操作天生冪等（`PUT` 整份取代、`DELETE`），或重複一次沒有代價，就不需要鍵。client 不重試、人也不會重複點擊的內部批次，也可以不做；但只要有自動重試，就該補上。

## 重點整理

- 逾時時 client 不知道請求有沒有生效；對非冪等的操作重試，可能做兩次。
- 冪等鍵由 client 為每個動作產生一次，每次嘗試都帶；伺服器記住第一次的結果，重複的鍵回放同一份回應。
- 查詢與登記要在同一個原子操作裡完成，同鍵同時進來才只會執行一次；測試要用真的阻塞證明，並檢查放行之前的執行次數。
- 同鍵不同內容要拒絕；記憶體版只適合示範，正式環境用資料庫的唯一限制，並訂保存期限。
- `GET`、`PUT`、`DELETE` 本來就冪等，`POST` 才需要鍵；冪等看的是效果，不是回應。
