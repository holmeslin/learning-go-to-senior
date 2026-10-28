# 效能調查流程

前作已經教過量效能的工具：〈[benchmark 與 `b.Loop`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/12_benchmark.html)〉、〈[逃逸分析](https://holmeslin.github.io/learning-go-from-zero/zh-TW/appendix2/b_escape_analysis.html)〉、〈[GC 概念](https://holmeslin.github.io/learning-go-from-zero/zh-TW/appendix2/c_gc.html)〉、〈[`pprof`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/appendix2/d_pprof.html)〉。會用這些工具的人，實際調查效能時常犯同樣的錯：沒量測就憑直覺改寫，程式變難讀卻沒有變快；benchmark 只跑一次，把雜訊當成改善；症狀是延遲高、CPU 卻不忙，還在看 CPU profile；聽說 PGO 能加速就開，卻拿不具代表性的 profile；在 CI 用一次 benchmark 的 ns/op 設門檻，結果時好時壞，最後被關掉。

這一章教的不是最佳化技巧的清單，而是一套調查流程：先定義指標，量測，用 profile 找熱點，一次只改一件事，再量一次。每一集處理流程中的一個判斷：怎麼讓量測可信、先看哪一種 profile、什麼時候該看 trace 或逃逸分析、常見熱點長什麼樣子、PGO 值不值得開、怎麼把效能回歸擋在 CI 之外。確定的結果（例如配置次數）用會被 `go test` 驗證的測試鎖住；時間每次、每台機器都不同，不放進測試，書中貼出的 benchmark 與 profile 輸出都標明是某一次的結果，結論只看相對的差異。
