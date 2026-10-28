# fake、stub、mock 的選擇

## 本集目標

替依賴挑替身時，依「這個依賴在測試裡要扮演什麼」選：要記得東西用 fake，只要固定回答用 stub，只有互動本身就是行為時才用 mock 斷言呼叫。

## 情境

下單服務的 `Place` 做三件事：向目錄查每一列商品的單價、把訂單存檔、存檔成功後寄一封確認信。三個依賴 `Catalog`、`Store`、`Mailer` 都是使用端宣告的小介面（第 3 章〈[在使用端定義介面](../chapter3/01_define_interface_at_consumer.md)〉），由建構函式注入（第 3 章〈[不用框架的相依注入](../chapter3/07_di_without_framework.md)〉）。測試不想連真的目錄、資料庫與 SMTP，所以要替身。

三種替身的差別在測試怎麼用它：

- **fake**：有狀態的簡化實作。存進去的東西讀得回來，測試靠檢查狀態判斷結果。
- **stub**：只給固定的回答，沒有自己的邏輯，也不記得什麼。
- **mock**：記下被怎麼呼叫，測試斷言呼叫的次數、順序或參數。

## 常見寫法

一個手寫的 mock 同時扮演三個依賴，把每次呼叫連同參數依序記下來，測試最後逐項比對：

```go,ignore
{{#include ../../../examples/chapter5/03_fake_stub_mock/before/orders/mock_test.go:mock}}
```

```go,ignore
{{#include ../../../examples/chapter5/03_fake_stub_mock/before/orders/orders_test.go:test}}
```

被測的 `Place` 如下，完整程式碼在 `examples/chapter5/03_fake_stub_mock/before/`：

```go,ignore
{{#include ../../../examples/chapter5/03_fake_stub_mock/before/orders/orders.go:place}}
```

## 問題在哪

有人發現客戶常把同一個商品放在好幾列，同一個商品查三次價格太浪費，於是重構成 `PlaceCached`：同一個商品只查一次。

```go,ignore
{{#include ../../../examples/chapter5/03_fake_stub_mock/before/orders/orders.go:cached}}
```

訂單與確認信不變。同一個 mock 測試對兩個版本各跑一次，`// Output:` 由 `go test` 逐字比對：

```go,ignore
{{#include ../../../examples/chapter5/03_fake_stub_mock/before/orders/example_test.go:example}}
```

總額一樣、沒有錯誤，mock 卻說「呼叫不同」。這個測試只剩下一個功用：把 `Place` 現在的寫法逐行抄一份。

- 目錄被查了幾次，對呼叫端沒有差別，它不是行為，卻被寫成斷言；重構的人要先改測試，才分得出紅燈是真的壞了還是只是改了寫法。
- 它也沒驗證結果：`Save` 只記下參數，沒有人確認訂單存進去、讀得回來。

## 較好的做法

對每個依賴問：測試要它做什麼？

**要記得東西：用 fake。** 訂單存進去之後讀得回來，測試就直接看狀態：

```go,ignore
{{#include ../../../examples/chapter5/03_fake_stub_mock/after/orders/doubles_test.go:fake}}
```

**只要固定回答：用 stub。** 目錄只需要說某個商品多少錢；要讓存檔失敗，也只需要一個永遠回錯誤的 `Store`：

```go,ignore
{{#include ../../../examples/chapter5/03_fake_stub_mock/after/orders/doubles_test.go:stub}}
```

用 fake 與 stub 的測試只檢查結果：總額與訂單存檔的內容。同一個測試對 `Place` 與 `PlaceCached` 各跑一次（`t.Run` 子測試見前作〈[`t.Run` 子測試](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/11_subtests.html)〉）：

```go,ignore
{{#include ../../../examples/chapter5/03_fake_stub_mock/after/orders/orders_test.go:state}}
```

**互動本身就是行為：才用 mock。** 確認信寄出去，客戶收得到、也數得出收到幾封，這是 `Place` 對外的行為，而且 `Mailer` 沒有狀態可讀回，只能看它被呼叫了什麼：

```go,ignore
{{#include ../../../examples/chapter5/03_fake_stub_mock/after/orders/doubles_test.go:mock}}
```

```go,ignore
{{#include ../../../examples/chapter5/03_fake_stub_mock/after/orders/orders_test.go:interaction}}
```

這兩個測試只斷言重要的部分：恰好一封、寄給客戶、主旨提到訂單編號；以及存檔失敗時一封都不寄。它們不管呼叫的順序，也不管主旨怎麼寫。重構前後的行為一樣，同一組測試兩個版本都過：

```go,ignore
{{#include ../../../examples/chapter5/03_fake_stub_mock/after/orders/example_test.go:example}}
```

`orders.go` 與 before 的內容相同，只是少了 anchor 註解。完整程式碼在 `examples/chapter5/03_fake_stub_mock/after/`。

## 取捨

after 的代價：

- fake 是第二份實作，會和真的實作走樣：`fakeStore` 的編號規則、同一份訂單存兩次會怎樣，都是寫的人憑印象決定的。怎麼讓 fake 與真的實作跑同一組契約測試，第 4 章〈[測試資料層：fake 與真實資料庫](../chapter4/07_fake_or_real_database.md)〉談過，這裡不重講。
- 替身多了，測試檔就多了：三個依賴、四個替身。依賴的方法越少，替身越短，這又回到使用端宣告小介面的好處。替身放在哪裡、怎麼讓各個套件共用，本章〈測試替身放在哪裡〉會談。
- 斷言互動的測試，在你改變對外行為（例如把確認信改成合併通知）時仍然會失敗，這是它該做的事。

mock 的寫法本身沒有錯。`gomock`、`mockery`、`testify/mock` 這類套件能產生或簡化 mock 的程式碼，省下手寫的功夫，但選哪一種替身的判斷不變，本書不使用它們。互動才是行為時，斷言呼叫是對的：

- 呼叫的順序就是契約：先扣款再出貨、先加鎖再解鎖。
- 呼叫的次數就是成本或副作用：對外 API 按次計費、通知恰好一次。
- 依賴沒有狀態可讀回：寄信、推播、稽核紀錄。

就算如此，也只斷言那一項，不要把整段呼叫序列都抄進測試。測行為而不是測實作的原則，本章〈[測什麼、不測什麼](01_what_to_test.md)〉談過。

## 重點整理

- 選替身先問測試要它做什麼：要記得東西用 fake，只要固定回答用 stub，要驗證互動才用 mock。
- 把每次呼叫都記下來再逐項比對的 mock，只是在抄實作：不改變結果的重構也會讓它失敗。
- 結果看得到狀態就檢查狀態；只有寄信、扣款這類互動本身就是行為時，才斷言呼叫，而且只斷言那一項。
- fake 是第二份實作，要靠契約測試保證它和真的一致，不要靠印象。
- `gomock` 這類套件只省手寫的功夫，不替你做選擇。
