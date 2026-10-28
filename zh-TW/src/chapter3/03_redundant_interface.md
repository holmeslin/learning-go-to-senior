# 介面何時是多餘的

## 本集目標

看到一個「只有一個實作、方法和實作一對一照抄」的介面時，判斷該不該刪；並分辨哪些介面值得留下：擋在 I/O 邊界上的，和已經有第二個實作的。

## 情境

你在寫會員點數的套件 `loyalty`：消費每 10 元得 1 點，累積到 100 點升級成金卡，升級時要通知會員。會員資料放在記憶體裡，通知則要送到系統外面（寄信、呼叫 webhook）。

review 時有人留言：「每個元件都該有介面，以後才能換實作，測試也才能 mock。」介面與隱式實作，前作〈[`interface`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter4/01_interface.html)〉與〈[隱式實作](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter4/02_implicit_implementation.html)〉已經講過，這集只談這個介面該不該存在。建構函式回傳介面本身的問題，本章〈接受介面、回傳 struct〉會談。

## 常見寫法

每個元件都先宣告介面，再寫唯一的實作，建構函式回傳介面：

```go,ignore
{{#include ../../../examples/chapter3/03_redundant_interface/before/loyalty.go:interfaces}}
```

```go,ignore
{{#include ../../../examples/chapter3/03_redundant_interface/before/loyalty.go:service}}
```

`Store` 也是同一個模子：介面 `Store`，實作 `memStore`，兩個方法逐一照抄。完整程式碼在 `examples/chapter3/03_redundant_interface/before/`。

## 問題在哪

先看測試。`Service` 想隔離 `Store`，有了介面就能放進一個手寫的替身：

```go,ignore
{{#include ../../../examples/chapter3/03_redundant_interface/before/service_test.go:fake}}
```

替身很短，也很容易寫得和真的不一樣：這個 `fakeStore` 對不認識的 ID 回傳零值和 `nil`，真的 `Store` 卻回傳 `ErrNotFound`。同一個 `Earn("ghost", 200)`，換一個 `Store`，結果就不同：

```go,ignore
{{#include ../../../examples/chapter3/03_redundant_interface/before/example_test.go:example}}
```

用替身的測試會通過，但它驗證的是替身。`Store` 只是記憶體裡的 map，沒有網路、沒有磁碟，本來就又快又穩，沒有理由換掉它；介面卻讓我們多寫了一份實作，還得自己保證它和真的一致。

除了替身，這種介面還有幾筆帳：

- **加一個方法要改三處。** 介面、實作，以及每一個替身；簽名一變也是三處。
- **跳到定義只到介面。** 在多數編輯器裡，對 `svc.Earn` 按「跳到定義」會停在介面的方法宣告，要再找實作才看得到程式。
- **「以後可以換」是照猜的形狀做的。** 介面的方法是照第一個實作長出來的。哪天真的換成資料庫，方法多半會多一個 `context.Context`，錯誤的種類也不同，介面照樣得改。

## 較好的做法

刪掉 `Service` 與 `Store` 的介面，直接用 struct，建構函式回傳 `*Service` 與 `*Store`。`Notifier` 留著：

```go,ignore
{{#include ../../../examples/chapter3/03_redundant_interface/after/loyalty.go:service}}
```

`Notifier` 留下來，是因為寄信不能在測試裡真的寄，而且「通知失敗」這種情況，你沒辦法對真的通知服務隨時叫它發生。`Store` 沒有這個問題，直接用真的。

測試因此變簡單：用真的 `Store`，只在 `Notifier` 放替身。

```go,ignore
{{#include ../../../examples/chapter3/03_redundant_interface/after/service_test.go:fake}}
```

```go,ignore
{{#include ../../../examples/chapter3/03_redundant_interface/after/service_test.go:test}}
```

測試沒有變少：金卡通知只送一次、找不到會員、通知失敗時點數仍然入帳、非正數的消費，在 `service_test.go` 裡都寫得出來，跑的是真的 `Store`。同一個 `ghost` 案例，這次不靠替身，錯誤就是真的 `Store` 回傳的：

```go,ignore
{{#include ../../../examples/chapter3/03_redundant_interface/after/example_test.go:example}}
```

測試替身要用 fake、stub 還是 mock、放在哪裡，第 5 章〈測試策略〉會談。完整程式碼在 `examples/chapter3/03_redundant_interface/after/`。

## 取捨

刪掉介面，是把「要不要換實作」的決定往後延，不是取消：

- 測試的範圍變大了。`Service` 的測試也會跑到 `Store` 的程式，`Store` 壞了，`Service` 的測試跟著紅。對記憶體裡的 map，這通常是好事（兩邊接起來真的能動）；當 `Store` 變慢或有副作用，就不是了。
- 哪天 `Store` 真的換成資料庫，那時再補介面，要多花一次重構。不過 Go 的介面是隱式實作，補介面不需要改任何實作：在 `loyalty` 裡宣告一個只含 `Get` 與 `Add` 的介面，`Service` 的欄位與 `NewService` 的參數改用它，`*Store` 自然滿足，呼叫端與測試都不用動（我們在暫存 module 實際試過，這集 after 的測試一行沒改就通過）。
- 如果某個呼叫端想替換 `Service`，同樣是由它在自己的套件宣告只含用到方法的小介面，`*Service` 一行都不用改；這個介面該放哪、該多大，本章〈[在使用端定義介面](01_define_interface_at_consumer.md)〉談過了。

介面值得存在的情況：

- **跨越 I/O 或行程邊界。** 資料庫、網路、檔案系統、時鐘、子行程，真的實作在測試裡慢、不穩，或有副作用，也很難隨時製造它的失敗。`Notifier` 就是這種介面；第 1 章〈[循環相依與依賴方向](../chapter1/07_import_cycle.md)〉裡的 `Notifier` 也是。
- **已經有第二個實作。** 同時要寄 email 與簡訊，或 `Store` 同時有記憶體版與 SQL 版。這時介面描述的是兩者真正共同的事，不是猜的。

before 反而對的情況：`Store` 如果本來就是資料庫，它就在邊界上，介面該留。

判斷的問法：「把這個實作換成別的，測試會變慢、變不穩，或產生副作用嗎？」不會，就直接用真的；會，才為它放一個介面。

## 重點整理

- 認出多餘的介面：只有一個實作、方法和實作逐一照抄、只為了「以後可以換」或「為了測試」。
- 替身是第二份實作，沒人保證它和真的一致；實作本來就快又穩時，替身只是多一個會出錯的地方。
- 刪掉介面後測試仍然寫得出來：用真的實作，替身只放在真正的 I/O 邊界。
- 介面值得存在的情況：跨越 I/O 或行程邊界，或已經有第二個實作。
- 隱式實作讓「等需要時再補介面」幾乎不花成本，不必預先付。
