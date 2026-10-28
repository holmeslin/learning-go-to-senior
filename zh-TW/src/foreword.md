# 邁向資深 Go 工程師

你好！這是《[從零開始學 Go](https://holmeslin.github.io/learning-go-from-zero/zh-TW/)》的續作。前作教你把 Go 寫出來，這本書要帶你走到下一步：知道**該怎麼取捨**。

套件怎麼切、錯誤怎麼設計、測試要測到什麼程度、效能問題從哪裡查起、服務怎麼上線與維運、code review 該看什麼。這些正是初階與資深工程師之間最明顯的差距，也是前作刻意沒有教的部分。

本書以 **Go 1.27** 為準。

## 這本書寫給誰？

寫給**會前作內容的人**。你不一定要讀過前作，只要下面這份清單裡的東西你都用過，就可以直接開始。

### 前置知識

前置知識是前作全書；以下是最常用到的部分。某一項不太熟的話，點進去補那一塊就好，不必重讀整本前作。

- **語法與基本型別**：變數、控制流程、函式、切片與 map、`struct` 與方法。見[第一部第 1 章（基礎）](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter1/index.html)、[第 2 章（函式、陣列、切片與 map）](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter2/index.html)、[第 3 章（結構、指標與方法）](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter3/index.html)。
- **介面**：隱式實作、介面值、小介面。見[第 4 章（介面）](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter4/index.html)。
- **錯誤處理與 `defer`**：`error`、`%w` 包裝、`errors.Is` 與 `errors.As`、`panic` 與 `recover`。見[第 5 章（錯誤處理與 `defer`）](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter5/index.html)。
- **泛型與迭代器**：型別約束、`slices` 與 `maps`、`iter.Seq`。見[第 6 章（函式值、閉包與泛型）](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter6/index.html)、[第 7 章（迭代器）](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter7/index.html)。
- **套件、模組與測試**：`go.mod`、套件與 `internal`、`go test`、表格驅動測試、benchmark。見[第 8 章（套件、模組與測試）](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/index.html)，從 [`go test`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/09_go_test.html) 開始讀起。
- **並行與 `context`**：goroutine、channel、`sync`、`-race`、`context` 的傳遞慣例。見[第 9 章（並行）](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter9/index.html)、[第 10 章（`context`）](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter10/index.html)。
- **標準庫**：`io.Reader` 與 `io.Writer`、`encoding/json`、`log/slog`。見[第 12 章（進階標準庫）](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter12/index.html)。
- **Web 服務與資料庫**：`net/http`、`httptest`、`database/sql`。見[第 14 章（實戰：Web 服務）](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter14/index.html)、[第 15 章（實戰：`database/sql`）](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter15/index.html)。
- **效能與 runtime 的入門概念**：逃逸分析、GC、`pprof`。見前作的[附錄二](https://holmeslin.github.io/learning-go-from-zero/zh-TW/appendix2/index.html)。

## 這本書教什麼？

重點是**工程判斷**：面對同一個問題，有哪幾種做法、各自的代價是什麼、什麼情況下選哪一種。全書分成七部：

1. 程式碼設計：套件與 API 設計、錯誤設計、抽象的取捨、資料存取的設計
2. 測試：測試策略
3. 並行實務：結構化並行
4. 效能：效能調查流程、runtime
5. 上線與維運：可觀測性、韌性、安全實務、建置與發布
6. 在團隊裡寫 Go：code review、設計文件、事故處理
7. 綜合實戰：一個服務從設計到上線

runtime 只講到「足以做出判斷」的深度，不是要你去讀 Go 的原始碼。軟技能只談 code review、設計文件、事故處理這三項，因為它們最直接影響程式碼怎麼寫、怎麼改。

前作教過的語法與標準庫用法，這裡不會重講，最多用一行連回前作。

## 這本書怎麼讀？

- 大多數的「集」都用同樣的七段結構：**本集目標**、**情境**、**常見寫法**、**問題在哪**、**較好的做法**、**取捨**、**重點整理**。先看到一個真實的情境，再看大家常怎麼寫、這樣寫哪裡出問題，最後才是比較好的做法。有些主題不適合這樣改寫（例如事故處理、設計文件），那幾集會省略中間三段，改用自己的段落。
- 「取捨」這一段很重要。較好的做法不是萬靈丹，有些情況下常見寫法反而是對的，這一段會說清楚。
- 每集的篇幅很短，10 分鐘內讀得完。
- 你可以跳著讀，挑自己現在最需要的主題。不過後面的章節會引用前面的章節，卡住時請回頭看。
- 目錄裡有些章節還沒寫完，以沒有連結的標題呈現。已經可以讀的才有連結。

## 範例在哪裡？

範例分成兩種：

- **單檔範例**：直接寫在書裡，是完整的 Go 程式，可以複製到電腦上用 `go run` 執行。
- **多檔案範例**：放在 [GitHub repo](https://github.com/holmeslin/learning-go-to-senior) 的 `examples/` 目錄，按章、集分資料夾，每集再分 `before` 與 `after` 兩個子目錄，分別是「常見寫法」與「較好的做法」。書中引用的程式碼就是這些檔案的片段。

把 repo 抓下來（需要 Go 1.27），在專案根目錄執行：

```bash
go test -race ./...
```

就能一次跑完所有多檔案範例的測試。只想看某一集的話，把 `./...` 換成那一集的目錄即可。加上 `-tags integration` 會連整合測試一起跑（見第 5 章〈[整合測試與 build tags](chapter5/05_integration_tests_build_tags.md)〉）。有「常見寫法」與「較好的做法」的集，`before` 的程式碼保證能編譯並通過 `go vet`，`after` 的程式碼一定附有測試。

## 搭配 AI 學習

想討論某一集的判斷、把書中的做法套用到自己工作上的程式碼，或想要練習題時，可以請 AI 幫忙。下載本書的原始檔壓縮檔：

> `go-senior-src.zip`：<https://holmeslin.github.io/learning-go-to-senior/zh-TW/go-senior-src.zip>

壓縮檔裡有全書的原始檔與所有範例程式碼。把整個壓縮檔上傳給 AI，跟它說「請先閱讀壓縮檔裡的 `zh-TW/src/GUIDE.md` 再回答我」，再提出你的問題。問題和某一集有關時，順便說你在讀哪一集。這樣 AI 會以書中的判斷為準，並且把取捨講清楚，而不是只丟給你一個「標準答案」。建議使用能力較好的 AI 模型。

---

本書的初稿由 AI 協助撰寫，再經人工審閱修改。
