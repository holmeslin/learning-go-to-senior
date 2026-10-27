# 建構函式與 functional options

## 本集目標

依參數的數量與之後會不會繼續增加，在長參數列表、config struct、functional options 三種建構方式之間挑一種，而不是看到可選參數就套 functional options。

## 情境

你寫了一個固定容量的記憶體快取。一開始只要容量；後來有人要限制值的長度，另一個人要在資料被淘汰時記一筆 log。每一次需求都讓 `New` 多一個參數。這個套件有好幾個團隊在用，呼叫端你改不完。

可變參數與函式當參數，前作〈[可變參數 `...`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter6/05_variadic.html)〉與〈[閉包](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter6/03_closure.html)〉已經講過，這集只談什麼時候該用在建構函式上。

## 常見寫法

每多一個設定就多一個參數：

```go,ignore
{{#include ../../../examples/chapter1/05_constructor_functional_options/before/cache/cache.go:new}}
```

呼叫端長這樣：

```go,ignore
{{#include ../../../examples/chapter1/05_constructor_functional_options/before/app/app.go:callers}}
```

完整程式碼在 `examples/chapter1/05_constructor_functional_options/before/`。

## 問題在哪

看呼叫端：`cache.New(1000, 0, nil)` 裡的 `0` 和 `nil` 是什麼，得回頭查簽名。`cache.New(100, 4096, ...)` 的兩個整數如果寫反成 `cache.New(4096, 100, ...)`，編譯器不會有意見，兩個都是 `int`。

更麻煩的是演進。再加一個過期時間，簽名就多一個參數，每個呼叫端都要改，包括完全不在乎過期時間的那些：

```text
cache.New(1000, 0, nil)
cache.New(1000, 0, nil, 0)   // 為了新參數，每個呼叫端都得補一個值
```

參數列表只適合參數少、順序一看就懂、而且之後不太會增加的情況。

## 較好的做法

有兩種常見的改法，各自適合不同情況。

**Config struct**：把設定收進一個 struct，建構函式只收這一個參數。

```go,ignore
{{#include ../../../examples/chapter1/05_constructor_functional_options/after/cacheconf/cacheconf.go:config}}
```

**Functional options**：每個設定是一個函式，建構函式收可變數量的 `Option`。

```go,ignore
{{#include ../../../examples/chapter1/05_constructor_functional_options/after/cache/cache.go:options}}
```

兩種的呼叫端都看得到欄位或選項的名字，加新設定也不會讓既有的呼叫端壞掉。下面兩段 `Example` 放在外部測試套件，`go test` 會比對輸出，可以並排比較兩種建構的寫法：

```go,ignore
{{#include ../../../examples/chapter1/05_constructor_functional_options/after/cache/example_test.go:example}}
```

```go,ignore
{{#include ../../../examples/chapter1/05_constructor_functional_options/after/cacheconf/example_test.go:example}}
```

怎麼選：

- 設定幾乎都是單純的資料（數字、字串、布林），而且零值有合理的意思：用 config struct。零值可用的細節，前作〈[讓零值有用](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter3/15_useful_zero_value.html)〉講過，本章〈什麼時候讓零值可用，什麼時候該強制用建構函式〉會談什麼時候該反過來。
- 選項多、大多數呼叫端一個都不用給，預設值不是零值，或某些選項需要在建構時做檢查與組合：用 functional options。
- 只有一兩個選項，而且不太會增加：參數列表也可以，不必為了「以後可能會加」而先包起來。

完整程式碼在 `examples/chapter1/05_constructor_functional_options/after/`。

## 取捨

Functional options 不是免費的：

- API 面積變大。每個選項都是一個匯出的函式，加上 `Option` 型別，每一個都要寫文件、長期維護。
- 不容易一眼看全。用 `go doc` 看這兩個套件（省略套件標頭與部分輸出）：

```text
$ go doc cache
type Cache struct{ ... }
    func New(opts ...Option) *Cache
type Option func(*Cache)
    func WithCapacity(n int) Option
    func WithMaxValueLen(n int) Option
    func WithOnEvict(fn func(key string)) Option

$ go doc cacheconf Config
type Config struct {
	Capacity    int              // 最多保留幾筆；0 表示預設值 100
	MaxValueLen int              // 值的長度上限；0 表示不限制
	OnEvict     func(key string) // 淘汰一筆資料時的回呼；nil 表示不呼叫
}
```

- Config struct 把每個欄位與它的註解放在同一處；functional options 只看得到函式名稱，預設值要讀 `New` 或文件註解才知道。

Config struct 也有它的代價：零值同時代表「沒設定」與「明確設成 0」，所以 `Capacity: 0` 沒辦法要求容量為 0，只能當成預設值。欄位一多、型別很複雜，或需要在建構時檢查時，才值得換成 functional options。

另外，**必填**的值（例如連線位址）留在參數列比較誠實：把它做成選項，編譯器就不能提醒你漏了。before 的參數列表在參數少而穩定時才是對的，也最簡單。

## 重點整理

- 參數列表只適合參數少、一看就懂、不太會增加的建構函式。
- 設定大多是單純資料、零值有意義：config struct 最簡單，所有欄位在同一處看得到。
- 選項多、預設值不是零值、或需要建構時檢查：functional options，代價是 API 面積變大、預設值藏在程式碼裡。
- 必填的值留在參數列，可選的才進 config 或選項。
- 不要只因為「以後可能會加」就先用 functional options。套件已有外部使用者時，改建構函式的簽名會讓呼叫端壞掉（本章前面的〈匯出名稱與 API 穩定性〉談過這類修改），所以先想一下它會不會長出很多選項。
