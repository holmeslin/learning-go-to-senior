# 循環相依與依賴方向

## 本集目標

遇到 import cycle 時，先判斷「誰該依賴誰」，再決定要靠調整依賴方向、由使用端定義介面，還是乾脆合併套件，而不是多開一個「共用」套件硬拆。

## 情境

你的服務有 `order`（訂單）與 `notify`（通知）兩個套件。下訂單之後要發通知，所以 `order` 呼叫 `notify`；通知的內容要寫訂單編號、品項與金額，所以 `notify` 的函式收的是 `order.Order`。兩邊都很合理，各自寫完都沒問題，直到第一次一起編譯。套件與目錄的基本規則，前作〈[套件與目錄](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/02_package_directory.html)〉已經講過。

## 常見寫法

兩個套件各自 import 對方：

```text
order/order.go     import ".../notify"    Place 成立訂單後呼叫 notify.OrderPlaced(o)
notify/notify.go   import ".../order"     OrderPlaced(o order.Order) 讀 Order 的欄位
```

```text
order  ──►  notify
  ▲            │
  └────────────┘
```

## 問題在哪

編譯會直接失敗（這是在 repo 外的暫存目錄、Go 1.27 實際重現的輸出，省略了路徑前綴）：

編譯錯誤：

```text
package .../before/notify
	imports .../before/order from notify.go
	imports .../before/notify from order.go: import cycle not allowed
```

Go 要先編譯被依賴的套件，才能編譯依賴它的套件，所以依賴必須排得出先後；兩個套件互相 import，就沒有誰先誰後。錯誤訊息只告訴你環在哪，沒告訴你該拆哪一邊。

這時最順手的辦法，是多開一個 `common`，把兩邊共用的東西都丟進去：

```text
common/    Order、通知相關的型別……（兩邊用到的都放這裡）
order/     import common
notify/    import common
```

環是消失了，但 `Order` 明明是訂單的核心，卻住在一個沒有職責的套件裡；之後每個「兩邊都要用」的東西，也都往 `common` 丟。這和本章〈[套件切分的依據（依職責，不依型別）](01_package_by_responsibility.md)〉裡的 `models` 是同一種切法：依「被誰共用」分，而不是依職責分。

## 較好的做法

先問：這兩個套件，哪一個比較核心、比較不該為了對方而改？訂單規則是核心，怎麼通知是細節。細節依賴核心，核心不依賴細節：`notify` 可以 import `order`，`order` 不該 import `notify`。

可是 `Place` 在執行時確實要通知。解法是由使用端（`order`）定義它需要的介面，只描述「訂單成立後把它交出去」：

```go,ignore
{{#include ../../../examples/chapter1/07_import_cycle/after/order/order.go:notifier}}
```

```go,ignore
{{#include ../../../examples/chapter1/07_import_cycle/after/order/order.go:place}}
```

`notify` 照常 import `order`，並實作這個介面。`order` 沒有 import `notify`，環就不存在了：

```go,ignore
{{#include ../../../examples/chapter1/07_import_cycle/after/notify/notify.go:console}}
```

把兩者接起來的是呼叫端，`order_test` 用 `Example` 示範：

```go,ignore
{{#include ../../../examples/chapter1/07_import_cycle/after/order/example_test.go:example}}
```

連帶的好處是 `order` 的測試不需要 `notify`，用一個假的 `Notifier` 就能驗證「成立訂單會通知」與「通知失敗時的行為」：

```go,ignore
{{#include ../../../examples/chapter1/07_import_cycle/after/order/order_test.go:fake}}
```

為什麼介面放在使用端、介面該多大，第 3 章〈[在使用端定義介面](../chapter3/01_define_interface_at_consumer.md)〉會談；這裡只看它在解環裡的角色：它讓「執行時的呼叫」和「編譯時的 import」不必同一個方向。完整程式碼在 `examples/chapter1/07_import_cycle/after/`。

## 取捨

- 介面多了一層間接：讀 `Place` 的人看不出實際通知了誰，要去呼叫端找接線的地方。
- 呼叫端得負責接線，每個 `order.Place` 都要傳入 `Notifier`。
- 介面要維持得小。`Notifier` 如果長成十個方法，`order` 就又知道太多 `notify` 的事。

這集的做法不是每次都該用：

- 兩個套件如果幾乎每次都一起改，它們可能本來就是同一個職責，合併成一個套件，比加介面更誠實。這時 before 的「兩邊互相需要」就是在告訴你切錯了。
- 如果只是共用一個純資料的型別（沒有行為），把型別移到兩邊都依賴的下層，有時比介面簡單；前提是那一層有清楚的職責，而不是 `common`。
- 如果只有單向需要，直接 import 就好，不必為了「可能以後」先放介面。

判斷的方法：畫出 import 的箭頭，讓它們指向比較穩定、比較貼近業務規則的那一邊。

## 重點整理

- import cycle 是編譯錯誤：套件的依賴必須排得出先後。
- 解環的第一步是決定依賴方向：細節（通知、儲存）依賴核心規則，核心不依賴細節。
- 核心在執行時需要細節的行為時，由使用端定義小介面，讓呼叫方向與 import 方向分開。
- 多開 `common` 往往只是把環藏起來，卻換來一個沒有職責的套件。
- 兩邊真的離不開對方時，合併套件也是合理的答案。
