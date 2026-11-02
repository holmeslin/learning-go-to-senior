# PR 大小與拆分

## 本集目標

判斷一個 PR 該不該拆、怎麼拆：改名、搬動這類不改行為的變更各自單獨送，功能最後送；PR 該多大沒有固定的行數，看 reviewer 能不能一次看完並做出判斷。

## 情境

候補名單 PR 動了 12 個檔案。reviewer 照本章〈[先看設計，再看細節](01_design_before_details.md)〉的順序，先看改了哪些檔案，這時候就看得出這個 PR 的形狀。在範例目錄 `examples/chapter13/01_design_before_details/` 下執行：

```bash
git diff --no-index --stat base before
```

執行結果：

```text
 {base => before}/errors.go           |  4 +-
 {base => before}/event.go            | 79 ++++++++++++++++++++----------------
 {base => before}/event_test.go       | 19 ++++-----
 {base => before}/example_test.go     | 12 +++---
 {base => before}/handler.go          | 28 +++++++++----
 {base => before}/handler_test.go     |  7 +---
 /dev/null => before/notifier.go      |  8 ++++
 {base => before}/store.go            | 17 ++++----
 {base => before}/store_test.go       |  8 ++--
 /dev/null => before/validate.go      | 12 ++++++
 /dev/null => before/waitlist.go      | 41 +++++++++++++++++++
 /dev/null => before/waitlist_test.go | 79 ++++++++++++++++++++++++++++++++++++
 12 files changed, 235 insertions(+), 79 deletions(-)
```

`{base => before}` 與 `/dev/null =>` 是 git 比較兩個目錄的寫法。這個數字不算嚇人，麻煩的是裡面有三種性質不同的變更，作者一次送出。

## 三種變更混在一起

- **改名（行為不變）**：名額欄位 `capacity` 改成 `seats`，`Capacity()` 改成 `Seats()`，`Attendees()` 改成 `Confirmed()`；`handler.go` 裡回應的欄位也跟著改，JSON 的名稱不變，線上格式不受影響。涉及 `event.go`、`store.go`、`handler.go` 與三個測試檔。
- **搬動（行為不變）**：`normalize` 從 `event.go` 搬到新檔 `validate.go`，內容一字不差。
- **功能**：`Notifier`、候補名單、遞補，涵蓋 `notifier.go`、`waitlist.go`、`waitlist_test.go` 三個新檔，以及其他檔案裡的功能部分。

大部分的檔案不只一種：`event.go` 三種都有，`store.go`、`handler.go` 和三個測試檔是改名加功能，只有 `errors.go`、`handler_test.go` 與新檔是單純的功能或搬動。行為變更就埋在改名之間。底下是 `event.go` 的 diff，指令如下：

```bash
git diff --no-index base/event.go before/event.go
```

輸出有 3 個 hunk。下面只貼第 1 個 hunk 裡 `Event` 型別、`NewEvent` 與 `Seats` 的部分，省略標頭、`@@` 行、開頭的套件說明與 import，以及第 2、3 個 hunk：

```diff
 // Event 是一場有名額上限的活動。零值不可用，請用 NewEvent 建立。
 type Event struct {
 	mu        sync.Mutex
-	capacity  int
-	attendees []string
+	seats     int
+	notifier  Notifier
+	confirmed []string
+	waitlist  []string
 }
 
-// NewEvent 建立名額為 capacity 的活動。
-func NewEvent(capacity int) *Event {
-	return &Event{capacity: capacity}
+// NewEvent 建立名額為 seats 的活動；有人從候補名單遞補時，用 n 通知他。
+func NewEvent(seats int, n Notifier) *Event {
+	return &Event{seats: seats, notifier: n}
 }
 
-// Capacity 回傳名額上限。
-func (e *Event) Capacity() int {
-	return e.capacity
+// Seats 回傳名額上限。
+func (e *Event) Seats() int {
+	return e.seats
 }
```

型別那段刪了 2 行、加了 4 行，沒辦法逐行對應：`seats`、`confirmed` 是改名，`notifier`、`waitlist` 是新欄位。reviewer 得自己比對，才分得出哪幾行需要判斷。這個檔案的 79 行 diff 裡，真正需要想的是 `Register` 與 `Cancel` 的候補分支，其餘是改名與搬動。

## 拆成三個 PR

reviewer 看到這個形狀，不逐行讀，先要求拆分：

> 這個 PR 同時做了三件事：欄位與方法改名（含匯出的 `Capacity()`、`Attendees()`）、把 `normalize` 搬到 `validate.go`、候補名單功能。行為變更埋在改名之間，無法一次判斷。請拆成三個 PR：先改名，再搬動，最後是候補名單。前兩個不改行為，很快能審完。

順序與依據：

1. **改名先送。** 它碰到的檔案最多，最容易和別人在途的分支衝突，越早合併、衝突越少。審起來也快：測試只改名字，斷言沒動，漏改的地方編譯器會找出來。但它不是零風險：`Capacity()` 與 `Attendees()` 是匯出的方法，外部呼叫端會編譯失敗。PR 描述要寫明這一點，reviewer 要問 repo 外還有沒有呼叫端；想讓外部平順遷移，可以保留舊名並標 `// Deprecated:`。第 1 章〈[匯出名稱與 API 穩定性](../chapter1/04_exported_api_stability.md)〉與〈[棄用（deprecation）與相容性](../chapter1/08_deprecation_and_compatibility.md)〉談過這些判斷；這類一旦合併就難以回頭的變更要不要放行，本章〈什麼時候該擋、什麼時候可以放行〉會談。
2. **搬動第二個送。** 只動 `event.go` 與 `validate.go`，reviewer 只要確認刪掉的與新增的是同一段。搬動不能和修改混在一起：diff 會把搬動顯示成整段刪除、整段新增，同時改了內容，reviewer 也看不出來。
3. **功能最後送。** 前兩個合併之後，功能 PR 的 diff 只剩行為變更，注意力可以全放在設計與正確性。R1 要作者改的只有這一個 PR，改名與搬動不受影響，可以先合併。

順序的依據是：不改行為、能機械驗證的先送，需要判斷的最後送。每個 PR 只有一種性質，各自編譯、各自通過測試，功能出問題時可以單獨 revert，不必連改名一起退回去。這和第 12 章〈[`go.mod` 與相依升級](../chapter12/06_go_mod_dependency_upgrade.md)〉一次升一個相依是同一個道理：一次只動一種東西，出問題才知道是哪一步。

三個中間狀態不在 repo 裡（本章只有 `base/`、`before/`、`after/` 三個目錄）。下面是在 repo 外，用 `base/` 與 `before/` 的檔案組合出三個狀態，各自與前一個狀態比較的 `git diff --no-index --stat` 最後一行；三個狀態各自都能編譯、通過 `go test -race`：

```text
改名   6 files changed, 29 insertions(+), 29 deletions(-)
搬動   2 files changed, 12 insertions(+), 10 deletions(-)
功能  11 files changed, 199 insertions(+), 45 deletions(-)
```

功能 PR 還是最大的，但裡面每一行都是行為變更，不必再從改名裡分辨。

## 沒有固定的行數門檻

PR 該多大，數字因團隊而異，本書不給門檻。要問的是：reviewer 能不能一次看完、看懂、做出判斷。

- 同樣 400 行，`go fix` 或改名工具產生的，reviewer 驗證的是範圍與指令：重跑一次、比對輸出，再抽查幾處就行。手寫的 40 行鎖與 channel 修改，可能比它更費時間。
- 機械式的大改動可以大，但要單獨送：PR 描述寫出產生它的指令與範圍，不混任何手改。混進一行手改，重跑指令的捷徑就失效，reviewer 得讀完全部才找得到那一行。
- 反過來，一個 PR 三種性質都有，就算只有 100 行，reviewer 也得分三種方式去看。

## 取捨

拆分要付成本：

- 三個 PR 各要等 review 與 CI。疊在前一個 PR 上開下一個，前一個被要求修改，後面的要跟著 rebase；改名這種碰到很多檔案的變更尤其容易衝突。
- 中間狀態要自己整理。把大 diff 切成三份不是機械動作：同一個檔案裡的改名與功能要分開提交，每一份都要能編譯、通過測試。
- 拆開之後，reviewer 看不到全貌。第一個 PR 的描述要說明它是三個之一、後面還有什麼；PR 描述怎麼寫，本章〈作者的自我審查〉會談。

小的變更不必拆。改一個函式裡的區域變數名稱，順手夾在同一個 PR，拆開的等待比好處大。同一個 PR 分成幾個 commit、讓 reviewer 逐個 commit 看，是成本較低的折衷，但做不到單獨 revert。

## 重點整理

- PR 的大小看變更的種類，以及 reviewer 能不能一次判斷，不看行數；沒有固定的門檻。
- 改名、搬動這類不改行為的變更各自單獨送，功能最後送：從能機械驗證的，到需要判斷的。
- 搬動與修改不要混在一起；改名若動到匯出的名稱，對外是不相容的變更，要在描述寫明。
- 機械式的大改動（`go fix`、改名）可以大，但要單獨送，並寫出產生它的指令。
- 拆分要付等待、衝突與整理中間狀態的成本；小變更不必拆，可以同一個 PR 分成幾個 commit。
