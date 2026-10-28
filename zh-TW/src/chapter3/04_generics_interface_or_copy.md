# 什麼時候用泛型、什麼時候用介面、什麼時候複製貼上

## 本集目標

看到兩三段長得像的程式碼時，判斷該合併成泛型、合併成介面，還是乾脆複製；並認得「約束越寫越長」是合併得太勉強的訊號。

## 情境

你在寫報表服務的 `ledger` 套件，裡面有四段長得很像的程式：加總訂單金額、加總退款金額、加總一份 `[]int`、加總一份 `[]float64`。每一段都是走訪一遍、把數字加起來。差別在型別，還有「哪些要算」：訂單排除取消的，退款只算已核准的。

「行為不同用介面、做法相同用泛型」這個基本判斷，前作附錄二〈[介面 vs 泛型的取捨](https://holmeslin.github.io/learning-go-from-zero/zh-TW/appendix2/e_interface_vs_generics.html)〉已經講過，泛型與約束的語法，前作〈[型別約束](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter6/09_type_constraint.html)〉也教過。這集談的是第三個選項：複製貼上，以及怎麼認出前兩個選項用過頭了。

## 常見寫法

四段迴圈看起來只差型別，合併成一個泛型 `Total` 順理成章。可是約束是一路長出來的：

- 第一版只有 `Number`，`Total[N Number](xs []N)` 剛好夠加總 `[]int` 與 `[]float64`。
- 訂單與退款不是數字，`Total` 得改成收「任何有金額的東西」：多一個介面 `Entry[N]`，多一個型別參數 `E`。
- 取消的訂單、未核准的退款不能算，`Entry` 再多一個 `Counts()`。

```go,ignore
{{#include ../../../examples/chapter3/04_generics_interface_or_copy/before/ledger.go:constraint}}
```

訂單與退款各自補上這兩個方法：

```go,ignore
{{#include ../../../examples/chapter3/04_generics_interface_or_copy/before/ledger.go:adapters}}
```

完整程式碼在 `examples/chapter3/04_generics_interface_or_copy/before/`。

## 問題在哪

合併之後算出來的數字沒有錯，問題是每個呼叫端都得遷就 `Entry`：

- `Order` 與 `Refund` 多了四個只為 `Total` 存在的方法。更糟的是兩個 `Counts` 看起來是同一件事，其實是兩條不相干的規則，一個看有沒有取消，一個看有沒有核准，只是被塞進同一個方法名。
- 純數字也得先包起來，因為 `[]int` 沒有方法：

```go,ignore
{{#include ../../../examples/chapter3/04_generics_interface_or_copy/before/ledger.go:wrappers}}
```

別處傳來的 `[]float64` 沒辦法直接轉型成 `[]ledger.GB`（編譯器不允許），只能一個一個包。四次呼叫的結果在 `// Output:`，由 `go test` 逐字驗證：

```go,ignore
{{#include ../../../examples/chapter3/04_generics_interface_or_copy/before/example_test.go:example}}
```

- `Entry` 與 `Total` 是匯出的，約束就是 API。日後哪個呼叫端需要再多一個方法，所有實作者都得跟著改，這是第 1 章〈[匯出名稱與 API 穩定性](../chapter1/04_exported_api_stability.md)〉談過的「介面加方法」。

換來的，只是省下每個不到十行的迴圈。

## 較好的做法

逐段問：這幾段到底哪裡一樣？

加總純數字的部分，做法完全相同，只差元素型別。泛型就是用在這種地方，而且約束只有一行聯集：

```go,ignore
{{#include ../../../examples/chapter3/04_generics_interface_or_copy/after/ledger.go:sum}}
```

呼叫端不必包裝，進去什麼型別，出來還是什麼型別，連 `time.Duration` 都行：

```go,ignore
{{#include ../../../examples/chapter3/04_generics_interface_or_copy/after/example_test.go:duration}}
```

標準庫的 `slices` 就是這種情況：`slices.Max`、`slices.Contains` 對任何元素型別做法都一樣，前作〈[`slices` 與 `maps` 套件](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter6/15_slices_maps_packages.html)〉介紹過。

訂單與退款長得像，規則卻不同，也會各自改變：今天退款只算已核准的，明天可能要扣手續費，訂單不會跟著變。這種情況退回複製，各寫各的迴圈：

```go,ignore
{{#include ../../../examples/chapter3/04_generics_interface_or_copy/after/ledger.go:totals}}
```

規則和簽名寫在同一個地方，讀的人不用先跳到 `Entry`、再跳到 `Counts` 才知道怎麼算。測試鎖住的是各自的規則，包括兩個旗標的零值意思相反：

```go,ignore
{{#include ../../../examples/chapter3/04_generics_interface_or_copy/after/ledger_test.go:rules}}
```

四次呼叫算出同樣的數字，呼叫端簡單多了：

```go,ignore
{{#include ../../../examples/chapter3/04_generics_interface_or_copy/after/example_test.go:example}}
```

第三個選項是介面：不同型別要放進同一個集合、各自算出自己的金額，這是行為不同，才輪到它（基本判斷見前面連結的附錄二）。這四段程式沒有這個需求，所以 after 裡沒有介面。完整程式碼在 `examples/chapter3/04_generics_interface_or_copy/after/`。

## 取捨

複製不是免費的：

- 修 bug 要每一份都修。金額日後改成 `int64`，或要加溢位檢查，`OrderTotal` 與 `RefundTotal` 兩份都得改，漏掉一份沒有編譯器會提醒你。能做的是每一份都有測試，同一個行為在每一份的測試裡都鎖一次。
- 份數會增加。兩份是少量；到了第三、第四份，又總是一起改，就該重新考慮抽出來。幾份才抽，本章〈過早抽象的代價〉會談。

泛型在這些情況下才是對的：

- 演算法完全相同，只差型別。訊號是約束很短（`comparable`、`cmp.Ordered`，或上面一行的 `Number`），而且函式不需要知道元素的任何領域概念。`slices` 就是這樣的例子。
- 反過來，約束越長越是警訊：多了方法、多了型別參數、呼叫端得為它改型別或包裝資料。多半是把「規則不同」的程式硬湊成「做法相同」，退回複製。

before 也不是一律不對。如果 `Amount` 與 `Counts` 本來就是這些型別該有的行為，別的地方也會用到，實作它們就不算額外負擔，約束再長一點也撐得住。

泛型與介面的執行效能差異，第 7 章〈效能調查流程〉會談；這集的判斷不依據效能。

判斷的方法：把兩段程式疊起來，標出不一樣的地方。只差型別，用泛型；差在行為，用介面；差在規則，而且會各自改，就複製。

## 重點整理

- 長得像不等於做法相同：先標出差別在型別、行為還是規則。
- 只差型別、約束很短時，用泛型；`slices` 是現成的例子。
- 約束長出方法或型別參數，或讓呼叫端為它改型別、包裝資料，是合併得太勉強的訊號，退回複製。
- 複製要付代價：每一份都得修，所以每一份都要有測試；一起改的份數變多時，重新考慮抽出來。
- 行為不同才用介面；判斷的基本原則在前作附錄二。
