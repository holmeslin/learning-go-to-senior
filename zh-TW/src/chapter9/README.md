# 可觀測性

前作已經教過 `log/slog` 的基本用法：〈[`log/slog`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter12/10_slog.html)〉，也教過 `net/http/pprof` 的掛法：〈[`pprof`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/appendix2/d_pprof.html)〉。服務上線之後，缺的不是工具，而是判斷。實際上線時常犯同樣的錯：把變數拼進日誌訊息，事後無法依欄位查詢；同一個錯誤每一層都記一次，或在迴圈裡用 Info 記每一筆；指標的屬性放了使用者代號，時間序列一路長上去；追蹤的 span 開在每個小函式，跨服務的邊界卻斷掉；`/debug/pprof` 掛在對外的 mux 上；liveness 檢查資料庫，資料庫一慢整批服務被重啟；延遲直方圖的 bucket 跟 SLO 的門檻對不上，算不出 SLI。

這一章的主題是：服務上線之後，怎麼讓它說得出自己的狀態。日誌、指標、追蹤回答的是不同的問題：日誌說「這一次請求發生了什麼」，指標說「整體現在怎麼樣、趨勢往哪走」，追蹤說「一個請求經過哪些地方、時間花在哪、在哪裡斷掉」。三者不是互相取代，每一集處理其中一個判斷，另外也談診斷端點、健康檢查與服務水準目標。範例的遙測都在記憶體裡收集：日誌寫進 `bytes.Buffer`，span 與指標用 OpenTelemetry 的記憶體 recorder 與 reader，端點用 `httptest`，所以 `go test` 不需要 collector 或任何外部服務；確定的結果才進測試與 `Example` 的輸出，時間戳、ID 與耗時不斷言。
