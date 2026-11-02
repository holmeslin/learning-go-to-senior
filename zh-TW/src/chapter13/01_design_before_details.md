# 先看設計，再看細節

## 本集目標

review 一個 PR 時，先判斷方向對不對（責任放在對的地方嗎），再看命名與格式；方向有問題，就先留一則設計層級的意見，暫停逐行 review。

## 情境

虛構的活動報名平台：活動有名額上限，可以報名、取消，另有一個 HTTP handler。作者送出標題為「加入候補名單」的 PR：活動額滿時，新的報名者排進候補；有人取消時，依排隊順序遞補第一位，並用呼叫端提供的 `Notifier` 通知他。

本章全章用這個 PR 當例子。程式碼在 `examples/chapter13/01_design_before_details/`：`base/` 是 PR 之前的主幹，`before/` 是 PR 送出時的樣子，`after/` 是 review 之後的結果，三個目錄各自能編譯、通過自己的測試。這個 PR 動了 12 個檔案（在範例目錄下執行 `git diff --no-index --stat base before`，最後一行是 `12 files changed`），除了功能，還順手改了欄位與函式的名字、搬了一段程式碼，這些本章〈PR 大小與拆分〉會談。

reviewer 打開 PR，面前是 12 個檔案的 diff。

## 常見寫法

從 diff 的第一個檔案、第一行開始，逐行讀，看到什麼就留什麼。這個 PR 收到 13 則意見，全是命名、註解與格式，下面是其中 6 則：

```text
errors.go       ErrFull 的訊息有點長，要不要縮短？
event.go        Seats 的註解可以補上「建構之後不會變」。
waitlist.go     PromoteNext 裡的 cf、wl、nx 看不出是什麼意思，請改名。
handler.go      cancel 開頭那行同時取 id 和 who，拆成兩行比較好讀。
store.go        NewStore 的參數 n 建議寫成 notifier。
event_test.go   測試名稱建議統一成 Test<型別>_<方法>。
```

作者逐則修改，reviewer 再看一輪，核准，合併。

## 問題在哪

13 則意見裡，沒有一則問過：遞補寫在哪裡？

```go,ignore
{{#include ../../../examples/chapter13/01_design_before_details/before/handler.go:cancel}}
```

取消之後呼叫 `ev.PromoteNext()` 的是 handler。`Event.Cancel` 本身只把人移除，空出來的名額沒有人遞補。所以任何不經過這個端點的取消（之後要加的管理員取消、逾期未付款自動取消）都不會遞補：名額空著，候補的人也收不到通知。

作者的測試全部通過，因為它們走的路徑剛好避開問題：一個經過 handler，一個自己呼叫 `PromoteNext`。要證明問題是真的，測試得直接對 domain 取消：

```go,ignore
{{#include ../../../examples/chapter13/01_design_before_details/after/review_test.go:r1}}
```

這個測試放到 `before/` 會失敗。做法是把 `after/review_test.go` 放進 `before/` 的副本（副本在 repo 外，不進本書的 repo）執行；副本只補兩個轉接，不改任何行為：`newEvent` 改成呼叫 before 的 `NewEvent(seats, n)`，另一個檔案宣告 `ErrWaitlisted` 這個名稱，讓同一個檔案裡的其他測試編得過。輸出省略了耗時與最後的套件那一行：

執行結果：

```text
--- FAIL: TestCancelPromotesFirstWaitlisted_R1
    review_test.go:36: Confirmed() = [], want [u2]
    review_test.go:39: Waitlisted() = [u2 u3], want [u3]
    review_test.go:42: Notified() = [], want [u2]
FAIL
```

這不是改名或補註解能補救的：13 則意見全部處理完，遞補還是在 handler。

## 較好的做法

review 的順序：

1. 讀 PR 描述：要解決什麼問題，不做什麼。沒有描述就先問。
2. 看改了哪些套件與公開 API：這個 PR 新增了匯出的 `Notifier`、`PromoteNext`、`MaxWaitlist`，`NewEvent` 與 `NewStore` 的簽名也變了。`PromoteNext` 要求呼叫端「取消之後記得呼叫」，這個匯出本身就是警訊。
3. 看責任放在哪一層：HTTP 的職責是把請求翻成對 domain 的呼叫，規則不放在那裡，第 1 章〈[套件切分的依據（依職責，不依型別）](../chapter1/01_package_by_responsibility.md)〉談過；各層只做自己那一層該做的事，第 2 章〈[在哪一層處理、哪一層只回傳](../chapter2/03_handle_at_boundary.md)〉談過同樣的分工。遞補是名額規則，屬於 `Event`。
4. 方向對了，才看細節：命名、註解、格式、測試的寫法。

方向有問題時，先留一則設計層級的意見，然後暫停逐行 review：

> R1：取消之後的遞補寫在 handler 的取消端點，`Event.Cancel` 本身不遞補。之後任何不經過這個端點的取消（管理員取消、逾期未付款）都會漏掉遞補。建議把遞補移進 `Event`，不論從哪裡取消都由 domain 完成。這則處理好之前，其餘意見先暫停，設計改好再逐行看。

暫停有理由：設計一改，細節意見常常跟著作廢。`after/` 的 `Cancel` 在鎖內決定誰遞補，`PromoteNext` 整個消失，針對它的命名意見（cf、wl、nx）也就沒有對象了：

```go,ignore
{{#include ../../../examples/chapter13/01_design_before_details/after/event.go:cancel}}
```

handler 的取消端點只剩下把請求翻成 `ev.Cancel`；同一個 R1 測試在 `after/` 通過。鎖外通知（`Notify` 放在 `remove` 之外）是另一則意見，不在本集的範圍。

這個 PR 裡其他的意見，包括錯誤契約、持鎖通知、重複報名、匯出的可變變數、缺的測試與命名，本章〈Go 程式碼的 review 檢查清單〉會談怎麼用清單找出來，本章〈什麼時候該擋、什麼時候可以放行〉會談各則要不要擋。

## 取捨

先看設計不是儀式，也有代價：

- reviewer 要先花時間讀懂 PR 的意圖。十幾行的小改動，設計一眼看完，直接逐行看就好。
- 有些 PR 的細節就是重點，例如熱路徑上的小改動、安全修正。這時候逐行看到的，本來就是設計的一部分。
- 逐行 review 在方向已經確認時才是對的做法：設計在動手之前就討論過，PR 剩下的只有細節。設計層級的討論該寫在設計文件裡，PR 不是討論設計的地方，第 14 章〈什麼時候需要寫〉會談。
- 暫停不等於否決。作者可能說明了 reviewer 沒看到的限制，設計意見被說服之後，再繼續逐行看。
- PR 才看出方向不對，成本已經付了：作者寫完了、測完了，而且這時候還是會有人覺得「都做完了，先放行吧」。

## 重點整理

- review 的順序是：PR 描述、改了哪些套件與公開 API、責任放在哪一層，最後才是細節。
- 方向有問題，先留一則設計意見並暫停逐行 review；設計一改，細節意見常常跟著作廢。
- 設計意見也要有證據：R1 用一個直接對 domain 取消的測試，`before/` 失敗、`after/` 通過。
- 作者的測試全綠，不代表設計對：測試走的路徑，可能剛好避開問題出現的路徑。
- 逐行 review 在方向已經確認時才有價值；方向沒確認就挑細節，會把力氣花在即將被改掉的程式碼上。
