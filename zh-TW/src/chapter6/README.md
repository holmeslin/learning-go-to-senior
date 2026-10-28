# 結構化並行

前作已經教過並行的工具：〈[goroutine](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter9/01_goroutine.html)〉、〈[`sync.WaitGroup` 與 `wg.Go`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter9/02_waitgroup.html)〉、〈[channel](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter9/08_channel.html)〉、〈[`select`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter9/11_select.html)〉，還有〈[為什麼需要 context](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter10/01_why_context.html)〉。會用這些工具的人，組成一個服務時常犯同樣的錯：在 handler 裡 `go` 出去就不管，關機時停不下來、錯誤也不見了；一萬筆資料就開一萬個 goroutine 打下游；請求取消了，goroutine 還在做沒人要的工作；洩漏的 goroutine 只能在正式環境的 goroutine 數量上看到。

這一章談的是結構化並行：每個 goroutine 的開始與結束都有明確的範圍與擁有者，誰開始，誰負責等它結束，函式回傳之後不留下沒人管的 goroutine。這是和前作第 9 章最大的差別：前作教你怎麼開 goroutine，這一章教你怎麼管。`errgroup` 也在這一章第一次出現，前作沒有教。每一集處理一個判斷，書中的輸出盡量來自會被 `go test` 逐字驗證的 `Example`；並行的範例輸出每次都一樣，不依賴 goroutine 的執行順序。
