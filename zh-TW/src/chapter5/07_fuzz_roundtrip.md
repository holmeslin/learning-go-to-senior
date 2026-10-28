# 什麼時候值得寫 fuzz 測試

## 本集目標

看到一個函式，判斷它有沒有一條「不管輸入是什麼都該成立」的性質，值不值得為它寫 fuzz 測試；也知道 fuzz 在 CI 裡該怎麼跑。

## 情境

你寫了一個連續長度編碼（run-length encoding）：`Encode` 把連續相同的位元組寫成「長度加位元組」，`"aaab"` 變成 `"3a1b"`；`Decode` 把它還原。編碼與解碼互為反函式，所以有一條很明確的性質：任何字串編碼再解碼，都要得到原本的字串。

fuzz 的語法與機制前作已經教過：〈[fuzzing](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/14_fuzzing.html)〉。這集不教寫法，只談什麼時候值得寫。

## 常見寫法

挑幾組輸入，把想得到的情況都放進表格：空字串、單一字元、有重複、長度兩位數的連續段、含中文的字串。編碼的結果與來回轉換各測一次：

```go,ignore
{{#include ../../../examples/chapter5/07_fuzz_roundtrip/before/rle_test.go:table}}
```

全部通過。`Encode` 的核心只有這幾行：

```go,ignore
{{#include ../../../examples/chapter5/07_fuzz_roundtrip/before/rle.go:encode}}
```

## 問題在哪

表格裡的字串都是「字母」，因為寫測試的人想到的輸入都長得像資料。可是編碼的輸入什麼都可能，包括數字。下面的 `Example` 把三個輸入來回轉換一次，並印出結果（`// Output:` 由 `go test` 逐字比對）：

```go,ignore
{{#include ../../../examples/chapter5/07_fuzz_roundtrip/before/example_test.go:example}}
```

`"7"` 編碼成 `"17"`，`Decode` 讀完數字後沒有位元組可接，回傳錯誤。`"1a"` 更糟：編碼成 `"111a"`，`Decode` 把 `111` 當成長度，悄悄回傳 111 個 `a`，沒有任何錯誤。解碼的人分不出哪裡是長度的結尾、哪裡是原字串裡的數字。

這種錯誤用手挑輸入很難想到，但檢查性質的測試不必想到它。把同一條性質寫成 fuzz 測試，再用 `-fuzz` 跑：

```go,ignore
{{#include ../../../examples/chapter5/07_fuzz_roundtrip/after/rle_test.go:fuzz}}
```

把它加到 before 的 `rle_test.go` 後面，執行 `go test -fuzz=FuzzRoundTrip -fuzztime=10s`（輸出省略了：進度行裡的耗時、worker 數與 baseline 數量，失敗輸入檔名裡的雜湊值，重跑提示，以及測試的耗時與套件路徑）：

```text
fuzz: elapsed: 0s, gathering baseline coverage: ...
fuzz: elapsed: 0s, minimizing
--- FAIL: FuzzRoundTrip
    --- FAIL: FuzzRoundTrip
        rle_test.go:52: Decode(Encode("0")) error = decode: count "10" has no byte after it (encoded as "10"), want nil

    Failing input written to testdata/fuzz/FuzzRoundTrip/<hash>
FAIL
```

這是我們實際跑出來的一次。fuzz 是隨機的，每次找到的輸入不一定相同：我們重跑幾次，也看過 `"00"`。所以別把某個特定輸入當成保證；能保證的是「性質寫在那裡，會不斷用新的輸入去檢查它」。

## 較好的做法

修正格式：在長度與位元組之間加一個冒號，標出長度在哪裡結束，`"1a"` 變成 `"1:11:a"`，解碼時不會再混淆。`Encode` 只多寫一個冒號，`Decode` 跟著改成先讀長度、再認冒號。

```go,ignore
{{#include ../../../examples/chapter5/07_fuzz_roundtrip/after/rle.go:encode}}
```

同樣三個輸入，現在都能還原：

```go,ignore
{{#include ../../../examples/chapter5/07_fuzz_roundtrip/after/example_test.go:example}}
```

fuzz 測試留在 after 的測試檔裡，被找到的輸入存在 `testdata/fuzz/FuzzRoundTrip/` 底下，之後每次普通的 `go test` 都會把它當成種子重跑，同一個 bug 不會再回來。這個資料夾放在 after，不放在 before：before 的程式有這個 bug，corpus 檔放進去，預設的 `go test` 就會失敗。

after 的 `go test` 只花一瞬間（輸出省略了耗時）：

```text
ok  	github.com/holmeslin/learning-go-to-senior/examples/chapter5/07_fuzz_roundtrip/after
```

修正之後要再 fuzz 一次，確認修正沒有留下別的漏洞。我們對 after 跑了三次 `-fuzztime=15s`，都沒有找到失敗的輸入；但「沒找到」不是「沒有」，它只代表這段時間內的輸入都通過。

## 取捨

fuzz 要付出的：

- **要有可檢查的性質。** 編碼解碼來回、解析再格式化、正規化兩次等於一次、排序後有序，這類性質不需要知道正確答案。如果檢查的唯一方法是和另一份實作比對，或是自己把預期結果算一遍，fuzz 測試就只是把函式重寫一遍，不值得。
- **CI 預設只跑 seed corpus。** 沒加 `-fuzz` 時，`go test` 只跑 `f.Add` 的種子與 `testdata/fuzz` 的檔案，和普通測試一樣快、一樣確定。真的 fuzzing 要另外安排：排程在夜間跑，或發版、改了解析程式之前手動跑一陣子，找到的輸入再提交進 `testdata/fuzz`。排程的設定第 12 章〈`golangci-lint` 與 CI〉會談。
- **要看結果的人。** fuzz 失敗是排程裡的事，得有人看、有人修，不然只是在耗電。

手挑輸入的表格仍然有它的位置：它說明規格（`"aaabbc"` 該編成什麼），失敗時一眼看得出哪一列壞掉。fuzz 補上的是「想不到的輸入」，不取代表格。

## 重點整理

- 值得 fuzz 的函式有一條不必知道答案就能檢查的性質：來回轉換、解析再格式化、重複套用結果不變。
- 手挑輸入只會挑得到你想得到的；性質加上 `-fuzz` 會用新的輸入反覆檢查，但每次找到的不一定相同，不能保證找得到某一個。
- 找到的輸入存進 `testdata/fuzz`，變成普通 `go test` 就會跑的回歸測試；它屬於已修正的程式，不放在還有 bug 的 before。
- CI 預設只跑 seed corpus；真的 fuzzing 要排程或手動另外跑，並且要有人處理結果。
- 只能和另一份實作比對的函式，不值得寫 fuzz 測試。
