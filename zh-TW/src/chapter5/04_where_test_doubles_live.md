# 測試替身放在哪裡

## 本集目標

多個套件的測試都要替身來取代同一個依賴時，判斷替身該由提供者附上測試用套件，還是各自在使用端寫一個小 stub。

## 情境

`kv` 是團隊共用的小型 key-value 套件，`DirStore` 把每個值存成一個檔案。兩個套件用它：`notes` 把一行字附加到筆記後面，`drafts` 在編輯時自動存草稿、之後還原。兩邊的 key 都是使用者打的標題。

`kv` 後來加了一條規則：key 就是檔名，所以不能含 `/`：

```go,ignore
{{#include ../../../examples/chapter5/04_where_test_doubles_live/before/kv/kv.go:validkey}}
```

兩個使用端都在自己的套件宣告要用的介面（第 3 章〈[在使用端定義介面](../chapter3/01_define_interface_at_consumer.md)〉），測試時換上 fake。fake、stub、mock 怎麼選，本章〈fake、stub、mock 的選擇〉談過，這集只談替身放在哪裡。

## 常見寫法

每個使用端把 fake 抄一份，放在自己的測試檔裡。`drafts` 這份是 `kv` 加規則之前寫的：

```go,ignore
{{#include ../../../examples/chapter5/04_where_test_doubles_live/before/drafts/fake_test.go:fake}}
```

`notes` 的作者撞到新規則，在自己那份補了檢查；`drafts` 的作者沒有人通知。`kv` 自己的測試只測 `DirStore`。完整程式碼在 `examples/chapter5/04_where_test_doubles_live/before/`。

## 問題在哪

`drafts` 的測試全部通過。同一個 `Autosave`、同一個標題 `Q3/plan`，分別接上這份 fake 與真的 `DirStore`。`// Output:` 是實際輸出，由 `go test` 逐字比對：

```go,ignore
{{#include ../../../examples/chapter5/04_where_test_doubles_live/before/drafts/example_test.go:example}}
```

fake 說成功，真的 `DirStore` 拒絕。使用者標題裡一打斜線，每次自動存檔都失敗，測試卻一片綠。

`drafts` 的作者沒有寫錯什麼。複製出去的 fake 和提供者之間沒有任何連結，規則一變，每一份都要有人記得手動改，而會去改的只有碰到問題的那一位。第 3 章〈[介面何時是多餘的](../chapter3/03_redundant_interface.md)〉看過替身與真實實作不一致；第 4 章〈[測試資料層：fake 與真實資料庫](../chapter4/07_fake_or_real_database.md)〉用契約測試讓 fake 不走樣，但那裡 fake 和真實實作在同一個套件。這裡 fake 有很多份，散在不同套件，契約測試沒有地方可放。

## 較好的做法

由提供者附上測試用套件 `kvtest`，命名比照標準庫的 `net/http/httptest`（前作〈[`httptest`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter14/09_httptest.html)〉）、`testing/fstest`、`testing/iotest`。裡面放兩樣東西：fake，和任何實作都要通過的契約。

```go,ignore
{{#include ../../../examples/chapter5/04_where_test_doubles_live/after/kv/kvtest/kvtest.go:fake}}
```

`Fake` 用 `kv.ValidKey` 判斷 key，規則只有一份，不必再抄。契約 `TestStore` 和 `fstest.TestFS`、`iotest.TestReader` 同型：收一個造出空 store 的函式，逐項檢查行為。

```go,ignore
{{#include ../../../examples/chapter5/04_where_test_doubles_live/after/kv/kvtest/kvtest.go:testfunc}}
```

保證 fake 與真實實作一致的，是提供者自己的測試：同一組契約，真的 `DirStore` 跑一次，`Fake` 跑一次。

```go,ignore
{{#include ../../../examples/chapter5/04_where_test_doubles_live/after/kv/kv_test.go:contract}}
```

這兩個測試放在外部測試套件 `kv_test`：`kvtest` 要 import `kv`，內部測試套件再 import `kvtest` 會形成循環相依。我把 `Fake.Put` 裡的 key 檢查拿掉試過，`TestFake` 的四個 invalid key 案例立刻失敗。契約是 fake 與真實實作之間唯一的連結；`kv` 再加規則，就在契約加一項，兩邊一起失敗。

使用端直接用 `kvtest.NewFake()`。使用端的介面是隱式實作的，`*kvtest.Fake` 不需要任何共用的介面。有了可信的 fake，過去沒人想到要測的案例，現在一行就寫得出來：

```go,ignore
{{#include ../../../examples/chapter5/04_where_test_doubles_live/after/notes/notes_test.go:test}}
```

同一個 `Autosave`，同一個標題，這次兩邊的答案一樣：

```go,ignore
{{#include ../../../examples/chapter5/04_where_test_doubles_live/after/drafts/example_test.go:example}}
```

只需要一個固定回答的地方，不必動用 `kvtest`，在使用端寫一個小 stub。`Restore` 遇到儲存失敗要怎麼辦，是 `drafts` 自己的行為；「磁碟壞了」不是 `kv` 契約的一部分，fake 也不該為它多開一個開關：

```go,ignore
{{#include ../../../examples/chapter5/04_where_test_doubles_live/after/drafts/drafts_test.go:stub}}
```

完整程式碼在 `examples/chapter5/04_where_test_doubles_live/after/`。

## 取捨

- **`kvtest` 是公開 API。** `Fake`、`TestStore` 的名稱與行為都是提供者的承諾：改了名字、收緊契約，所有使用端的測試跟著受影響（第 1 章〈[匯出名稱與 API 穩定性](../chapter1/04_exported_api_stability.md)〉）。契約也只涵蓋列出來的案例，沒列的差異照樣沒人發現。
- **只有同一個 module 的套件在用，放進 `internal/`。** 例如 `internal/kvtest`，不必對外承諾（第 1 章〈[什麼時候用 `internal`，什麼時候該公開](../chapter1/03_internal_or_public.md)〉）。要給別的 module 用，才放在公開路徑。
- **不要把替身放在正式程式的套件裡**，例如 `kv.NewFake`。它會出現在 `kv` 的公開 API 與 `go doc` 裡，使用者以為那是支援的功能，正式程式碼也可能開始 import 它。獨立的 `kvtest` 讓「這是測試用的」寫在套件名稱上。
- **before 在這些情況下是對的**：只有一個使用端、依賴很簡單，在自己的測試檔裡抄一份 fake 或寫個 stub，比維護一個公開套件划算。依賴本來就快又穩，直接用真的，連替身都不用（第 3 章〈[介面何時是多餘的](../chapter3/03_redundant_interface.md)〉）。

判斷的方法：有幾個使用端需要同一份行為，而那份行為會不會變？一個使用端、行為固定，抄一份就好；多個使用端、規則會演進，就讓提供者附上 fake 與契約。

## 重點整理

- 同一個依賴的 fake 抄在多個套件裡，沒有東西把它們和真實實作連在一起，規則一變就各自走樣，測試照過。
- 多個使用端共用的 fake，由提供者放在 `xxxtest` 套件（比照 `httptest`、`fstest`），並用提供者自己的契約測試，對 fake 與真實實作各跑一次。
- 只要固定回答、或要模擬依賴失敗時，在使用端寫小 stub；使用端自己宣告介面，所以不需要共用的介面。
- 測試用套件是要維護的公開 API；只有同一個 module 用，放進 `internal/`。
- 不要把替身放進正式程式的套件。
