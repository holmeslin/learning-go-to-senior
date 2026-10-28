# 什麼時候用嵌入，什麼時候該明確委派

## 本集目標

判斷一個 struct 該不該嵌入別的型別：嵌入之後，對方匯出的每個方法都成了你的公開 API；只想用其中幾個時，改成具名欄位加明確委派。

## 情境

你在寫 `visits` 套件，統計每個頁面的造訪次數。`Counter` 把次數存在記憶體裡，要能並行使用，所以需要一把鎖。`Store` 是讀寫次數的介面（`Incr`、`Count`、`Reset`），`Counter` 是它的實作；產品要求每次寫入都留下稽核紀錄，所以還有一個可以包住任何 `Store` 的 `Audited`。`Store` 有兩個實作，不是多餘的介面。

嵌入與方法提升的語法，前作〈[嵌入](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter3/12_embedding.html)〉與〈[欄位提升與方法提升](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter3/13_promotion.html)〉已經講過，這集只談什麼時候該用。

## 常見寫法

兩個型別都用嵌入，省下轉呼的程式碼。`Counter` 嵌入 `sync.Mutex`，方法裡直接 `c.Lock()`：

```go,ignore
{{#include ../../../examples/chapter3/06_embedding_or_delegation/before/visits.go:counter}}
```

`Audited` 嵌入 `Store`，只覆寫要稽核的 `Incr`，其餘的方法自動提升：

```go,ignore
{{#include ../../../examples/chapter3/06_embedding_or_delegation/before/visits.go:audited}}
```

完整程式碼在 `examples/chapter3/06_embedding_or_delegation/before/`。

## 問題在哪

先看 `Counter`。`sync.Mutex` 的 `Lock`、`Unlock`、`TryLock` 全部成了 `Counter` 的公開方法，`*Counter` 因此也是一個 `sync.Locker`。呼叫端可以從外面把它鎖住，`Incr` 就得等到他解鎖：

```go,ignore
{{#include ../../../examples/chapter3/06_embedding_or_delegation/before/example_test.go:counter}}
```

鎖本來是 `Counter` 的實作細節，現在變成要維持的承諾。`TryLock` 是 Go 1.18 才加進 `sync.Mutex` 的，嵌入它的型別升級 Go 之後，公開方法也跟著多一個。哪天想換個同步方式，例如改成依路徑分片的鎖，所有呼叫過 `c.Lock()` 的程式都會編譯失敗，見第 1 章〈[匯出名稱與 API 穩定性](../chapter1/04_exported_api_stability.md)〉。

`Audited` 的問題不同。`Reset` 沒有覆寫，直接穿過 `Audited` 進到 `Store`，稽核少了一筆，編譯器不會提醒；`Store` 日後再加寫入方法，也一樣默默繞過。欄位 `Store` 還是匯出的，用 struct literal 建立時漏設它照樣能編譯，要等到呼叫沒覆寫的方法才 panic：

```go,ignore
{{#include ../../../examples/chapter3/06_embedding_or_delegation/before/example_test.go:audited}}
```

## 較好的做法

改成具名欄位，只公開想公開的方法。`Counter` 的鎖放進未匯出的欄位 `mu`，外面看不到：

```go,ignore
{{#include ../../../examples/chapter3/06_embedding_or_delegation/after/visits.go:counter}}
```

```go,ignore
{{#include ../../../examples/chapter3/06_embedding_or_delegation/after/example_test.go:counter}}
```

`Audited` 的欄位也不匯出，由 `NewAudited` 收下；`Incr`、`Count`、`Reset` 各寫一次轉呼，要不要稽核由每個方法自己決定：

```go,ignore
{{#include ../../../examples/chapter3/06_embedding_or_delegation/after/visits.go:audited}}
```

現在 `Audited` 得自己實作 `Store` 的每個方法。少寫一個（例如 `Reset`），把它當 `Store` 用時就是編譯錯誤，不會等到線上才發現漏了稽核：

```go,ignore
{{#include ../../../examples/chapter3/06_embedding_or_delegation/after/example_test.go:audited}}
```

完整程式碼與測試在 `examples/chapter3/06_embedding_or_delegation/after/`。測試涵蓋三件事：`Counter` 不再是 `sync.Locker`、也沒有 `TryLock`；並行呼叫 `Incr` 不會漏算（搭配 `-race`）；`Audited` 的每個寫入方法都留下紀錄，也確實轉給底下的 `Store`。

委派不會讓 nil 消失：傳 `nil` 給 `NewAudited` 還是會 panic，`visits.Audited{}` 這樣的零值也不能用（取捨見第 1 章〈[什麼時候讓零值可用，什麼時候該強制用建構函式](../chapter1/06_zero_value_or_constructor.md)〉）。差別在於漏傳 `store` 變成少一個參數的編譯錯誤，不再是執行時才發現。

## 取捨

明確委派要付出的代價：

- 要自己寫轉呼。`Store` 愈大，`Audited` 要寫的就愈多。
- 已經發布的型別從嵌入改成委派，等於移除公開方法，是破壞相容性的修改；所以這個決定最好在第一版就做。

嵌入在這些情況下是對的：

- 外層型別未匯出，也沒有被匯出的函式回傳到套件外。方法集合沒人看得到，`c.Lock()` 比 `c.mu.Lock()` 短，也沒有人會依賴它。
- 確實要保留完整的方法集合。標準庫的 `sort.Reverse` 內部是未匯出的 `reverse`，嵌入 `sort.Interface` 只覆寫 `Less`；它一定經由 `Reverse` 建立，欄位不會漏設。`bufio.ReadWriter` 嵌入 `*Reader` 與 `*Writer`，就是要兩邊的方法都有。
- 測試替身偷懶時：嵌入介面、只實作用到的方法，沒實作的一被呼叫就 panic，在測試裡這反而能立刻看到。測試替身怎麼選、放哪裡，第 5 章會談。

包裝 `http.ResponseWriter` 這類型別時，嵌入與委派都要小心：包裝後只剩 `ResponseWriter` 介面的三個方法，底下的值即使還實作了 `http.Flusher`，包起來之後型別斷言就拿不到。標準庫的做法是包裝型別提供 `Unwrap() http.ResponseWriter`，呼叫端改用 `http.NewResponseController`，它會順著 `Unwrap` 找到底下的值：

```go,ignore
{{#include ../../../examples/chapter3/06_embedding_or_delegation/after/example_test.go:flusher}}
```

## 重點整理

- 嵌入之後，對方匯出的每個方法都成了你的公開 API，連它以後新增的也算。只想用幾個，就用具名欄位加明確委派。
- 匯出的型別嵌入 `sync.Mutex`，外面就能直接 `Lock`，鎖不再是實作細節；把鎖放在未匯出的欄位。
- 嵌入介面只覆寫部分方法，沒覆寫的會默默穿過去，欄位沒設就在呼叫時 panic；明確委派讓每個方法都是一個決定，漏寫編譯器會講。
- 未匯出的型別、確實要保留完整方法集合的包裝，嵌入是對的。
- 包裝 `http.ResponseWriter` 會遺失 `http.Flusher` 這類可選介面；提供 `Unwrap`，並用 `http.NewResponseController`。
