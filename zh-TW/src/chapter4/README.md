# 資料存取的設計

前作已經教過 `database/sql` 的機制：開連線、查詢與寫入、〈[交易](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter15/06_transaction.html)〉、`NULL`，以及〈[context 與連線池](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter15/08_context_pool.html)〉。會用這些 API 的人，組成一個服務時還是常犯同樣的錯：SQL 散在 handler 裡，或反過來包一層什麼都能做的 repository；每個 store 方法各自開交易，轉帳只成功一半；查詢自己設逾時，請求取消了它還在跑；連線池用預設值，一忘了關 `rows`，所有請求就卡在等連線。

這一章談的是這些決定。它們和前作的差別在於：平常看不出問題，要等上線、流量變大，才以事故的形式回來找你。所以每一集處理一個判斷：SQL 該放哪、交易邊界在哪、逾時誰設、連線池多大、查詢幾次、schema 怎麼演進、資料層怎麼測。範例使用真的 SQLite，不需要另外安裝資料庫，書中的輸出也盡量來自會被 `go test` 逐字驗證的 `Example`。
