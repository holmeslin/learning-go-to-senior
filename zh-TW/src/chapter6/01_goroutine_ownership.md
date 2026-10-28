# goroutine 的生命週期歸誰管

## 本集目標

寫下 `go` 之前，先說得出誰負責等這個 goroutine 結束：函式在回傳前等自己開的 goroutine；長期運作的元件提供 `Run(ctx)`，由呼叫它的人決定要不要開 goroutine，也由那個人等它回來。

## 情境

服務在使用者註冊後要寄歡迎信。寄信慢，所以排進佇列，由背景寄出；另外還有一個 `Broadcast`，一次寄給一批收件人。信怎麼寄不是重點，`SendFunc` 由呼叫端傳入。

goroutine 與 `WaitGroup` 前作都教過：〈[goroutine](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter9/01_goroutine.html)〉、〈[`sync.WaitGroup` 與 `wg.Go`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter9/02_waitgroup.html)〉。這集只談一件事：這些 goroutine 該由誰負責。

## 常見寫法

建構函式在背景啟動寄信迴圈，`Close` 關掉佇列；`Broadcast` 每個收件人開一個 goroutine，開完就回傳：

```go,ignore
{{#include ../../../examples/chapter6/01_goroutine_ownership/before/notify.go:notifier}}
```

```go,ignore
{{#include ../../../examples/chapter6/01_goroutine_ownership/before/notify.go:broadcast}}
```

## 問題在哪

兩個方法回傳時，信都還沒寄出去。下面的 `Example` 讓寄信卡在 `release` 上，只是為了讓觀察每次都一樣：

```go,ignore
{{#include ../../../examples/chapter6/01_goroutine_ownership/before/example_test.go:close}}
```

`Close` 回傳了，信還沒寄。呼叫端手上沒有東西可以等：沒有回傳值、沒有 channel、沒有 `Wait`。這時 `main` 若直接回傳，行程跟著結束，那封信永遠不會寄出。`Broadcast` 是同一個問題：

```go,ignore
{{#include ../../../examples/chapter6/01_goroutine_ownership/before/example_test.go:broadcast}}
```

這兩個 `Example` 為了不留下卡住的 goroutine，都得記得呼叫 `Close`；少了它，`New` 開的迴圈會一直卡在佇列上。

這些 goroutine 沒有擁有者，後果是：

- 關機時不知道要等誰。測試呼叫 `Welcome` 之後，也沒辦法知道信寄了沒，只能 `time.Sleep` 碰運氣。
- 錯誤只剩 `log.Printf`。`Broadcast` 的呼叫端不知道哪些收件人失敗（錯誤怎麼傳回呼叫端，本章〈`errgroup`〉會談）。
- `Close` 呼叫兩次、或 `Close` 之後再 `Welcome`，都會 panic；沒有人規定誰能在什麼時候呼叫。

## 較好的做法

規則只有一條：誰開始一個 goroutine，誰負責等它結束。

函式範圍的 goroutine，在函式回傳前等完。`Broadcast` 用 `wg.Go` 開、`Wait` 等，每個 goroutine 把結果寫進自己的欄位（依索引，不需要 mutex），等完再整理成失敗的收件人：

```go,ignore
{{#include ../../../examples/chapter6/01_goroutine_ownership/after/notify.go:broadcast}}
```

```go,ignore
{{#include ../../../examples/chapter6/01_goroutine_ownership/after/example_test.go:broadcast}}
```

測試要證明的是「回傳時真的都寄完了」，而不是「剛好寄完」。`synctest.Wait` 會等到 bubble 裡所有 goroutine 都 durably blocked：三封信都卡在 `release`，這時 `Broadcast` 若已回傳，就是沒等。`synctest` 的規則見前作〈[`testing/synctest`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter9/19_synctest.html)〉與第 5 章〈[測時間與並行：用 `testing/synctest`，還是自己注入時鐘](../chapter5/08_synctest_or_clock.md)〉：

```go,ignore
{{#include ../../../examples/chapter6/01_goroutine_ownership/after/notify_test.go:broadcast}}
```

長期運作的元件，提供會阻塞的 `Run(ctx)`。`New` 不開 goroutine，`Run` 回傳就代表沒有寄信的工作還在進行：

```go,ignore
{{#include ../../../examples/chapter6/01_goroutine_ownership/after/notify.go:run}}
```

要不要在背景跑，由呼叫端決定。開的人也是等的人，兩件事寫在同一處，擁有關係一眼看得出來（`ctx` 怎麼傳進每個會阻塞的地方，本章〈取消與 `context` 的傳遞〉會談）：

```go,ignore
{{#include ../../../examples/chapter6/01_goroutine_ownership/after/example_test.go:run}}
```

測試做同樣的事：第一封卡在寄送中時取消 `ctx`，`Run` 不能回傳；放行之後 `Run` 回傳，第二封沒有寄。把 `Run` 裡的寄信改成 `go n.send(to)`，這個測試就會失敗，因為 `Run` 在信寄完以前就回傳了：

```go,ignore
{{#include ../../../examples/chapter6/01_goroutine_ownership/after/notify_test.go:run}}
```

## 取捨

跟行程一樣長的工作，擁有者就是 `main`：`main` 開 goroutine 跑 `Run`，結束前等它回來。這種工作不需要硬塞進某個函式的範圍，只要有人明確負責。要等多久、等不到怎麼辦，第 10 章〈關機時該等什麼、等多久（graceful shutdown）〉會談；這集的 `Run` 停止時，佇列裡還沒寄的信就丟了，寄完再停是關機的設計。

before 的寫法在「做不做完都無所謂」時可以接受，例如一筆遺失也沒關係的統計，或盡力而為的埋點。即使如此，仍要知道行程結束時它會被直接丟下，測試也無法確定它跑過。

`Run` 把「要不要開 goroutine」留給呼叫端。代價是呼叫端多寫一個 `wg.Go`，好處是它可以直接同步呼叫，測試也不必處理背景執行。

after 還留著兩個沒處理的限制：`Welcome` 在 `Run` 停止之後會一直排隊，佇列滿了就卡住（佇列滿了怎麼辦，本章〈背壓〉會談）；`Broadcast` 每個收件人一個 goroutine，同時進行的數量沒有上限（本章〈`errgroup` 的並行上限〉會談）。

## 重點整理

- 每個 `go` 都要有擁有者：誰開始，誰等它結束。
- 函式範圍的 goroutine，用 `wg.Go` 加 `Wait` 在回傳前等完，結果依索引寫進各自的位置。
- 長期運作的元件提供阻塞的 `Run(ctx)`，由呼叫端決定要不要背景執行並負責等；`Run` 回傳代表沒有工作還在進行。
- 跟行程一樣長的工作，擁有者是 `main`；fire-and-forget 只留給做不做完都無所謂的工作。
- 測試用 `synctest.Wait` 確認「還在等」與「等完了」，不用 `time.Sleep`。
