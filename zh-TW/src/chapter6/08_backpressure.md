# 背壓

## 本集目標

producer 比 consumer 快的時候，判斷工作佇列滿了該怎麼辦：讓 producer 等、立刻拒絕，還是丟掉工作；也知道為什麼「不設上限」只是把問題往後延。

## 情境

縮圖服務：使用者上傳照片，handler 把縮圖工作交給一個背景的 consumer 依序處理；另一支批次匯入程式，也用同一個佇列一次送進整本舊相簿。縮圖要讀寫儲存空間，儲存變慢時，consumer 的速度就跟著掉。

buffered channel 與 `select` 加 `default` 前作教過：〈[buffered channel](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter9/09_buffered_channel.html)〉、〈[`select`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter9/11_select.html)〉。這集談的是佇列滿了之後的決定。

## 常見寫法

`Submit` 把工作 append 到 slice，立刻回傳；consumer 一筆一筆取出來做。

```go,ignore
{{#include ../../../examples/chapter6/08_backpressure/before/queue.go:queue}}
```

呼叫端不會被擋住，也不必處理錯誤，看起來很省事。完整程式碼在 `examples/chapter6/08_backpressure/before/`。

## 問題在哪

佇列的長度只由 producer 的速度決定。下面的 `Example` 讓 consumer 取走第一筆之後卡住（模擬儲存變慢），再分別送 100 筆與 10000 筆；等待靠 channel 訊號，輸出每次都一樣：

```go,ignore
{{#include ../../../examples/chapter6/08_backpressure/before/example_test.go:example}}
```

工作量多十倍，排隊的數量也多十倍，沒有任何地方會喊停。真實的 `Job` 帶的是圖片，記憶體跟著排隊數量一路長上去，最後耗盡記憶體。

記憶體只是其中一個代價，另一個是等待時間：第 N 筆要等前面 N-1 筆做完。使用者早就放棄、重新整理了，那筆工作還在隊伍裡，輪到它時做出來的縮圖沒人要。很大的 buffered channel，或每個進來的工作開一個 goroutine，都是同樣的問題：只是換了排隊的地方。

## 較好的做法

排隊的地方改成有容量的 channel。容量就是上限，滿了以後 producer 要面對一個決定，所以 `Submit` 變成兩種：

```go,ignore
{{#include ../../../examples/chapter6/08_backpressure/after/queue.go:queue}}
```

`Submit` 是**等**：滿了就停在送出那一行，直到有空位或 `ctx` 結束。producer 被擋住，就是壓力往上傳，批次匯入程式會自然變慢到和 consumer 一樣快。`TrySubmit` 是**拒絕**：用 `select` 加 `default`，滿了立刻回傳 `ErrFull`。

同樣的卡住的 consumer，容量 3，這次用 `TrySubmit`，送 100 與 10000 筆，排隊的數量都停在 3：

```go,ignore
{{#include ../../../examples/chapter6/08_backpressure/after/example_test.go:example}}
```

`Submit` 的等待用 `ctx` 收尾：consumer 卡住、佇列又滿時，只有 `ctx` 結束它才會回來。取消要傳進每個會阻塞的地方，本章〈[取消與 `context` 的傳遞](04_cancellation_context.md)〉談過。

```go,ignore
{{#include ../../../examples/chapter6/08_backpressure/after/example_test.go:wait}}
```

「producer 真的在等」用 `synctest` 確定地證明，不必等任何時間：`gate` 讓 consumer 每次只做完一筆，`synctest.Wait` 等到 producer 卡住，檢查它還沒回傳；放行一筆之後，它就回傳了。`synctest` 的基本規則見第 5 章〈[測時間與並行：用 `testing/synctest`，還是自己注入時鐘](../chapter5/08_synctest_or_clock.md)〉。

```go,ignore
{{#include ../../../examples/chapter6/08_backpressure/after/queue_test.go:wait}}
```

拒絕的測試也在同一個檔案：滿了回 `ErrFull`、被拒絕的工作不會排進去、空出一格之後又收得下。

```go,ignore
{{#include ../../../examples/chapter6/08_backpressure/after/queue_test.go:reject}}
```

兩種各有適合的呼叫端：

- **批次匯入可以等。** 沒有人在等結果，慢一點只是花更久，而且被擋住正是想要的效果。
- **使用者請求應該快速失敗。** handler 把 `ErrFull` 轉成 `503 Service Unavailable`（可加 `Retry-After`），讓使用者或前端知道稍後再試，比讓請求卡到逾時好。

`errgroup` 的 `Go` 對應「等」，`TryGo` 對應「拒絕」，本章〈[`errgroup` 的並行上限](03_errgroup_limit.md)〉談過。

## 取捨

- **容量是在決定願意讓工作等多久。** 最壞的等待約為容量乘以單筆的處理時間，不是越大越好；容量大到跟沒有上限差不多時，問題又回來了。容量 0（無緩衝）也合法，producer 和 consumer 直接交接。
- **上限擋不住長期的失衡。** 佇列只能吸收短暫的突波。producer 長期比 consumer 快，有上限只是把「記憶體長大」換成「一直被擋、一直被拒絕」，要解決得增加 consumer 或減少進來的量。consumer 的形式（worker pool 或每個任務一個 goroutine），本章〈[什麼時候需要 worker pool，什麼時候每個任務一個 goroutine 就夠](06_worker_pool_or_goroutine_per_task.md)〉談過；開幾個最好是要量測的效能問題，第 7 章〈[先量測，再最佳化](../chapter7/01_measure_before_optimizing.md)〉談過。
- **丟掉工作也是一種選擇，但要讓呼叫端知道。** `TrySubmit` 就是丟掉新的，而且用 `ErrFull` 明說了。有些資料只有最新的才有意義（例如即時位置），可以改成丟掉最舊的。用 channel 做的話，要先取出最舊的、再放進新的，這兩步之間 consumer 或別的 producer 可能插進來：consumer 已經取走，取出那一步就會等；別的 producer 已經補上，放入那一步就會等。所以得把兩步包成一個不可分的動作，既然要上鎖，用 mutex 保護、只留最新一筆比較直接。不管丟哪一個，都要有人知道：回錯誤、記計數或 log。靜默丟棄是資料遺失。
- **負載卸載與降級是下一步。** 佇列滿了之後整個服務要怎麼縮小服務範圍，第 10 章〈負載卸載與降級〉會談。
- **範例的 `Close` 有前提。** 它要在所有 producer 停止之後才呼叫，否則對已關閉的 channel 送出會 panic。誰負責停誰，本章〈[goroutine 的生命週期歸誰管](01_goroutine_ownership.md)〉談過。

before 在這些情況下反而是對的：工作量有已知的小上界（例如啟動時載入固定幾十筆設定），或突波很短、記憶體綽綽有餘。這時 `Submit` 不會失敗，呼叫端不必處理 `ErrFull`，少一個要設的數字。

## 重點整理

- 沒有上限的佇列，排隊的數量由 producer 決定；consumer 一慢，記憶體與等待時間一起成長。
- 有上限的 channel 滿了，要在三種做法裡選：讓 producer 等、立刻拒絕、丟掉工作。
- 等的時候要帶 `ctx`，否則取消了還卡在送出；拒絕時回傳明確的錯誤，handler 轉成 503。
- 批次工作適合等，使用者請求適合快速失敗；丟掉工作一定要讓呼叫端知道。
- 用卡住的 consumer 加 channel 訊號或 `synctest`，就能確定地測出排隊數量與 producer 有沒有在等。
