# 什麼時候需要 worker pool，什麼時候每個任務一個 goroutine 就夠

## 本集目標

面對一批要並行處理的工作，判斷該手寫 worker pool，還是每個任務一個 goroutine 加並行上限：有限的批次用後者；worker 要持有昂貴的狀態、或要長時間消化沒有盡頭的來源時，才用 worker pool。

## 情境

批次抓取頁面標題：給一串 id，對每個 id 呼叫 `fetch`，同一時間最多 `limit` 個（上限由下游的容量決定，見第 4 章〈[連線池要設多大、設錯會怎樣](../chapter4/04_pool_size.md)〉）。結果要和 id 同順序，第一個錯誤要取消其餘的 `fetch`。

worker pool 的機制前作教過：〈[worker pool](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter9/17_worker_pool.html)〉，這集不重講。`errgroup` 的 `Go`、`Wait`、`WithContext` 見本章〈`errgroup`〉，`SetLimit` 見本章〈`errgroup` 的並行上限〉，這裡直接用。

## 常見寫法

看到「同時最多 N 個」，直覺是 worker pool：工作 channel、N 個 worker、結果 channel、`WaitGroup`：

```go,ignore
{{#include ../../../examples/chapter6/06_worker_pool_or_goroutine_per_task/before/batch.go:pool}}
```

順序、上限、第一個錯誤，`before/` 的測試都驗證過，行為是對的。

## 問題在哪

對一個有限的批次，這個函式要同時管：兩條 channel、N 個 worker、一個送工作的 goroutine、一個關 `results` 的 goroutine，以及出錯後的收尾。每一塊都有存在的理由，也都能寫錯。例如把送工作那段從 goroutine 拿出來，直接寫在函式裡：

```text
for i, id := range ids {
	jobs <- job{i, id}
}
close(jobs)
```

送到第三個工作時，兩個 worker 都卡在送結果，而收結果的迴圈還沒開始，彼此互相等。把這個版本放進 `synctest.Test`（repo 外的暫存模組重現），deadlock 會直接變成失敗：

執行結果：

```text
--- FAIL: TestFetchAll
panic: deadlock: all goroutines in bubble are blocked [recovered, repanicked]

goroutine N [chan send (durable), synctest bubble 1]:
repro.FetchAll(...)
	batch.go:45
```

輸出省略了耗時、其餘 goroutine 與 runtime 的 stack、參數與檔案路徑，goroutine 編號以 `N` 代替。

出錯時若收結果的迴圈直接 `return`，沒人再收 `results`，worker 與送工作的 goroutine 會永遠卡在送出，函式雖然回傳了，卻留下洩漏。這些都是為了「同時最多 N 個」付出的程式量：`FetchAll` 在 `before/batch.go` 有 53 行，下面的 after 是 20 行。

## 較好的做法

每個任務一個 goroutine，上限交給 `errgroup`：

```go,ignore
{{#include ../../../examples/chapter6/06_worker_pool_or_goroutine_per_task/after/batch.go:errgroup}}
```

沒有 channel，沒有關閉順序要記。`SetLimit(limit)` 讓 `g.Go` 在已有 `limit` 個任務時等到有人結束；每個 goroutine 只寫自己索引的那一格，所以結果保持順序，也沒有 data race（結果收集的做法本章〈fan-out / fan-in 的取捨〉會談）。before 與 after 的外部行為相同，同一組測試兩邊都通過，差別只有套件的 import 路徑。

上限用 `synctest` 驗證，不靠真的等：7 個任務、每個「花」一秒、上限 3，同時執行數的最大值必須是 3，經過的時間必須剛好 3 秒。`synctest` 的規則見第 5 章〈[測時間與並行：用 `testing/synctest`，還是自己注入時鐘](../chapter5/08_synctest_or_clock.md)〉。

```go,ignore
{{#include ../../../examples/chapter6/06_worker_pool_or_goroutine_per_task/after/batch_test.go:limit}}
```

第一個錯誤的測試用 channel 訊號，不用 `time.Sleep`：其他任務等 `ctx.Done()`，所以只有第一個錯誤取消了它們，`FetchAll` 才回傳得出來：

```go,ignore
{{#include ../../../examples/chapter6/06_worker_pool_or_goroutine_per_task/after/batch_test.go:firsterror}}
```

```go,ignore
{{#include ../../../examples/chapter6/06_worker_pool_or_goroutine_per_task/after/example_test.go:example}}
```

### worker pool 適合的情況

worker pool 不是沒有用處，只是理由不在「限制同時幾個」。第一種：每個 worker 要持有建立成本高的東西，例如自己的連線或緩衝區。每個任務一個 goroutine 的話，要嘛每個任務都重新建立一次，要嘛得另外做一個連線的池子，那就是 worker pool 換個名字。第二種：來源沒有盡頭，只有 `ctx` 能讓它停：

```go,ignore
{{#include ../../../examples/chapter6/06_worker_pool_or_goroutine_per_task/after/pool.go:consume}}
```

連線在 worker 一開始建立一次，工作做完才關。測試證明建立的連線數等於 worker 數，不是工作數；`ctx` 結束時所有連線都關閉：

```go,ignore
{{#include ../../../examples/chapter6/06_worker_pool_or_goroutine_per_task/after/pool_test.go:once}}
```

```go,ignore
{{#include ../../../examples/chapter6/06_worker_pool_or_goroutine_per_task/after/pool_test.go:endless}}
```

```go,ignore
{{#include ../../../examples/chapter6/06_worker_pool_or_goroutine_per_task/after/example_test.go:consume}}
```

`Consume` 一樣用 `errgroup`，不用自己管 `WaitGroup`。

## 取捨

goroutine 很便宜，它做的事不便宜：連線、記憶體、下游的配額。上限該放在真正稀缺的那個資源上，而不是放在 goroutine 數量上；一萬個任務各自開 goroutine 本身沒事，一萬個同時打下游才有事。需要依權重分配上限時有 `golang.org/x/sync/semaphore`，這裡只提名稱。

「worker 開幾個最好」是效能問題，要量測才回答得了，第 7 章會談。任務小到 goroutine 的啟動成本不能忽略時，也要先量測，再決定要不要改回 worker pool。

before 在這些情況下才是對的：每個 worker 本來就要持有狀態；或來源是一條不會結束的 channel，沒有「一批」可以展開成一個 slice。這時 worker pool 是設計本身，不是為了限制並行而寫。

`Consume` 也有代價：任何一個 worker 出錯，整組都會取消。它回傳之後沒有人再讀 `jobs`，送工作的那一端必須自己同時看 `ctx.Done()`（測試裡的 producer 就是這樣），否則會卡在送出。送的速度比處理快時怎麼辦，本章〈背壓〉會談。

## 重點整理

- 有限的批次，預設用每個任務一個 goroutine 加 `errgroup` 的 `SetLimit`：沒有 channel，也沒有關閉順序要寫對。
- 手寫 worker pool 要管兩條 channel、`WaitGroup` 與關閉順序，其中任何一處錯了都會卡住。
- worker pool 適合每個 worker 持有昂貴的狀態，或消化沒有盡頭的來源；理由是狀態與來源，不是限制並行。
- 並行上限放在稀缺的資源（連線、下游配額）上，goroutine 本身很便宜。
- 並行測試用 `synctest` 或 channel 訊號證明上限與錯誤，不靠 `time.Sleep`。
