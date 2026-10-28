# 表格驅動測試的邊界

## 本集目標

看到一張測試表開始長出旗標與 `setup` 函式時，判斷該不該把它拆開；也知道哪些表格不用拆。

## 情境

折價券兌換：`Book.Redeem(code, amount)` 回傳折扣後的應付金額（單位：分）。規則有這幾條：代碼不分大小寫；折扣金額無條件捨去；有些券有低消；單次券兌換過就不能再用；被拒絕的兌換不算用掉一次。

表格驅動測試與 `t.Run` 前作已經教過：〈[表格驅動測試](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/10_table_driven_test.html)〉與〈[`t.Run` 子測試](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/11_subtests.html)〉。這集不教寫法，只談表格什麼時候該拆。被測的程式是：

```go,ignore
{{#include ../../../examples/chapter5/02_table_driven_boundary/before/coupon.go:redeem}}
```

實際的呼叫結果（`// Output:` 由 `go test` 逐字比對）：

```go,ignore
{{#include ../../../examples/chapter5/02_table_driven_boundary/before/example_test.go:example}}
```

## 常見寫法

規則一條條冒出來，每條都想加進同一張表。輸入輸出對不上，就加一個欄位：要先做點準備就加 `setup`，這列不看金額就加 `skipAmount`，要看使用次數就加 `checkUsed`：

```go,ignore
{{#include ../../../examples/chapter5/02_table_driven_boundary/before/redeem_test.go:table}}
```

九列、九個欄位，跑得過。完整程式碼在 `examples/chapter5/02_table_driven_boundary/before/`。

## 問題在哪

九個欄位裡，只有 `name`、`code`、`amount` 每一列都用得到。`wantErr` 四列用，`want` 四列用，`setup`、`skipAmount`、`checkUsed` 各只有一列。迴圈裡的 `if` 把九列分到三條路：檢查金額、檢查錯誤、檢查次數。要知道某一列到底驗證什麼，得在腦中把迴圈對那一列跑一遍：這張表已經是一支小程式，而測試該是比程式更好讀的規格。

最危險的是零值。第 8 列「低消不足不消耗單次券」寫了 `wantUsed: 0`，看起來在斷言「次數是 0」，但 `checkUsed` 沒設，次數根本沒被檢查，只驗證了那次兌換回傳 `ErrMinSpend`。`wantUsed: 0` 與沒寫完全一樣，編譯器與 `go vet` 都不會提醒。

把這個疏漏變成真的 bug：在 `Redeem` 裡把 `b.used[key]++` 搬到檢查低消之前。被拒絕的兌換也算一次，客人湊不到低消，單次券就白白用掉了。把這個修改套用到 `before/coupon.go`，執行 `go test`，測試照過（輸出省略了套件後面的耗時）。

執行結果：

```text
ok  	github.com/holmeslin/learning-go-to-senior/examples/chapter5/02_table_driven_boundary/before
```

## 較好的做法

依行為拆開。形狀一致的案例仍然放在小表格裡：輸入一樣、輸出一樣、迴圈裡沒有 `if`。

```go,ignore
{{#include ../../../examples/chapter5/02_table_driven_boundary/after/redeem_test.go:amount}}
```

被拒絕的案例是另一張小表，輸出換成錯誤：

```go,ignore
{{#include ../../../examples/chapter5/02_table_driven_boundary/after/redeem_test.go:rejects}}
```

需要「先做一件事、再看結果」的案例不放進表格，各自成為一個測試，步驟照順序寫出來：

```go,ignore
{{#include ../../../examples/chapter5/02_table_driven_boundary/after/redeem_test.go:stateful}}
```

before 漏掉的檢查在這裡是明寫的斷言：被拒絕之後再用 500 兌換要成功，`Used` 也要是 1。同樣的 bug 套用到 `after/coupon.go`，只有這個測試失敗，名稱就是失敗的規格（輸出省略了每個測試與套件後面的耗時）。

執行結果：

```text
--- FAIL: TestRejectedRedeemKeepsSingleUseCoupon
    redeem_test.go:95: Redeem(500) after a rejection = 0, redeem WELCOME20: used up; want 400, nil
FAIL
FAIL	github.com/holmeslin/learning-go-to-senior/examples/chapter5/02_table_driven_boundary/after
FAIL
```

`newBook()` 每個測試各建一本，沒有共用的可變狀態。before 的九列在 after 裡逐案都有對應：金額四列、拒絕兩列、單次券、被拒絕不消耗、記一次使用。完整程式碼在 `examples/chapter5/02_table_driven_boundary/after/`。

## 取捨

拆開要付出的：

- 重複。每個測試各自建 `Book`、各自寫一次呼叫與斷言；要看全部案例，得讀好幾個函式，不再是一眼掃完的一張表。
- 命名。每個測試都要取一個說得出規格的名字，取不出來通常表示它還混著兩件事。

表格仍然是最好的寫法的情況：

- 輸入輸出形狀一致。`TestRedeemAmount` 四列，加一個案例只多一行，這比複製一個函式好得多。就算有成功與失敗兩種結果，只要欄位是 `want` 加 `wantErr`、每一列走同一條路，表格也沒有問題。
- 案例很多的邊界值（幾十列的輸入與輸出）。拆成幾十個函式只是把表格攤開，讀起來更累。

判斷的方法：看迴圈裡有沒有 `if tt.某個欄位`，而且它決定的是「做什麼」或「檢查什麼」，不只是「期望值是什麼」。有，就代表這些列不是同一種測試，該拆；沒有，就維持一張表。

所以 before 的毛病不在「用了表格」，而在每多一種案例，就為整張表多加一個只有少數列用得到的欄位。如果它的九列形狀都一樣，那張表就是對的。

## 重點整理

- 表格驅動測試適合「同一條路、不同資料」；案例形狀一致時，表格仍是最好的寫法。
- 欄位只有少數列用得到、迴圈裡用 `if` 決定檢查什麼，就是表格該拆的訊號。
- 零值會讓「忘了開旗標」的列悄悄什麼都不驗證，測試照過，bug 也照過。
- 依行為拆成各自的測試或小表格：測試名稱就是規格，需要先做準備的案例把步驟寫出來。
- 拆開的代價是重複與命名；不要為了拆而拆，一列一個函式不比一張表好讀。
