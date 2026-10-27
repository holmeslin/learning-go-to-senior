# 什麼時候用 `internal`，什麼時候該公開

## 本集目標

寫套件時，分辨哪些是要承諾給別人用的 API，哪些只是實作細節，並把細節放進 `internal`，而不是因為「反正要給別的套件用」就全部匯出。

## 情境

你維護一個內部共用的快取函式庫 `example.com/cache`：固定容量，滿了就淘汰最久沒用的項目。淘汰順序由一個 LRU 串列管理，你把它拆成獨立的 `lru` 套件，方便自己測試。`internal` 目錄怎麼擋人，前作〈[`internal` 目錄](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/05_internal.html)〉已經講過，這集談的是「什麼東西該放進去」。

## 常見寫法

要被 `cache` 套件 import，`lru` 的名稱就得大寫，所以順手把它做成一個一般的子套件，全部匯出：

```text
cache/
├── go.mod
├── cache.go        Cache（Get、Put、Stats）
└── lru/lru.go      List、New、Get、Put
```

另一個團隊的服務看到 `lru` 就在旁邊，讀起來又剛好合用，直接 import：

```go,ignore
import "example.com/cache/lru"

var sessions = lru.New[string, Session](1000)
```

## 問題在哪

匯出的名稱就是承諾。`lru.List`、`lru.New`、`Get`、`Put` 現在都有你看不到的呼叫端，你想換掉淘汰演算法，或讓 `Put` 回傳被淘汰的項目，改的每一個簽名都可能弄壞別人的建置。你並沒有決定要承諾，但事實上已經承諾了。

收回的方法，就是把 `lru` 搬進 `internal`。搬完之後，對方的程式碼就編不過了。他把 import 路徑改成新位置，得到：

編譯錯誤：

```text
package example.com/app
	main.go:3:8: use of internal package example.com/cache/internal/lru not allowed
```

原本的路徑 `example.com/cache/lru` 則直接找不到套件。這兩個錯誤都是在 Go 1.27 下，用兩個 module（函式庫與呼叫端，以 `replace` 連起來）實際重現的。

所以 `internal` 得在別人還沒依賴的時候就放好；等到有人依賴了，搬進去就是一次破壞相容性的改動（本章〈匯出名稱與 API 穩定性〉會談）。

## 較好的做法

一開始就把實作細節放進 `internal`，只匯出呼叫端需要的：

```text
cache/
├── cache.go                 Cache（Get、Put、Stats）
├── internal/lru/lru.go      List（只有 cache 能 import）
└── example_test.go          外部測試套件，只用公開 API
```

`Cache` 持有 `lru.List`，但這個欄位不匯出，呼叫端看不到、也 import 不到：

```go,ignore
{{#include ../../../examples/chapter1/03_internal_or_public/after/cache/cache.go:cache}}
```

`internal` 的規則是看路徑：`cache/internal/lru` 只有 `cache` 目錄以下的程式碼能 import。測試放在外部套件 `cache_test`，站在呼叫端的位置，只碰得到公開的部分，所以這個 API 的用法直接寫成 `Example`：

```go,ignore
{{#include ../../../examples/chapter1/03_internal_or_public/after/cache/example_test.go:example}}
```

現在要換淘汰演算法，只動 `internal/lru` 和 `cache.go`；所有測試不用改，呼叫端也不會知道。完整程式碼在 `examples/chapter1/03_internal_or_public/after/`。

## 取捨

放進 `internal` 也有代價：

- 別人想要的剛好是你藏起來的東西（例如單獨用 LRU 串列），只能 fork 或複製程式碼。`internal` 放太多，就是把使用者逼到這條路上。
- 它擋的是所有人，包括你自己的其他 module。要跨 module 共用，就得公開。

before 在這些情況下反而是對的：

- `lru` 本來就是要單獨提供的元件：有文件、有自己的測試、你願意為它維持相容。這時公開是有意識的決定，不是意外。
- 所有呼叫端都在同一個 repo、你能在一次修改裡一起改掉，承諾的成本很低。

判斷的問題：半年後，我敢不敢不通知任何人就改掉這個名稱的簽名？敢，就放進 `internal`；不敢，就要認真想它值不值得公開。

## 重點整理

- 匯出就是承諾，包括別人「剛好 import 了」的子套件；細節先放進 `internal`，需要時再公開。
- `internal` 看的是路徑：只有它上一層目錄以下的程式碼能 import；外部 import 會得到 `use of internal package ... not allowed`。
- 事後才把已經公開的套件搬進 `internal`，會弄壞已經依賴它的呼叫端，所以要在別人依賴之前決定。
- after 的測試放在外部套件，只用公開 API，這也驗證了實作細節可以隨時換掉。
- 要提供單獨可用的元件，或呼叫端都在同一個 repo 時，公開也是合理的選擇。
