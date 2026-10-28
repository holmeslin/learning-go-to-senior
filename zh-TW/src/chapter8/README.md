# runtime

前作已經教過 GC 的運作與調整：〈[GC 概念](https://holmeslin.github.io/learning-go-from-zero/zh-TW/appendix2/c_gc.html)〉談過標記清除、`GOGC`、`GOMEMLIMIT`。會用這些設定的人，實際上線時常犯同樣的錯：以為 goroutine 開越多，CPU 密集的工作越快；記憶體看起來高，就手動呼叫 `runtime.GC()` 或把 `GOGC` 調低，結果 CPU 被 GC 吃掉；服務在容器裡超過記憶體上限被 OOM 殺掉，卻不知道 `GOGC` 只看比例、不管容器的上限；為了舊版的容器問題寫死 `GOMAXPROCS`，反而關掉了 Go 1.25 起的自動調整；用 `runtime.ReadMemStats` 自己湊指標；大量 goroutine 卡在阻塞式呼叫，執行緒數一路長上去；看到 `gctrace` 不知道每一欄在說什麼。

這一章談的是 runtime 的預設值在什麼情況下不夠，以及什麼時候不要動它。大多數服務用預設值就好；每一集處理一個判斷：排程器怎麼看、什麼時候該調 `GOGC`、容器裡的記憶體與 CPU 上限、heap 為什麼在成長、怎麼從 runtime 讀出指標、阻塞的呼叫怎麼影響執行緒、`gctrace` 怎麼讀。確定的結果用會被 `go test` 驗證的測試鎖住；GC 次數、heap、執行緒數與時間每次、每台機器都不同，不放進測試。書中貼出的 runtime 診斷輸出都標明是某一次、在什麼平台與幾個 P 上取得的結果，結論只看欄位的意思與相對的變化。
