# 不用框架的相依注入

## 本集目標

元件需要儲存、寄信這類相依時，判斷該讓它自己去找（套件層級變數、`init` 接線），還是由建構函式收進來、在 `main` 一次接好；也能說出物件圖大到什麼程度，手動接線才需要幫手。

## 情境

你有一個註冊功能：新使用者用信箱註冊，記進儲存，再寄一封歡迎信。儲存是記憶體裡的 `Store`，寄信交給 `Mailer`（正式環境走 SMTP）。你要為它寫測試：註冊會寄一封信，同一個信箱註冊兩次只寄一封。

`init` 函式的規則與執行順序，前作〈[`init` 函式](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/06_init.html)〉已經講過，這集只談相依該怎麼交給元件。介面該宣告在哪裡、該多大，本章〈在使用端定義介面〉談過。

## 常見寫法

相依放在套件層級的變數，在 `init` 裡接線，`Register` 直接取用：

```go,ignore
{{#include ../../../examples/chapter3/07_di_without_framework/before/signup/signup.go:globals}}
```

```go,ignore
{{#include ../../../examples/chapter3/07_di_without_framework/before/signup/signup.go:register}}
```

呼叫端只要 `signup.Register("ann@example.com")`，什麼都不必準備，用起來很省事。完整程式碼在 `examples/chapter3/07_di_without_framework/before/`。

## 問題在哪

寫兩個測試，各自註冊 `ann@example.com`。下面兩個 `Example` 的 `// Output:` 是實際輸出（由 `go test` 逐字比對）：

```go,ignore
{{#include ../../../examples/chapter3/07_di_without_framework/before/signup/example_test.go:example}}
```

兩個問題一次看到：

- **換不掉的相依。** 第一個測試只想驗證註冊，卻真的走了正式的寄信實作（輸出的第一行；這裡只印出它會寄給誰，現實裡就是一封寄給真人的信）。`mailer` 是未匯出的套件層級變數，`signup_test` 這種外部測試套件碰不到；要換，只能把測試寫進 `signup` 套件、直接賦值，還得記得換回來，否則影響後面的測試。
- **互相影響的測試。** 兩個測試註冊同一個信箱，第二個卻得到 `already registered`：`store` 是整個套件共用的一份，第一個測試留下的資料還在。單獨執行第二個（`go test -run ExampleRegister_again`），輸出變成寄信加 `<nil>`，測試就失敗了。結果取決於前面跑過誰。

還有一個從輸出看不到的問題：`Register(email string) error` 的簽名完全沒提到儲存和寄信。想知道它依賴什麼、接到哪裡，得讀進函式內部，再去找 `init`。`init` 也不能回傳 `error`，接線出錯時只能 `panic`。

## 較好的做法

每個元件用建構函式收下自己需要的相依，不去找任何套件層級的狀態：

```go,ignore
{{#include ../../../examples/chapter3/07_di_without_framework/after/signup/signup.go:service}}
```

```go,ignore
{{#include ../../../examples/chapter3/07_di_without_framework/after/signup/signup.go:register}}
```

`Register` 的邏輯和 before 一樣，差別只在 `store` 與 `mailer` 從哪裡來：現在是 `Service` 的欄位，讀建構函式的簽名就知道它需要什麼。`Mailer` 是 `Service` 在自己這一端宣告的小介面，只有用到的 `Send`；`Store` 沒有介面，直接用具體型別，不是每個相依都得配一個介面（本章〈介面何時是多餘的〉談過）。第 1 章〈[循環相依與依賴方向](../chapter1/07_import_cycle.md)〉的 `order.Place` 把 `Notifier` 當參數傳入，是同一個做法的函式版。

接線只發生在 `main`：

```go,ignore
{{#include ../../../examples/chapter3/07_di_without_framework/after/signupd/main.go:main}}
```

這就是整張物件圖：`Store` 與 `SMTPMailer` 在最底下，交給 `Service`。順序一目了然，少接一個，編譯就不過。

測試組出同一張圖，只是把真的寄信換成替身：

```go,ignore
{{#include ../../../examples/chapter3/07_di_without_framework/after/signup/signup_test.go:graph}}
```

`fakeMailer` 只實作 `Send` 一個方法就能放進圖裡（前作〈[隱式實作](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter4/02_implicit_implementation.html)〉）。同樣兩個 `Example`：

```go,ignore
{{#include ../../../examples/chapter3/07_di_without_framework/after/signup/example_test.go:example}}
```

兩個都註冊 `ann@example.com`，這次各自組了一張新的圖，第二個不再被第一個影響，也沒有任何一封真的信寄出。`signup_test.go` 裡的測試加了 `t.Parallel()`，同時跑，`go test -race` 照樣通過。替身還讓寄信失敗測得出來：把 `err` 設成 `errors.New("smtp down")`，就能驗證 `Register` 回傳的錯誤包著它；before 的 `smtpMailer` 寫死在 `init` 裡，沒有地方叫它失敗。

完整程式碼在 `examples/chapter3/07_di_without_framework/after/`。

## 取捨

手動接線有成本：

- 每多一個相依，建構函式就多一個參數；深處的元件需要新相依，從它到 `main` 之間的每一層建構函式都要跟著改。
- 圖大了，`main` 的接線會變成一長串：幾十個元件、同一個資料庫連線池與 logger 要交給很多人、建立與關閉都得照相依順序。接線本身難以維護時，才是工具該出場的時候。`wire` 在建置時產生接線程式碼（撰寫時它的 README 標示已不再維護）；`dig`、`fx` 則是在執行時用容器解析相依。本書不示範，也不引入。
- `main` 不能被 import，接線本身不好測。需要時，把接線包成 `run` 函式，`main` 只呼叫它並處理結束碼。

在那之前，手動接線最容易讀：跳轉定義就看得到誰接了誰，漏接是編譯錯誤，不是啟動後才發現。

before 在這些情況下反而可以接受：

- 很小的程式或一次性的工具，相依只有一兩個，全域變數最短最直接；真的要寫測試時，再改也不遲。
- 標準庫自己也有套件層級的狀態，像 `http.DefaultServeMux`。它們是方便入口，不是元件之間傳遞相依的方式；你寫的函式庫別讓呼叫端非用它不可。
- 註冊表式的 `init`，例如前作提過的 `import _ "image/png"`，登記的是「有哪些實作可用」，不是「這個元件用哪一個」；`sql.Open` 仍要呼叫端指定驅動名稱。

判斷的方法：問自己，這個元件的測試能不能自己組出它要的相依？不能，就是該把相依改成從建構函式進來了。

## 重點整理

- 套件層級變數加 `init` 接線，讓測試換不掉相依、讓測試之間共用狀態，函式簽名也看不出它依賴什麼。
- 元件用建構函式收下相依，只在 `main` 接線；整張圖寫在一個地方，讀得到、編譯器幫你檢查。
- 測試用替身組出同一張圖，每個測試各組各的，不會互相影響，也能模擬失敗。
- 相依不必都配介面：跨過 I/O 的才值得有縫，介面宣告在使用端、只含用到的方法。
- 物件圖大到手動接線難以維護時，才考慮 `wire`、`dig`、`fx` 這類工具；在那之前手動接線最容易讀。
