# 在使用端定義介面

## 本集目標

決定一個介面該宣告在哪裡、要多大：由使用它的套件，只宣告自己用到的方法；而不是由提供者替所有人預先宣告一個大介面。

## 情境

你的服務有一個 `blob` 套件，包了物件儲存：讀、寫、刪除、列出、複製，共七個方法。另一個套件 `publish` 要把 `drafts/<id>` 的草稿讀出來、去掉前後空白，再寫成 `published/<id>`。儲存是 I/O，你希望 `publish` 的測試不用真的連儲存，所以需要一個替身。

介面與隱式實作，前作〈[`interface`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter4/01_interface.html)〉與〈[隱式實作](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter4/02_implicit_implementation.html)〉已經講過；小介面的基本原則，前作〈[小介面設計](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter4/10_small_interfaces.html)〉也講過。這集談的是：這個介面該放在誰的套件裡。

## 常見寫法

提供者想「讓大家好測試」，就在自己的套件裡把整個儲存宣告成介面，使用端依賴它：

```go,ignore
{{#include ../../../examples/chapter3/01_define_interface_at_consumer/before/blob/blob.go:store}}
```

```go,ignore
{{#include ../../../examples/chapter3/01_define_interface_at_consumer/before/publish/publish.go:publish}}
```

完整程式碼在 `examples/chapter3/01_define_interface_at_consumer/before/`。

## 問題在哪

`Publish` 的簽名說它收一個 `blob.Store`，也就是說它「可能」刪除、列出、複製。實際上它做了什麼？寫一個替身，把被呼叫的方法記下來就知道。這個 `Example` 的 `// Output:` 由 `go test` 逐字比對：

```go,ignore
{{#include ../../../examples/chapter3/01_define_interface_at_consumer/before/publish/example_test.go:fake}}
```

```go,ignore
{{#include ../../../examples/chapter3/01_define_interface_at_consumer/before/publish/example_test.go:example}}
```

`Publish` 只呼叫了 `Get` 和 `Put`，但替身得把七個方法全寫出來，其中五個永遠不會被呼叫，只是為了讓編譯器放行。這個成本會擴散：

- 每個使用 `blob.Store` 的套件，測試都要寫自己的七方法替身，或是搬一份大家共用。
- 提供者替 `Store` 加第八個方法，所有替身都編譯失敗，不管那個使用端用不用得到。第 1 章〈[匯出名稱與 API 穩定性](../chapter1/04_exported_api_stability.md)〉示範過，為什麼介面加方法是破壞相容性的修改。
- 讀 `Publish` 的人，光看簽名分不出它會不會刪東西，得讀完函式本體。

根本原因是提供者在猜：它不知道每個使用端要什麼，只好把全部都放進去。

## 較好的做法

介面交給使用端。`blob` 只匯出具體型別 `Bucket`，想有幾個方法就有幾個；`publish` 自己宣告它要的：

```go,ignore
{{#include ../../../examples/chapter3/01_define_interface_at_consumer/after/publish/publish.go:blobs}}
```

```go,ignore
{{#include ../../../examples/chapter3/01_define_interface_at_consumer/after/publish/publish.go:publish}}
```

`Blobs` 只有兩個方法，替身也只需要寫兩個。同一個 `Example`、同樣的輸出：

```go,ignore
{{#include ../../../examples/chapter3/01_define_interface_at_consumer/after/publish/publish_test.go:fake}}
```

```go,ignore
{{#include ../../../examples/chapter3/01_define_interface_at_consumer/after/publish/example_test.go:example}}
```

`publish_test.go` 另外測了草稿不存在時不會寫入、寫入失敗時錯誤會回傳。這些測試只用這兩個方法的替身就能通過，說明 `Publish` 的依賴就只有這麼多。

真正的 `*blob.Bucket` 呢？`blob` 不認得 `publish`，`publish` 也沒有 import `blob`，是接線的地方把兩者接起來。`*Bucket` 有 `Get` 和 `Put`，就自動滿足 `Blobs`：

```go,ignore
{{#include ../../../examples/chapter3/01_define_interface_at_consumer/after/publish/example_test.go:bucket}}
```

這和第 1 章〈[循環相依與依賴方向](../chapter1/07_import_cycle.md)〉的 `order.Notifier` 是同一個手法。那集用它解環，這集看的是它的另一個好處：每個使用端只依賴自己用到的方法。如果另一個程式只需要 `List` 和 `Delete`，它就宣告另一個兩個方法的介面，同一個 `*Bucket` 兩邊都傳得進去。

完整程式碼在 `examples/chapter3/01_define_interface_at_consumer/after/`。

## 取捨

after 的代價：

- 提供者看不到使用端的介面，所以像 before 那行 `var _ Store = (*Bucket)(nil)` 的檢查，只能放在接線的那一側。`Bucket` 少了某個方法，編譯錯誤會出現在接線處，而不是 `blob` 裡。
- 小介面會在不同的使用端各宣告一份。兩個方法的宣告抄幾份不貴，但要讀程式的人得到使用端去看它需要什麼。
- 如果使用端幾乎用到提供者的每個方法，它宣告的介面就只是把提供者的抄一遍，沒有好處。這時該問的是還需不需要介面，本章〈介面何時是多餘的〉會談。

介面放在提供端，也就是 before 的位置，在這些情況下才是對的：

- **介面本身就是產品。** `io.Reader` 定義在 `io`，因為 `io` 這個套件的工作就是訂出這些約定，讓各式各樣的型別照著實作。`database/sql/driver` 也一樣：它定義的介面，是給資料庫驅動程式實作的。`fs.FS` 是另一個例子，`embed.FS` 與 `testing/fstest` 的 `MapFS` 都是它的實作，`os.DirFS` 回傳的也是。
- **提供者自己就有多個實作。** 假設 `blob` 除了記憶體版，還提供磁碟版，並讓呼叫端依設定選一個，那麼大家共同遵守的介面就是 `blob` 的一部分，應該由 `blob` 宣告。

就算放在提供端，這些介面也都很小：`io.Reader` 與 `fs.FS` 都只有一個方法。判斷的方法是問「這個介面是誰的需求？」：使用端需要它，就由使用端宣告；它是提供者想讓別人照著實作的約定，就由提供者宣告。

替身（fake、stub、mock）怎麼選、放在哪裡，第 5 章會談。

## 重點整理

- 介面通常宣告在使用它的套件，只列出使用端真的會呼叫的方法，而不是由提供者替所有人預先宣告。
- 提供者宣告大介面是在猜使用端的需求：替身要寫滿所有方法，提供者加一個方法，所有替身都壞。
- 提供者只匯出具體型別，隱式實作會讓它自動滿足使用端的介面；接線處的編譯檢查取代了提供者端的檢查。
- 使用端用到的方法越接近提供者的全部，使用端定義介面的好處就越少。
- 介面本身是產品（`io.Reader`、`database/sql/driver`），或提供者自己有多個實作時，介面放在提供端才對。
