# 測試策略

前作已經教過測試的工具：〈[`go test`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/09_go_test.html)〉、〈[表格驅動測試](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/10_table_driven_test.html)〉、〈[Example 測試](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/13_example_test.html)〉、〈[fuzzing](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/14_fuzzing.html)〉，還有〈[data race 與 `-race`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter9/03_data_race.html)〉。會用這些工具的人，測試寫多了仍會碰到同一類麻煩：重構一次，十個測試一起壞，程式其實沒有錯；測試全綠，bug 卻照樣上線；`go test ./...` 愈跑愈慢，最後沒人想跑；偶爾紅一次，重跑就好，於是大家習慣重跑。

這一章談的不是測試工具怎麼用，而是測試怎麼設計，才撐得住重構、跑得夠快、失敗時值得相信。每一集處理一個判斷。這些判斷沒有一體適用的答案，所以每一集的「取捨」都會說明常見寫法在什麼情況下才是對的。書中的輸出盡量來自會被 `go test` 逐字驗證的 `Example`；本章的 after 本身就是測試，你可以直接照抄它的寫法。
