# 套件命名與 `util` 陷阱

## 本集目標

看到 `util`、`common`、`helpers` 這種名字時，知道它哪裡有問題，並能依內容替套件命名，讓呼叫端的 `套件.名稱` 讀起來順，不重複。

## 情境

你在寫部落格服務的註冊功能：使用者註冊時，要把顯示名稱轉成網址用的代稱（`Holmes Lin` 變成 `holmes-lin`），並檢查代稱是不是保留字；歡迎頁還要把自我介紹截短。轉代稱、截短文字、檢查字串在不在清單裡，這幾個函式不屬於任何一個既有套件，你順手開了一個 `util`。套件與匯出的基本規則，前作〈[套件與目錄](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/02_package_directory.html)〉與〈[匯出](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/03_exported.html)〉已經講過，這集只談名字怎麼取。

## 常見寫法

小函式丟進 `util`，使用者相關的邏輯放在 `user`，型別名稱帶著「User」好讓人一看就懂：

```text
before/
├── util/   Slugify、Truncate、Contains
├── user/   UserService
└── app/    呼叫端
```

```go,ignore
{{#include ../../../examples/chapter1/02_package_naming/before/user/user.go:service}}
```

呼叫端長這樣：

```go,ignore
{{#include ../../../examples/chapter1/02_package_naming/before/app/app.go:welcome}}
```

完整程式碼在 `examples/chapter1/02_package_naming/before/`。

## 問題在哪

先看呼叫端讀到的名字：`user.NewUserService()`，「user」說了兩次；`go doc` 列出來的也是 `user.UserService`。標準庫是 `http.Client`，不是 `http.HTTPClient`，套件名稱本來就是名字的一部分。

再看 `util` 對外說了什麼：

```text
$ go doc ./util
package util

Package util 放各種共用的小函式。

func Contains(list []string, s string) bool
func Slugify(title string) string
func Truncate(s string, n int) string
```

三個彼此無關的函式：一個處理網址、一個處理顯示文字、一個查清單。`util` 這個名字沒有告訴你它負責什麼，所以：

- 讀呼叫端的人看到 `util.Truncate`，要點進去才知道它在做什麼；`text.Truncate` 一眼就懂。
- 之後要加 `FormatDate` 或 `HashPassword`，沒有任何理由拒絕，因為 `util` 本來就什麼都收。
- `Contains` 其實標準庫的 `slices.Contains` 就有。名字叫 `util` 的套件，沒人會去想「這個該不該存在」。
- 專案裡很多套件都會 import `util`；之後 `util` 如果需要它們的型別，就會被迫形成循環相依（本章〈循環相依與依賴方向〉會談）。

## 較好的做法

依內容命名，名稱跟著套件走：

```text
after/
├── slug/   Make
├── text/   Truncate
├── user/   Service、NewService
└── app/    呼叫端
```

- `Slugify` 變成 `slug.Make`：套件名已經說了「代稱」，函式名只要說動作。
- `UserService` 變成 `user.Service`，建構函式叫 `user.NewService`，不再重複。
- `Contains` 刪掉，改用標準庫的 `slices.Contains`。

```go,ignore
{{#include ../../../examples/chapter1/02_package_naming/after/user/user.go:service}}
```

呼叫端少了重複，也看得出每個名字來自哪裡：

```go,ignore
{{#include ../../../examples/chapter1/02_package_naming/after/app/app.go:welcome}}
```

用法直接寫成 `Example`，放在外部測試套件，由 `go test` 比對輸出：

```go,ignore
{{#include ../../../examples/chapter1/02_package_naming/after/user/example_test.go:example}}
```

完整程式碼在 `examples/chapter1/02_package_naming/after/`。

## 取捨

- 套件變多：`slug` 只有一個函式，也要自己一個目錄、一個 import。標準庫也有這種小套件（`path`、`errors`），不算問題，但小專案可能覺得麻煩。
- 改名字是破壞相容性的修改：已經有人在用的 API，不要為了名字好看就改（本章〈匯出名稱與 API 穩定性〉會談）。
- 名字不是越短越好：`text` 這種範圍很大的名字，有可能哪天又變成新的 `util`。如果發現裡面開始放不相干的東西，就是該拆的訊號。

`util` 在這些情況下反而可以接受：

- 只在自己的程式內用、只有兩三個小函式：直接放在用到它們的套件裡，寫成不匯出的函式，連套件都不用開。
- 真的一堆彼此無關的小函式，又沒有地方放：複製幾行，往往比多一個套件、多一條依賴划算。確定多個套件都要用，再放進 `internal/` 底下，並依內容命名。
- 已經有很多呼叫端的舊套件：改名的成本比名字不好看更高，先維持。

## 重點整理

- 套件名稱是呼叫端讀到的名字的一部分，`user.UserService` 這種重複，把「user」拿掉就好：`user.Service`。
- `util`、`common`、`helpers` 沒有說明套件負責什麼，所以什麼都能往裡面放。
- 先問函式屬於誰：屬於某個套件就放進去，標準庫已經有就直接用，真的無關才依內容另開套件。
- 少量、只有自己用的小函式，寫成不匯出的函式，或複製幾行，比開 `util` 好。
- 已經有人依賴的名字，改名是破壞相容性的修改，不要只為了好看而改。
