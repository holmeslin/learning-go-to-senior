# 過早抽象的代價

## 本集目標

看到共用函式靠一串旗標參數撐住不同的呼叫端時，判斷該把它拆回各個呼叫端、只留下規則真的相同的部分；也判斷什麼時候重複才是比較貴的那一邊。

## 情境

你在維護一個小商店的後端，有四個地方要把「分」顯示成金額：CSV 匯出、收據、帳務報表、發票。最早只有 CSV 匯出，你寫了 `Format(cents)`。之後每個新的顯示場合都來找這個函式，各要一點不一樣的東西。

這集只談共用函式。介面與泛型是另外兩種抽象，本章〈[介面何時是多餘的](03_redundant_interface.md)〉與本章〈[什麼時候用泛型、什麼時候用介面、什麼時候複製貼上](04_generics_interface_or_copy.md)〉各談過一種。補零用的 `%02d`，前作〈[`fmt.Printf` 格式動詞](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter2/05_printf_verbs.html)〉已經講過。

## 常見寫法

三個地方都在把數字轉成金額，抽成一個共用函式很合理。收據來的時候，加了 `symbol` 與 `thousands`；帳務報表來的時候，加了 `parens`，順手把會計報表的習慣「零顯示成 `-`」寫在 `parens` 底下：

```go,ignore
{{#include ../../../examples/chapter3/05_premature_abstraction/before/amount.go:format}}
```

三個呼叫端各自拿到想要的結果，程式碼也只有一份：

```go,ignore
{{#include ../../../examples/chapter3/05_premature_abstraction/before/example_test.go:callers}}
```

完整程式碼在 `examples/chapter3/05_premature_abstraction/before/`。

## 問題在哪

發票上線了。它要 `$`、千分位、負數用括號，三個旗標全是現成的，一行都不用改，`Format(cents, true, true, true)` 就好：

```go,ignore
{{#include ../../../examples/chapter3/05_premature_abstraction/before/example_test.go:invoice}}
```

零元的品項（贈品）在發票上印成了 `$-`。「零顯示成 `-`」是帳務報表加的，寫在加 `$` 之前，當時沒有人會同時開 `symbol` 與 `parens`。每個旗標單獨看都沒錯，組合起來沒有人想過。

修這一行不難，把零的判斷搬到 `symbol` 之後就好。難的是背後的結構：

- 三個旗標有 8 種組合，呼叫端用到 4 種，另外 4 種沒有人驗證過，文件也沒說它們會怎樣。
- 呼叫端看不出自己要什麼。`Format(cents, true, true, true)` 的第三個 `true` 是什麼，得回頭查簽名；三個參數都是 `bool`，順序寫反，編譯器也不會說話。
- 每個呼叫端的需求其實各管各的，卻都擠進同一個函式。改任何一行，都得對照所有組合想一遍，下一個呼叫端來，就多一個旗標、多一層 `if`。

把旗標換成 functional options（第 1 章〈[建構函式與 functional options](../chapter1/05_constructor_functional_options.md)〉），呼叫端會好讀一些，`WithParens()` 比 `true` 清楚。但那解決的是「看不懂參數」，不是「旗標互相牽制」：`$-` 照樣會出現。

## 較好的做法

拆回去：每個呼叫端一個函式，各自寫自己要的格式。拆的步驟是先把 `Format` 複製成每個呼叫端一份、旗標寫死，刪掉走不到的分支，再看剩下哪些部分每份都一樣。before 的 `ExampleFormat` 一路都在，確認三個既有的呼叫端輸出沒變。

四份裡規則真的一樣的只有兩件事：`split` 把「分」拆成正負號、整數與小數，四份都用；`group` 加千分位，收據、帳務報表、發票用，CSV 不用。這兩件事沒有哪個呼叫端想要不一樣的規則，而且都要做對，所以只有它們留在共用的地方：

```go,ignore
{{#include ../../../examples/chapter3/05_premature_abstraction/after/amount.go:shared}}
```

直接用 `/` 與 `%` 處理負數，要小心：Go 的除法往 0 取整，`-50 / 100` 是 0，`-50 % 100` 是 -50，負號不是消失，而是跑進了小數部分；整數部分的 0 沒有正負號，拼回去就不是 -0.50 了。這種邊角寫一次，四個呼叫端一起受益。

四個呼叫端各寫各的，每個函式只有一條路徑，讀函式名就知道輸出長什麼樣：

```go,ignore
{{#include ../../../examples/chapter3/05_premature_abstraction/after/amount.go:callers}}
```

`Ledger` 的零是 `-`，`Invoice` 的零是 `$0.00`，兩邊互不影響。同樣的呼叫，輸出的最後一行不再是 `$-`：

```go,ignore
{{#include ../../../examples/chapter3/05_premature_abstraction/after/example_test.go:example}}
```

測試放在外部套件，每個呼叫端各有一欄預期值：

```go,ignore
{{#include ../../../examples/chapter3/05_premature_abstraction/after/amount_test.go:test}}
```

改收據的格式，只會讓收據那一欄失敗。`-50` 分與最小的 `int64` 另外有測試，鎖住共用的部分。完整程式碼在 `examples/chapter3/05_premature_abstraction/after/`。

## 取捨

拆開的代價看得見：`Receipt` 與 `Invoice` 只差負數的寫法，看久了會想再合併一次。更實際的是，規則真的改了，就得改好幾個地方。公司決定把 `$` 換成 `NT$`，收據與發票各改一行，漏掉哪一個，各自的測試都不會抱怨，因為每一欄鎖的只是自己的格式。

所以判斷的依據是：**這些重複的程式碼，是不是一起變動？**

- 每次需求都是收據與發票一起改，那它們其實是同一件事，該抽出共用的部分。重複的代價是修一個 bug 要改很多處，這時抽象比較便宜。
- 各改各的，像這集的四個呼叫端，重複只是看起來像，抽象反而要用旗標把差異都收進去。
- 「重複第三次才抽」是常見的經驗法則：兩份還看不出哪裡不變，第三份才看得出來。但它是必要條件，不是充分條件。這集有四個呼叫端，規則真的一樣的只有 `split`（四個都用）與 `group`（要千分位的三個用），所以只抽這兩個。

before 在這些情況下反而是對的：旗標彼此獨立、不互相牽制，呼叫端也不會再增加，只有兩三個呼叫端；或者四種格式其實是同一份規格的變體，規則一起變動，一個 `Format` 能讓人一處改完。拆回去之前，先用 `Example` 或測試鎖住既有呼叫端的輸出，才知道拆的時候有沒有改壞。

## 重點整理

- 共用函式每多一個呼叫端就多一個旗標，旗標的組合會比呼叫端多，沒人驗證過的組合就是下一個 bug，像 `$-`。
- 呼叫端的需求各管各的時候，抽象是錯的：函式知道得比任何一個呼叫端都多，改它得對照所有組合。
- 退回的做法：複製成每個呼叫端一份，刪掉走不到的分支，只留下規則真的一樣的部分。
- 判斷該不該抽，看重複的程式碼是不是一起變動：一起變動就抽，各改各的就留著重複。
- 「第三次才抽」是必要條件，不是充分條件：三份必須是同一件事的三份複製。
