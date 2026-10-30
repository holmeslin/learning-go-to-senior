# 分散式追蹤與 OpenTelemetry

## 本集目標

讀得懂 trace 的輸出，並判斷 span 該開在哪裡：進來的請求、對外的呼叫、資料庫查詢，而不是每個小函式；錯誤要記在 span 上；`TracerProvider` 由參數傳入，不從全域取。

## 情境

播放清單服務的 `GET /playlists/{id}`：查清單的曲目編號，每首曲目再查一次標題，最後呼叫封面服務取封面。有人回報某些清單偶爾很慢、偶爾回 500，團隊想用 trace 看出時間花在哪一步、哪一步失敗。

先把名詞講清楚。一個請求從頭到尾是一條 **trace**，由許多 **span** 組成；span 是一段有名稱、有起訖的工作，子 span 在父 span 的 `context` 下開出來，所以 `context` 要一路傳下去（前作〈[context 傳遞慣例](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter10/07_context_conventions.html)〉）。span 可以帶**屬性**（鍵值）、**事件**（某個時間點的記錄，例如錯誤）與 **status**（`Unset`、`Ok`、`Error`）。

OpenTelemetry 分成兩半。API（`go.opentelemetry.io/otel/trace` 等）只定義介面，沒有人設定時什麼都不記錄；SDK（`go.opentelemetry.io/otel/sdk/trace`）才負責記錄、取樣與送出。函式庫和業務程式碼只依賴 API，接哪個 SDK 由 `main` 或測試決定，這就是兩者分開的原因。HTTP client 與 `httptest` 見前作〈[HTTP client](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter14/06_http_client.html)〉與〈[`httptest`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter14/09_httptest.html)〉。

## 常見寫法

每個函式各開一個 span，provider 從全域取得：

```go,ignore
{{#include ../../../examples/chapter9/04_tracing_opentelemetry/before/playlist.go:spans}}
```

看起來很徹底：函式一路傳 `ctx`，每一層都有 span。完整程式碼在 `examples/chapter9/04_tracing_opentelemetry/before/`。

## 問題在哪

用 `tracetest.SpanRecorder` 收 span，`dump` 把它們依父子關係縮排印出來，方括號裡是 status：

```go,ignore
{{#include ../../../examples/chapter9/04_tracing_opentelemetry/before/example_test.go:dump}}
```

`p1` 有三首曲目，`p2` 的封面服務回 503。`Example` 先在沒有替換 `tracer` 時送一次 `p1`，再替換、送 `p1` 與 `p2`。輸出由 `go test` 逐字比對：

```go,ignore
{{#include ../../../examples/chapter9/04_tracing_opentelemetry/before/example_test.go:example}}
```

三個問題都在輸出裡：

- **recorder 一個 span 都沒收到。** `NewService` 沒有 provider 參數，`tracer` 來自全域，測試自己建的 provider 接不上去。要看到 span，只能替換套件變數；測試因此不能平行，而且忘了還原就會影響下一個測試。
- **span 開在不該開的地方，該開的地方沒有。** `title` 與 `normalize` 是純粹的小計算，卻占了 `p1` 八個 span 裡的六個；真正會慢、會失敗的查詢與封面呼叫沒有 span。清單慢的時候，trace 只能說 `render` 很久，說不出是哪一步。
- **失敗的請求在 trace 裡看起來是成功的。** `p2` 回了 500，所有 span 都是 `Unset`、沒有任何事件。依 status 找出失敗的 trace 時，這一條不會出現。

## 較好的做法

provider 當參數，span 開在請求的入口。名稱用路由樣板 `r.Pattern`，實際的 id 放進屬性，名稱才不會每個請求都不同；屬性寫在 `Start` 的選項裡，因為取樣器只看得到建立時已經有的屬性。失敗時 `RecordError` 加上 `SetStatus(codes.Error, ...)`，兩個都要呼叫（`RecordError` 只加事件，不改 status）。找不到清單回 404，是呼叫端的錯，status 留在 `Unset`：

```go,ignore
{{#include ../../../examples/chapter9/04_tracing_opentelemetry/after/playlist.go:entry}}
```

這個套件只 import API 的 `trace`、`attribute`、`codes`，沒有 import SDK。接著是資料庫查詢與對外呼叫，兩者都是會花時間、會失敗的邊界。查無資料不是查詢失敗，所以 `SELECT playlists` 不記錄錯誤；`title` 每首曲目查一次，N 首就是 N 個同名的 `SELECT tracks`：

```go,ignore
{{#include ../../../examples/chapter9/04_tracing_opentelemetry/after/playlist.go:queries}}
```

對外呼叫用 `SpanKindClient`，錯誤集中在 `defer` 裡記錄一次，不管從哪個 `return` 離開：

```go,ignore
{{#include ../../../examples/chapter9/04_tracing_opentelemetry/after/playlist.go:cover}}
```

同樣的資料與請求，輸出（`dump` 與前面相同）：

```go,ignore
{{#include ../../../examples/chapter9/04_tracing_opentelemetry/after/example_test.go:example}}
```

樹裡只剩請求、查詢、對外呼叫；`p2` 的失敗從封面呼叫一路標到請求，事件各留一筆 `exception`。N+1 也看得見了：`SELECT tracks` 是請求底下一串同名的子 span。修法在第 4 章〈[N+1 與查詢組裝](../chapter4/05_n_plus_one.md)〉，上線後看到這種形狀就知道該回去數查詢次數。

測試放在外部測試套件，每個測試建自己的 provider 與 recorder，不碰全域設定，所以都可以 `t.Parallel`：

```go,ignore
{{#include ../../../examples/chapter9/04_tracing_opentelemetry/after/service_test.go:fixture}}
```

N+1 用子 span 的數量斷言：曲目數 N 就有 N 個 `SELECT tracks`，帶著各自的 `track.id`；所有子 span 的父 span 都是請求 span：

```go,ignore
{{#include ../../../examples/chapter9/04_tracing_opentelemetry/after/service_test.go:siblings}}
```

status 與事件：404 與成功都沒有 `Error` 的 span；封面服務掛掉時，封面與請求這兩個 span 是 `Error`，而且各有 `exception` 事件：

```go,ignore
{{#include ../../../examples/chapter9/04_tracing_opentelemetry/after/service_test.go:status}}
```

取樣由 provider 的選項決定：同一份程式碼，比例 1 時收到六個 span，比例 0 時一個都沒有：

```go,ignore
{{#include ../../../examples/chapter9/04_tracing_opentelemetry/after/service_test.go:sampling}}
```

另外有一個測試傳入 `noop.NewTracerProvider()`，確認不要 trace 時服務照常運作。測試不斷言 trace ID、span ID 與耗時，這些每次都不同。完整程式碼在 `examples/chapter9/04_tracing_opentelemetry/after/`。

## 取捨

- **span 越多不是越好。** 每個 span 都有建立、記錄、送出與儲存的成本，太細的 span 也讓 trace 難讀。判斷的標準是「這裡會不會花時間或失敗」：網路、資料庫、鎖與佇列的等待值得；純計算不值得，CPU 的熱點交給第 7 章〈[`pprof`：先看 CPU 還是 heap](../chapter7/03_pprof_cpu_or_heap.md)〉。N+1 的 N 很大時，trace 也會有 N 個 span，成本跟著成長。
- **取樣在 provider 設定，程式碼不用改。** `ParentBased(TraceIDRatioBased(x))` 在根 span 依 trace ID 取一個比例，子 span 跟著父 span 的決定，整條 trace 要嘛完整、要嘛沒有。測試用比例 1 與 0 證明這件事（見上面的 `TestSamplingIsProviderConfig`）。代價是決定在請求一開始就做，當時還不知道這個請求會不會失敗，所以「只留出錯的 trace」要在 collector 端做（尾端取樣），這裡只提名稱。
- **全域 provider 不是錯的，是有適用情境。** 小工具或 `main` 裡設定一次、函式庫無法改簽名時，用 `otel.SetTracerProvider` 最省事；contrib 的 `otelhttp` 預設也是讀全域。代價就是 before 看到的：測試接不上、不能平行、依賴藏在各處。範例選擇傳參數；兩者可以並存，`main` 設定全域，函式庫提供「傳入 provider」的選項。
- **exporter 與 collector 只提名稱。** 實際送出 span 要接 OTLP exporter 與 collector，這集用記憶體裡的 recorder 取代。`otelhttp` 可以替 handler 與 client 自動開 span，上面兩個邊界手寫是為了看清楚 span 裡有什麼。
- **這條 trace 只在這個程序裡。** 對外呼叫沒有把 trace 傳給封面服務，對方的 span 不會接在這條 trace 底下；怎麼傳遞，本章〈在 `context` 裡傳遞 trace〉會談。
- **id 可以放 span 屬性，不能放指標屬性。** 本章〈指標（metrics）設計與高基數問題〉談過原因；span 屬性仍有成本，也不要放敏感資料。

## 重點整理

- trace 由 span 組成，父子關係靠 `context` 傳遞；span 有屬性、事件與 status。API 只定義介面、沒設定就不記錄，SDK 負責記錄與取樣，所以程式碼只依賴 API。
- span 開在請求、對外呼叫與資料庫查詢的邊界，名稱用樣板、id 放屬性；每個小函式各開一個，只會把真正的邊界淹沒。
- 失敗要 `RecordError` 加 `SetStatus(codes.Error, ...)`；呼叫端造成的 4xx 與查無資料不算服務的錯。
- N+1 在 trace 裡是請求底下一串同名的子 span，測試用 span 數量斷言。
- `TracerProvider` 由參數注入，測試用 `SpanRecorder` 斷言名稱、父子關係、屬性與 status；取樣是 provider 的設定，不寫在程式碼裡。
