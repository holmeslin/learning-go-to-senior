# 指標（metrics）設計與高基數問題

## 本集目標

寫 HTTP 的指標時，認得哪些屬性會讓時間序列的數量跟著請求一路成長（高基數），改用有限集合的屬性；並依用途選計數器、量測值（gauge）或直方圖，替它們取帶單位的名稱。

## 情境

會員服務要看每個端點的請求量與錯誤，於是在 HTTP 中介層加一個計數器。屬性放什麼，寫的人通常覺得「越詳細越好查」：誰打的、打的是哪個網址。同一個服務還有一個 `*sql.DB`，第 4 章〈連線池要設多大、設錯會怎樣〉談過它的 `Stats()` 要長期收集與告警，這集示範怎麼把它匯出。HTTP 伺服器、`ServeMux` 與 middleware 前作教過：〈[`ServeMux` 路由](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter14/02_servemux.html)〉、〈[middleware](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter14/05_middleware.html)〉。

## 常見寫法

屬性取自請求本身：方法、原始路徑、使用者代號。`MeterProvider` 由呼叫端傳入，中介層不碰全域設定：

```go,ignore
{{#include ../../../examples/chapter9/03_metrics_cardinality/before/middleware.go:middleware}}
```

每一種屬性組合在 SDK 裡是一個資料點，匯出到後端就是一條時間序列。程式很短，單看每個屬性也都合理。完整程式碼在 `examples/chapter9/03_metrics_cardinality/before/`。

## 問題在哪

下面的 `Example` 把請求一批批送進去，每次送完用 `ManualReader` 收一次，數計數器有幾個資料點。`/users/{id}` 只是一個路由，ID 與請求路徑每次都不同：

```go,ignore
{{#include ../../../examples/chapter9/03_metrics_cardinality/before/example_test.go:count}}
```

```go,ignore
{{#include ../../../examples/chapter9/03_metrics_cardinality/before/example_test.go:example}}
```

資料點的數量等於不同請求的數量，沒有上限。一個路由、一個方法，卻有一千條序列；上線後每個新使用者都多一條序列。使用者代號可以換成訂單編號、`request_id`、錯誤訊息，結果都一樣。只用原始路徑也會出事：掃描程式探 `/wp-admin/1`、`/wp-admin/2`，每條路徑都是新的序列，連 404 都會灌進去。

SDK 有保險：資料點超過上限（預設 2000），多出來的組合被併進一個 `otel.metric.overflow` 資料點。測試把上限明設為 2000，送 3000 種路徑：

```go,ignore
{{#include ../../../examples/chapter9/03_metrics_cardinality/before/overflow_test.go:overflow}}
```

這只能擋住記憶體與儲存，不會讓指標變準：超過上限之後，哪個端點出錯已經看不出來。屬性要在進入 SDK 之前就限制住。

## 較好的做法

屬性只留三個，值的集合都有上限：

- `http.request.method`：標準方法本身，客戶端自創的方法一律歸成 `_OTHER`。
- `http.route`：路由樣板，不是實際的路徑。`ServeMux` 比對完之後把樣板放在 `r.Pattern`（Go 1.23 起），所以中介層要包在 mux 外面，等 `next.ServeHTTP` 回來再讀。沒有比對到的（包含路徑對、方法不對的 405）是 `unmatched`。
- `http.response.status_class`：`2xx`、`4xx`、`5xx`，不是三位數的狀態碼。

```go,ignore
{{#include ../../../examples/chapter9/03_metrics_cardinality/after/middleware.go:middleware}}
```

```go,ignore
{{#include ../../../examples/chapter9/03_metrics_cardinality/after/middleware.go:bounds}}
```

三種儀器各有用途。計數器只會增加，數「發生了幾次」（請求、錯誤）；直方圖記一批數值的分布，耗時這類要看百分位的值用它；量測值是「現在多少」，可增可減，例如連線數、佇列長度。名稱用小寫加點，單位放在 `WithUnit`（秒是 `s`，計數的東西用 `{request}` 這種花括號），名稱本身不重複寫單位。名稱與單位怎麼對應到後端，由匯出的那一端處理。

測試送出 500 種不同的路徑與使用者，加上一個 404、一個 POST 與 100 條掃描路徑，斷言屬性組合恰好是下面四組。helper 在 `after/helpers_test.go`，這裡一併列出：

```go,ignore
{{#include ../../../examples/chapter9/03_metrics_cardinality/after/helpers_test.go:helpers}}
```

```go,ignore
{{#include ../../../examples/chapter9/03_metrics_cardinality/after/middleware_test.go:bounded}}
```

不認得的方法只產生一組：

```go,ignore
{{#include ../../../examples/chapter9/03_metrics_cardinality/after/middleware_test.go:method}}
```

同樣的情境，這次的 `// Output:` 是一千個使用者加一個沒有路由的路徑：

```go,ignore
{{#include ../../../examples/chapter9/03_metrics_cardinality/after/example_test.go:example}}
```

直方圖與計數器共用同一組屬性，序列數一樣（`TestHistogramSharesAttributes` 鎖住）；耗時的數值不斷言，只斷言序列。

連線池的統計用非同步的量測值匯出：數值在收集那一刻才去讀，不必另開 goroutine 定時推。`Idle` 與 `InUse` 是目前的值，用量測值；`WaitCount` 本來就是累計值，用非同步計數器直接回報，自己算差額再 `Add` 只是多繞一圈：

```go,ignore
{{#include ../../../examples/chapter9/03_metrics_cardinality/after/pool.go:pool}}
```

測試用假的連線池，改變統計後再收一次，斷言讀到的是新的值：

```go,ignore
{{#include ../../../examples/chapter9/03_metrics_cardinality/after/pool_test.go:pool}}
```

第 8 章〈`runtime/metrics`〉的讀數也可以這樣匯出：在回呼裡呼叫 `metrics.Read`，把 `/gc/heap/live:bytes` 這類單一數值用 `ObserveInt64` 或 `ObserveFloat64` 回報。

## 取捨

- **高基數的資訊不是不要記，是放對地方。** 使用者代號、請求代號、完整路徑與錯誤訊息放日誌的欄位（本章〈結構化日誌：`log/slog` 的欄位與 handler 怎麼選〉談過欄位的寫法）或 span 的屬性（本章〈分散式追蹤與 OpenTelemetry〉會談）；指標只回答「整體怎麼樣」。要找某個使用者的失敗，從日誌找。
- **屬性集合要在 code review 時看。** 看到 `attribute.String(..., 變數)` 先問：這個值有幾種可能、誰決定？客戶端能決定的（路徑、標頭、方法）都要先歸類。
- **屬性越多，成本相乘。** 序列數是各屬性取值數的乘積；直方圖的每個資料點還帶著每個 bucket 的計數，比計數器貴。bucket 邊界怎麼選，本章〈SLI 與 SLO〉會談。
- **狀態碼類別有時不夠。** 想分開看 429 與 503，可以把有限的幾個碼列進白名單，其餘歸類；前提仍是集合有上限。
- **before 在內部小工具上不一定錯。** 使用者只有幾十個、路由是固定的，序列數本來就有上限，用不著為它限制。問題是對外服務的使用者與路徑由別人決定。
- **序列數在後端也是成本。** 儲存與查詢都跟著序列數走；Prometheus 等後端各有各的設定，這裡只談送出去之前自己守住。
- **全域的 `otel.SetMeterProvider` 在這集用不到。** 函式庫接收 `metric.MeterProvider`，進入點決定用哪個 SDK，測試才換得掉。

## 重點整理

- 每一種屬性組合就是一條時間序列；使用者代號、原始路徑、錯誤訊息這類值的數量由請求決定，會讓序列數無限成長。
- 屬性用有限集合：方法（其餘 `_OTHER`）、路由樣板（對不上 `unmatched`）、狀態碼類別。SDK 的上限只是保險，不會讓資料變準。
- 計數器數次數、量測值記現在的值、直方圖記分布；單位放 `WithUnit`。
- 連線池統計用非同步儀器匯出：目前值用量測值，累計值用非同步計數器。
- 高基數的資訊放日誌或 span；屬性的集合在 code review 時看。
