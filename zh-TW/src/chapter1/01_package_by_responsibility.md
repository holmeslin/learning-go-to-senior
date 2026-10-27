# 套件切分的依據（依職責，不依型別）

## 本集目標

遇到新功能要決定放哪個套件時，改用「這段程式為了誰、因為什麼原因而改」來切，而不是「它是哪一種程式碼」。

標題裡的「型別」，指的是程式碼的種類（model、handler、service），不是 Go 的型別（`type`）。後面的正文都用「種類」。

## 情境

你在寫一個待辦任務的小服務：有 `Task` 資料、新增與完成任務的規則，還有一組 HTTP 端點。專案才剛開始，你照著很多教學的做法，先開了 `models`、`services`、`handlers` 三個套件。套件與目錄的基本規則、匯出規則，前作〈[套件與目錄](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/02_package_directory.html)〉與〈[匯出](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/03_exported.html)〉已經講過，這集只談怎麼切。

## 常見寫法

依種類切：資料放 `models`，邏輯放 `services`，HTTP 放 `handlers`。

```text
before/
├── models/task.go      Task
├── services/task.go    TaskService
└── handlers/task.go    TaskHandler
```

現在要加一個功能：任務可以標記為完成。

```go,ignore
{{#include ../../../examples/chapter1/01_package_by_responsibility/before/models/task.go:task}}
```

```go,ignore
{{#include ../../../examples/chapter1/01_package_by_responsibility/before/services/task.go:complete}}
```

```go,ignore
{{#include ../../../examples/chapter1/01_package_by_responsibility/before/handlers/task.go:handler}}
```

完整程式碼在 `examples/chapter1/01_package_by_responsibility/before/`。

## 問題在哪

「標記完成」是同一個功能，卻散在三個套件：`Done` 欄位在 `models`，規則在 `services`，HTTP 處理在 `handlers`。

接著要加「截止日」：任務可以設定截止日，HTTP 也要能設定。兩種切法各要動哪些檔案：

```text
before/
├── models/task.go      Task 加 DueDate 欄位
├── services/task.go    Add 多收一個截止日
└── handlers/task.go    解析請求裡的截止日
                        （三個檔案，三個套件）

after/
├── task/task.go        Task 加 DueDate、Add 多收一個截止日
└── taskhttp/handler.go 解析請求裡的截止日
                        （兩個檔案；資料與規則只在 task 一處）
```

這三個套件的切法，對應的是程式碼的種類，不是改動的原因。所以同一個功能的改動，常常得橫跨三個套件：

- 一個 PR 常常橫跨三個套件，review 的人要在三個目錄之間跳來跳去。
- `models` 只有資料、沒有任何行為；規則全在 `services`，`handlers` 只是呼叫它。看到 `Task` 的人得去 `services` 才找得到它能做什麼。
- 套件名稱看不出它負責什麼。專案長大後，`services` 容易變成一個什麼都有的大桶子。

## 較好的做法

依職責切：任務本身的資料、規則和儲存放在 `task`；HTTP 是另一個職責，放在 `taskhttp`。

```text
after/
├── task/        Task、Store（資料、規則、儲存）
└── taskhttp/    把 Store 透過 HTTP 提供出去
```

「標記完成」的規則只在 `task` 裡：

```go,ignore
{{#include ../../../examples/chapter1/01_package_by_responsibility/after/task/task.go:complete}}
```

`taskhttp` 只負責把 HTTP 請求翻成對 `task` 的呼叫：

```go,ignore
{{#include ../../../examples/chapter1/01_package_by_responsibility/after/taskhttp/handler.go:handler}}
```

呼叫端的寫法也跟著變順：`task.NewStore()`、`store.Complete(id)`，讀起來就是在說這個套件做的事。這個 API 用起來的樣子，直接寫成 `Example`，放在外部測試套件 `task_test`，`go test` 會真的執行並比對輸出：

```go,ignore
{{#include ../../../examples/chapter1/01_package_by_responsibility/after/task/example_test.go:example}}
```

回頭看上面的截止日：資料與規則的改動都在 `task/task.go` 一個檔案；`taskhttp` 只負責把請求翻成對 `task` 的呼叫，不放規則。如果 HTTP 暫時不開放設定截止日，連 `taskhttp` 都不用動。

完整程式碼在 `examples/chapter1/01_package_by_responsibility/after/`。

## 取捨

依職責切不是沒有代價：

- 功能變多以後，`task`、`user`、`comment` 彼此之間會有依賴，切得不好就會出現循環相依（本章〈循環相依與依賴方向〉會談）。
- 同一個套件裡放了資料、規則和儲存，如果它長到難以閱讀，仍然要再拆，只是這時拆的依據是職責，不是種類。

依種類切在這些情況下反而是對的：

- 程式很小，一個套件就夠，不需要先切。
- 真的是多個功能共用的一層，例如好幾個套件都要用的資料結構或 API 的請求格式，獨立成一個套件，反而能避免互相 import。
- 團隊本來就是依層分工，而且各層的介面很穩定。

判斷的方法：想像下一個要加的功能，要改幾個套件。如果每次都是同一批套件一起改，它們可能該合併。

## 重點整理

- 切套件的依據是「為了誰、因為什麼原因而改」，不是「這是哪一種程式碼」。
- 一個功能散在 `models`、`services`、`handlers` 三處，代表切法對應的是種類，不是改動的原因；`handlers` 只負責呼叫，規則集中在 `services`。
- 依職責切的 `task` 與 `taskhttp`，讓功能的規則集中在一處，呼叫端的名稱也更好讀。
- after 用外部測試套件的 `Example` 示範呼叫端的用法，同時被 `go test` 驗證。
- 很小的程式或真正共用的一層，依種類切也可以。
