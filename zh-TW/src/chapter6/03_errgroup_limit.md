# `errgroup` 的並行上限

## 本集目標

批次處理資料時，判斷同時進行的 goroutine 該不該設上限：用 `SetLimit` 把上限設在下游撐得住的數字，並分得清「並行上限」和「限流」。

## 情境

匯入服務收到一份客戶名單，每一筆地址都要呼叫下游的地址驗證服務查一次。名單實際上有上萬筆，範例用 20 筆。下游的合約寫明每個客戶同時最多受理 5 個請求，超過的會被拒絕或排隊到逾時（數字是舉例）。

`errgroup` 的 `Go`、`Wait`、`WithContext` 本章〈`errgroup`〉談過；goroutine 與 buffered channel 前作已經教過：〈[goroutine](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter9/01_goroutine.html)〉、〈[buffered channel](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter9/09_buffered_channel.html)〉。這集只談 `SetLimit` 與 `TryGo`。

## 常見寫法

每一筆資料開一個 goroutine，用 `errgroup` 等全部結束、取第一個錯誤：

```go,ignore
{{#include ../../../examples/chapter6/03_errgroup_limit/before/check.go:checkall}}
```

程式很短，錯誤有人收，也沒有 data race。完整程式碼在 `examples/chapter6/03_errgroup_limit/before/`。

## 問題在哪

同時進行的數量等於資料量。下面的 `Example` 用一個假的下游，記錄「同時執行數的最大值」。每個呼叫進來後先卡住，等 20 筆都進了下游才放行；等待靠的是 channel 訊號，不是 `time.Sleep`，所以輸出每次都一樣：

```go,ignore
{{#include ../../../examples/chapter6/03_errgroup_limit/before/example_test.go:example}}
```

合約上限是 5，下游同時收到 20 個請求。名單變成一萬筆，就是一萬個請求，加上一萬個 goroutine 和它們各自占住的連線。下游撐不住時，回來的錯誤是「被拒絕」或「逾時」，看起來像下游的問題，其實是呼叫端的迴圈沒有節制。

## 較好的做法

多一行 `g.SetLimit(limit)`：

```go,ignore
{{#include ../../../examples/chapter6/03_errgroup_limit/after/check.go:checkall}}
```

`SetLimit` 要在第一次 `Go` 之前呼叫。設了之後，同時活著的 goroutine 達到上限時，`Go` 會阻塞，直到有一個結束，所以迴圈停在第 `limit + 1` 筆，不會一口氣建出全部 goroutine。內部就是一個容量為 `limit` 的 buffered channel。

`limit` 小於 1 要先擋掉。文件寫明 `SetLimit(0)` 不允許任何 goroutine 加入，第一個 `Go` 會永遠等下去；負數則是不設上限，正好是 before 的行為。

同樣的假下游，這次的 `Example`，前 5 筆進了下游就放行：

```go,ignore
{{#include ../../../examples/chapter6/03_errgroup_limit/after/example_test.go:example}}
```

這個 `Example` 只等前 5 筆，證明不了「第 6 筆不會進來」。嚴格的證明交給 `synctest` 測試（用法見第 5 章〈[測時間與並行：用 `testing/synctest`，還是自己注入時鐘](../chapter5/08_synctest_or_clock.md)〉）：bubble 裡每個呼叫睡 100 毫秒，用的是假時鐘，不會真的等。

```go,ignore
{{#include ../../../examples/chapter6/03_errgroup_limit/after/check_test.go:limit}}
```

12 筆、一次 5 筆，是 5 + 5 + 2 三輪，所以最大值是 5、耗時剛好 300 毫秒。拿掉 `SetLimit` 這一行，測試失敗，輸出如下（省略了耗時與最後的套件路徑行；同時失敗的 `ExampleCheckAll` 印出的數字取決於排程，也省略）。

執行結果：

```text
--- FAIL: TestCheckAllLimit
    check_test.go:58: peak concurrent calls = 12, want 5
    check_test.go:62: CheckAll() took 100ms, want 300ms
FAIL
```

`SetLimit` 之外還有 `TryGo`：`Go` 滿了會等，`TryGo` 不等，滿了就回傳 `false`：

```go,ignore
{{#include ../../../examples/chapter6/03_errgroup_limit/after/example_test.go:trygo}}
```

回傳 `false` 時函式不會被執行，所以一定要檢查回傳值；滿了要跳過、稍後再試還是回錯誤給呼叫端，由呼叫端決定。滿了之後怎麼辦，本章〈背壓〉會談。

## 取捨

- **n 由下游的容量決定。** 下游的合約、資料庫的連線池，都是上限的來源；怎麼從資料庫的連線上限推出每個副本能開幾條，見第 4 章〈[連線池要設多大、設錯會怎樣](../chapter4/04_pool_size.md)〉。
- **這個上限只管一次 `CheckAll`。** 4 個副本各設 5，下游看到的是 20；同一個副本同時處理 3 個請求、各自呼叫 `CheckAll`，也是 15。需要整個行程共用一個上限，要共用同一個限制，`golang.org/x/sync/semaphore` 可以做到，還能依權重分配，這裡只提名稱。
- **並行上限不是限流。** 上限管「同時幾個」，限流管「每秒幾個」。下游一筆只要 1 毫秒，同時 5 個一秒仍能送出幾千筆；下游一筆要 3 秒、限每秒 10 個，則同時會有 30 個。兩者常常要一起設，限流第 10 章〈`golang.org/x/time/rate` 限流〉會談。
- **太小會拖慢批次。** 輪數是筆數除以上限、無條件進位，耗時約為輪數乘以單筆耗時，上面的 300 毫秒就是這樣來的。
- **`Go` 會擋住呼叫端。** 呼叫端若是請求處理函式，名單很長時，這個呼叫本身就在排隊。

before 在這些情況下反而可以接受：筆數有已知的小上界（例如固定每次最多 10 筆），或下游沒有同時請求的限制，或只是一次性的腳本。這時多一個上限，只是多一個要調的數字。需要長時間消化沒有盡頭的工作、或每個 worker 要持有昂貴的狀態時，不是這一集的問題，本章〈什麼時候需要 worker pool，什麼時候每個任務一個 goroutine 就夠〉會談。

## 重點整理

- 每筆資料一個 goroutine，同時進行的數量等於資料量；下游撐不住時，症狀是它的拒絕與逾時。
- `g.SetLimit(n)` 在第一次 `Go` 之前呼叫，滿了之後 `Go` 會阻塞；`TryGo` 不阻塞，回傳是否啟動。
- `limit` 要驗證：0 讓 `Go` 永遠等下去，負數等於沒有上限。
- 用 `synctest` 的假時鐘，可以確定地驗證同時執行數的最大值與總耗時，不需要 `time.Sleep`。
- n 來自下游的容量，而且只管這一組；「同時幾個」不能取代「每秒幾個」。
