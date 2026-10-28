# 阻塞、syscall 與排程

## 本集目標

大量 goroutine 同時卡在阻塞式 syscall 時，看得出執行緒數為什麼跟著長，分清它和網路 I/O 的差別，並把同時進行的數量限制在能接受的範圍。

## 情境

匯出服務啟動時要讀入一批資料來源，實際上有幾百個，範例用 64 個。每個來源的讀取都會卡在作業系統裡，直到資料到齊：像是掛載在慢速儲存上的檔案。範例用沒有資料的管線代替，同樣是一個卡住的 `read`。

goroutine 前作教過：〈[goroutine](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter9/01_goroutine.html)〉；G、M、P 三個角色本章〈排程器（GMP）怎麼看〉談過。這集只談 goroutine 卡進 syscall 時，M（執行緒）會發生什麼事。

## 常見寫法

每個來源一個 goroutine，I/O 密集的 goroutine 便宜，所以沒有設上限：

```go,ignore
{{#include ../../../examples/chapter8/07_blocking_syscall_scheduling/before/readall.go:readall}}
```

邏輯沒有問題，完整程式碼在 `examples/chapter8/07_blocking_syscall_scheduling/before/`。

## 問題在哪

下面的 `main` 讓 `ReadAll` 同時讀 64 個來源，等每一條都進了 `read` 之後，用 `threadcreate` profile 的 `Count` 讀出行程到目前為止建立過幾個執行緒。這支程式會同時卡住 64 條讀取，只放在 `main` 套件，CI 只編譯、不執行。

```go,ignore
{{#include ../../../examples/chapter8/07_blocking_syscall_scheduling/after/cmd/threads/main.go:main}}
```

`-mode=net` 讀 TCP 連線，`-mode=pipe` 用 `syscall.Read` 讀阻塞式管線，沒有 `-limit` 就是上面的 before。`os.Pipe` 回傳的管線 Go 會交給 netpoller，所以管線要用 `syscall.Pipe` 才拿得到阻塞式的描述子。

執行結果（某一次）：

```bash
go run ./examples/chapter8/07_blocking_syscall_scheduling/after/cmd/threads -mode=net
go run ./examples/chapter8/07_blocking_syscall_scheduling/after/cmd/threads -mode=pipe
```

```text
mode=net sources=64 limit=0 GOMAXPROCS=14 threads created: 5 -> 5
bytes read: 64
mode=pipe sources=64 limit=0 GOMAXPROCS=14 threads created: 5 -> 67
bytes read: 64
```

平台是 macOS（arm64）、Go 1.27，P 的數量是 14（`GOMAXPROCS=14`）。`threads created: 5 -> 67` 是來源準備好之後、全部卡住時的兩次讀數；兩次輸出是各自獨立的執行，輸出沒有省略任何行。

兩種讀取同樣各有 64 個 goroutine 卡著，執行緒數卻不同：讀網路的沒有增加，讀管線的增加約等於來源數，遠超過 P 的數量。差別在於卡住的是誰。網路連線是非阻塞的，沒有資料時 goroutine 停放在 netpoller（Linux 的 epoll、macOS 的 kqueue），執行緒馬上去跑別的 goroutine。阻塞式 syscall 沒有這條路：執行緒卡在核心裡，回不來。runtime 發現這個執行緒占著 P 太久，會把 P 交給別的執行緒，需要時建立新的，好讓其他 goroutine 繼續跑，每個卡住的 goroutine 因此各占一個執行緒。`GOMAXPROCS` 限制的是同時執行 Go 程式碼的執行緒數，卡在 syscall 的不算。

一般檔案也是阻塞式：它不能註冊到 netpoller，檔案很慢時就是這個形狀。

## 較好的做法

和第 6 章〈[`errgroup` 的並行上限](../chapter6/03_errgroup_limit.md)〉相同，多一行 `g.SetLimit(limit)`，並擋掉小於 1 的 `limit`：

```go,ignore
{{#include ../../../examples/chapter8/07_blocking_syscall_scheduling/after/readall.go:readall}}
```

測試用 `counter` 記錄同時進行的讀取數與最大值，`Example` 先等前 4 個讀取進來，才放行：

```go,ignore
{{#include ../../../examples/chapter8/07_blocking_syscall_scheduling/after/readall_test.go:counter}}
{{#include ../../../examples/chapter8/07_blocking_syscall_scheduling/after/example_test.go:example}}
```

`Example` 證明不了第 5 個不會進來。嚴格的證明交給 `synctest`（第 5 章〈[測時間與並行：用 `testing/synctest`，還是自己注入時鐘](../chapter5/08_synctest_or_clock.md)〉）：`read` 真的卡在 channel 上，`synctest.Wait` 等到所有 goroutine 都卡住，那一刻同時進行的讀取數一定是上限，不需要 `time.Sleep`：

```go,ignore
{{#include ../../../examples/chapter8/07_blocking_syscall_scheduling/after/readall_test.go:limit}}
```

拿掉 `SetLimit` 這一行，這個測試會失敗：同時進行的讀取數與最大值都是 20，不是 4。after 的另一個測試 import before 當基準，確認兩邊對同樣的輸入加總出同樣的結果。同樣的 `main` 加上 `-limit=8`：

執行結果（某一次）：

```bash
go run ./examples/chapter8/07_blocking_syscall_scheduling/after/cmd/threads -mode=pipe -limit=8
```

```text
mode=pipe sources=64 limit=8 GOMAXPROCS=14 threads created: 5 -> 11
bytes read: 64
```

平台與 P 的數量同上。同樣 64 個來源，執行緒數增加的量和上限接近，不再跟來源數走。

## 取捨

- **上限由來源撐得住的量決定，不是 `GOMAXPROCS`。** 卡住的執行緒不算 P，所以這個數字看的是儲存能同時服務幾個、你願意為這類呼叫多付幾個執行緒。太小會拖慢整批，輪數是來源數除以上限、無條件進位。
- **只管這一組。** 同一個行程裡有幾組 `ReadAll` 同時跑，總數是各組上限相加。要整個行程共用，見〈[`errgroup` 的並行上限](../chapter6/03_errgroup_limit.md)〉的取捨；固定數量的 worker 也能達到同樣的上限，見第 6 章〈[什麼時候需要 worker pool，什麼時候每個任務一個 goroutine 就夠](../chapter6/06_worker_pool_or_goroutine_per_task.md)〉。
- **`debug.SetMaxThreads` 是保護網，不是解法。** 預設上限是 10000 個執行緒，超過時 runtime 不回傳錯誤、也不能 `recover`，直接讓程式結束，目的是在它拖垮作業系統之前先停掉自己。同一支程式加上 `-maxthreads=20`，輸出只留前兩行，其餘是堆疊：

  執行結果（某一次）：

  ```text
  runtime: program exceeds 20-thread limit
  fatal error: thread exhaustion
  exit status 2
  ```

  平台同上。限制要放在呼叫端，等碰到保護網就太晚了。
- **`runtime.LockOSThread` 把 goroutine 綁在目前的執行緒上。** 用途是呼叫依賴執行緒狀態的 OS 服務或 C 函式庫；代價是那個執行緒不能跑別的 goroutine，綁住的 goroutine 結束時若沒有 `UnlockOSThread`，執行緒會被終止。大量使用的效果和阻塞式 syscall 類似。
- **cgo 只提概念。** C 函式執行期間同樣占住一個執行緒，成本也比一般函式呼叫高；上限的想法一樣適用，cgo 的用法不在本書範圍。
- **想看實際卡了多久，用 trace。** 第 7 章〈[`runtime/trace`](../chapter7/04_runtime_trace.md)〉談過怎麼收與看。

before 在這些情況下反而可以接受：來源只有幾十個、讀取很快就回來，或是網路 I/O。這時執行緒不會長多少，多一個上限只是多一個要調的數字。多數服務不需要動，先有症狀（執行緒數上升、延遲升高）再處理。

## 重點整理

- 阻塞式 syscall（一般檔案、阻塞式描述子、cgo）占住一個執行緒；網路 I/O 走 netpoller，goroutine 停放，不占執行緒。
- runtime 會替卡住的執行緒補上新的，所以 goroutine 一多，執行緒數跟著長；`GOMAXPROCS` 不限制卡在 syscall 的執行緒。
- 用 `g.SetLimit` 限制同時進行的阻塞式呼叫；測試用 `synctest.Wait` 讓所有 goroutine 卡住，再斷言同時進行的數量與最大值。
- `debug.SetMaxThreads` 碰到上限就讓程式結束，只是保護網；`LockOSThread` 讓執行緒專屬於一個 goroutine，要付出那個執行緒的代價。
