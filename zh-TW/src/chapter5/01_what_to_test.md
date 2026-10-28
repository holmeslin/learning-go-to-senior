# 測什麼、不測什麼

## 本集目標

決定測試該釘住什麼：測公開 API 看得到的行為，而不是內部怎麼實作；也能分辨哪些東西不值得測，以及什麼時候直接測未匯出的函式才是對的。

## 情境

你的服務有個 `tagset` 套件，管文章的標籤：加進去的標籤不分大小寫、去掉前後空白，空白的標籤直接忽略，列出來要依字母排序。內部用一個 `[]string` 存，每次 `Add` 都從頭掃一遍判斷有沒有重複。

標籤多了之後，你想把它改成排好序的 slice，用二分搜尋找位置。功能完全不變，只是換個內部做法。`go test` 怎麼寫，前作〈[`go test`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/09_go_test.html)〉教過；這集談的是測試要看哪裡。

## 常見寫法

測試和被測的程式放在同一個套件，所以什麼都看得到。要確認 `Add` 有存進去，最直接的做法是看 `s.items`；`indexOf` 有自己的邏輯，也就順手測了：

```go,ignore
{{#include ../../../examples/chapter5/01_what_to_test/before/set.go:set}}
```

```go,ignore
{{#include ../../../examples/chapter5/01_what_to_test/before/set_test.go:test}}
```

這些測試都通過。完整程式碼在 `examples/chapter5/01_what_to_test/before/`。

## 問題在哪

把 `Add` 的 `append` 換成排序插入，`List` 不必再排序，`indexOf` 改用二分搜尋。公開的行為一個字都沒變：

```diff
-	s.items = append(s.items, tag)
+	i, _ := slices.BinarySearch(s.items, tag)
+	s.items = slices.Insert(s.items, i, tag)
```

```diff
-	out := slices.Clone(s.items)
-	slices.Sort(out)
-	return out
+	return slices.Clone(s.items)
```

```diff
-	return slices.Index(s.items, tag)
+	if i, found := slices.BinarySearch(s.items, tag); found {
+		return i
+	}
+	return -1
```

把這個修改套用到 `before/set.go`，執行 `go test`（輸出省略了每個測試與套件後面的耗時）。

執行結果：

```text
--- FAIL: TestAddKeepsInsertionOrder
    set_test.go:25: items = ["go" "rust"], want ["rust" "go"]
--- FAIL: TestAddDuplicateLeavesItemsUnchanged
    set_test.go:36: items = ["go" "rust"], want ["rust" "go"]
--- FAIL: TestIndexOf
    set_test.go:44: indexOf("go") = -1, want 1
FAIL
FAIL	github.com/holmeslin/learning-go-to-senior/examples/chapter5/01_what_to_test/before
FAIL
```

三個測試失敗，但沒有任何一個說得出使用者受到什麼影響：前兩個斷言「依插入順序存放」，這從來不是承諾，`List` 本來就排序；第三個手動塞了一份未排序的 `items`，新的內部做法恰好不接受這種狀態。沒壞的兩個，`TestListIsSorted` 透過 `List` 看結果，`TestAddStoresNormalizedTag` 則只是碰巧只有一個標籤。

更糟的是，這些測試同時漏掉了真的 bug。`Add` 會正規化標籤，`Contains` 卻沒有：

```go,ignore
{{#include ../../../examples/chapter5/01_what_to_test/before/example_test.go:example}}
```

加進去的是 `Go`，問 `Go` 卻說沒有。所有測試都看著 `items`，沒有人從呼叫端的角度問「加了就找得到嗎」。重構時它們壞了，上線時它們不吭聲，兩頭都沒幫上忙。

## 較好的做法

測試放進外部套件 `tagset_test`，只能用匯出的 `New`、`Add`、`Contains`、`Len`、`List`，一個測試描述一件使用者看得到的事：

```go,ignore
{{#include ../../../examples/chapter5/01_what_to_test/after/set_test.go:test}}
```

`TestContainsNormalizesTags` 抓得到 before 的 bug：`Contains("Go")` 沒有先 `normalize`，找不到存成 `go` 的標籤。修法是 `Contains` 先 `normalize` 再找。`Example` 同樣的呼叫，現在輸出一致：

```go,ignore
{{#include ../../../examples/chapter5/01_what_to_test/after/example_test.go:example}}
```

`after/set.go` 的內部已經是重構之後的做法：排序插入加二分搜尋，`List` 不再排序，就是上面那三個修改再加上 `Contains` 的 `normalize`。同樣的六個測試與 `Example` 一個字都沒改，全部通過。測試斷言的是 `List()` 與 `Contains()` 的結果，內部用 slice、map 還是排序後的 slice，它們都不在乎；每個失敗訊息都寫出「得到什麼、預期什麼」，哪個行為壞了一看就知道。外部套件還有一個好處：它想偷看內部也做不到，不必靠自律。

完整程式碼在 `examples/chapter5/01_what_to_test/after/`。匯出的名稱才是對使用者的承諾，這和第 1 章〈[匯出名稱與 API 穩定性](../chapter1/04_exported_api_stability.md)〉是同一件事：承諾以外的東西，應該可以隨時改。

## 取捨

只測公開 API 要付出的：

- 失敗時離原因比較遠。`List()` 不對，你得自己找是 `Add` 還是排序出錯；直接看內部狀態的測試會直接指向出錯的那一行。
- 某些邊界從公開 API 很難走到，或要先搭一大堆前置才走得到。

所以直接測未匯出的函式，在這些情況下是合理的：

- **函式本身有複雜的演算法**，例如解析器的詞法分析、日期區間合併，邊界很多，而公開 API 每走一個邊界都要大費周章。
- **它的輸入輸出就是穩定的契約**，不依賴任何物件的內部狀態，像 `func parseDuration(s string) (time.Duration, error)`。

這種測試放在內部測試套件（`package tagset`，與外部測試同一個目錄），最好獨立一個檔案，讓人一看就知道它綁著實作。重構時它可能得跟著改，這是你選擇的成本。判斷方法：這個函式改名或換做法時，這些測試該不該跟著壞？如果答案是「不該」，就該改測公開 API。注意 before 失敗的三個測試，恰好是依賴物件內部狀態的那種，不是上面那種純函式。

有些東西則不值得測：

- 沒有邏輯的 getter，像 `Len` 只是回傳 `len(s.items)`；它已經被其他測試用到了，單獨寫一個只是複誦實作。
- 標準庫本身的行為。`strings.ToLower` 會轉小寫，不需要你證明；你要測的是「`Add` 有呼叫它」帶來的結果。

替身怎麼選、放在哪裡，本章〈fake、stub、mock 的選擇〉與〈測試替身放在哪裡〉會談。

## 重點整理

- 測試釘住的東西，就是之後不能改的東西：釘行為，內部才可以自由重構；釘實作，每次重構都要陪著改測試。
- 看到測試在斷言內部欄位或未匯出的狀態，先問使用者看得到嗎；看不到，就改成透過公開 API 觀察。
- 只看內部狀態的測試，既擋不住使用者遇到的 bug（`Contains` 沒正規化），也會在無害的重構時壞掉。
- 外部測試套件讓「只用公開 API」由編譯器保證；內部測試套件留給演算法複雜、邊界很多、且輸入輸出穩定的未匯出函式。
- 不值得測：沒有邏輯的 getter、標準庫自己的行為。
