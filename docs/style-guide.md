# 寫作風格指南

撰寫或修改 `zh-TW/src/` 內容前請先讀完。名詞定義見根目錄 `GLOSSARY.md`，全書章節見 `docs/outline.md`。

## 讀者與語氣

- 讀者會前作《從零開始學 Go》的內容，通常有一兩年的 Go 經驗，但不一定讀過前作。他們會寫 Go，缺的是「該怎麼取捨」。
- 繁體中文原生撰寫（台灣用語：程式、函式、變數、物件、預設、檔案、介面、迴圈、陣列、記憶體），不是翻譯腔。
- 口語、友善、短句，用「我們」「你」。不要說「這很簡單」「顯然」「不過如此」。
- 不寫成維基百科或規格書。先講直覺，再上程式碼。
- 中英文之間加半形空格；程式碼、識別字、指令用反引號。
- 不參考、不改寫、不翻譯其他教學的文字（特別是 learning-rust-from-zero，其文字為 CC BY-NC-ND）。

## 每一集的格式

檔名 `##_slug.md`，放在章目錄 `chapterN/`，標題與 `SUMMARY.md` 的標題一致。每集由七段組成，順序固定，每段用 `##` 標題：

```markdown
# <與 SUMMARY 相同的標題>

## 本集目標

一兩句話：讀完這集你會做出什麼判斷。

## 情境

一個讀者可能遇到的具體狀況：要解決什麼問題、誰在用、限制是什麼。

## 常見寫法

多數人第一直覺會寫的樣子（before）。

## 問題在哪

這個寫法在什麼條件下會出問題，最好能用測試、benchmark 或錯誤訊息指給讀者看。

## 較好的做法

改過之後的樣子（after），並說明為什麼比較好。

## 取捨

較好的做法要付出什麼代價；常見寫法在什麼情況下反而是對的。

## 重點整理

- 3～5 條，條列本集真正教了什麼。
```

「常見寫法」「問題在哪」「較好的做法」三段是「改寫模式」。不是每個主題都適合改寫，例如事故處理、設計文件、code review，或 runtime 的概念介紹。可省略與不可省略的規則：

| 段落 | 規則 |
| --- | --- |
| 本集目標 | 不可省略 |
| 情境 | 可省略，但只在「本集目標」已經把狀況講清楚時 |
| 常見寫法、問題在哪、較好的做法 | 可省略，但要三段一起省，不能只留一兩段 |
| 取捨 | 不可省略 |
| 重點整理 | 不可省略 |

- 省略「改寫模式」三段時，用一或多個 `##` 段落取代，標題自訂，位置在「情境」之後、「取捨」之前。
- 「取捨」要誠實：說清楚 before 在什麼情況下才是對的，不要把「較好的做法」寫成唯一正解。
- 每章的 `README.md` 只有 `# <章名>` 加一兩段導言，說明這章要學什麼、為什麼重要，不列目錄。

## 篇幅

- 每集約 **80～150 行 markdown**，不含 `{{#include}}` 引入的程式碼（`{{#include}}` 那一行算一行），讀者應在 **10 分鐘內讀完**。
- 超過就拆成兩集，不要硬塞：一集只講一個判斷或取捨。

## 前置知識與引用

- 前置知識是**前作全書**。前作教過的主題不重講：只用一行連到前作的對應集，之後的正文只講判斷與取捨。
- 本書**後面的章節可以引用前面的章節**，用相對連結並寫明章與集的名稱（例如「第 2 章〈錯誤設計〉」）。
- 不要倒過來：不得不提到後面的內容時，用一句話帶過，不要讓讀者必須先讀後面的集才看得懂本集。後面的章寫「第 X 章會談」；同一章裡排在後面的集寫「本章〈集名〉會談」（例如「本章〈循環相依與依賴方向〉會談」）。
- 同一章裡回頭引用前面的集，寫成「本章〈[集名](NN_slug.md)〉」，用相對連結；不要省略「本章」，也不要寫「前面的」。
- 本書不沿用前作的「進度邊界」與「固定句型」：不需要避開 `SUMMARY.md` 中排在後面的概念，也不要求讀者先照抄、之後才解釋；範例直接用寫給有經驗的人看的寫法。

## 程式碼區塊

- 一律 `gofmt` 格式（Tab 縮排）。
- 單檔的小範例寫在 markdown 的 ```` ```go ```` 區塊，必須是**完整可編譯的 `package main`**，會被 `tools/checkcode` 執行：`go vet` 必須通過，執行必須以結束碼 0 結束。
- 需要特殊處理時，在 info string 加標註。標註清單見根目錄 `README.md`；值裡面不能有空白，也不能有逗號（`checkcode` 以逗號切開 info string）。`go,ignore` 只用在 `{{#include}}` 引入的程式碼（見下節）與確實不完整的片段，例如 `go.mod` 以外的檔案內容。

- 輸出區塊前一行固定寫「執行結果：」（輸出不固定時寫「執行結果（某一次）：」），編譯錯誤寫「編譯錯誤：」。
- 被 include 的 `Example` 若 `// Output:` 就在程式碼片段裡，輸出已經是片段的一部分，不需要再加「執行結果：」。
- 單檔範例的 `go.mod` 由 `tools/checkcode/deps/go.mod` 提供（`go 1.27`，`require` 一開始是空的）。某一集真的需要第三方套件時才加入，並遵守下方「第三方套件」。
- 書中呈現錯誤訊息、錯誤鏈或程式輸出時，優先寫成 `Example` 函式，用 anchor 把程式碼與 `// Output:` 註解一起 include 進書中，由 `go test` 逐字驗證，不要手動貼 `text` 區塊（見「before/after 的驗證標準」）。
- 只有 `Example` 無法表達的輸出（例如程式因 panic 結束）才手動貼 ```` ```text ```` 區塊，**必須與實際執行結果完全一致**（自己跑過再貼）。輸出順序不固定的（map 走訪、goroutine），要在文中說明。手動貼的輸出若含每次執行都不同的細節（檔案路徑、位址、goroutine 編號），可以省略這些細節，但文中要明說省略了什麼。
- 終端機指令用 ```` ```bash ````，`go.mod` 內容用 ```` ```gomod ````（這兩種不會被驗證）。
- 展示編譯錯誤時，錯誤訊息用 ```` ```text ```` 貼出 Go 1.27 實際的訊息，可省略檔案路徑前綴。
- 不要用 `go,ignore` 逃避驗證：能寫成完整程式就寫成完整程式；跨檔案、需要測試或需要多個套件的範例，改用下一節的多檔案範例。

## 多檔案範例與 include

套件、介面、測試這類範例一個檔案放不下，就放進 repo 根目錄 module 底下的 `examples/`，由 `go vet ./...` 與 `go test -race ./...` 驗證，書中再用 mdBook 的 include 引入片段，所以書上顯示的就是被驗證過的程式碼。

- 目錄：`examples/chapterN/##_slug/before/` 與 `examples/chapterN/##_slug/after/`。章與集的目錄名和 `zh-TW/src/` 底下的一致。
- 命名：用 `before` 與 `after`，不要用其他名字。每個目錄是 module 內獨立的 Go 套件，單獨就能 build。範例需要多個套件時，在 `before`、`after` 底下再按套件分子目錄（例如 `after/task/`、`after/taskhttp/`）。
- 整個 repo 只有根目錄一個 Go module，所有範例共用相依；不要在 `examples/` 裡再放 `go.mod`。
- 不在 markdown 裡複製貼上範例程式碼。要引入的片段，在原始檔用 anchor 標出來：

```go,ignore
// ANCHOR: handler

// handler 處理 /tasks。
func handler(w http.ResponseWriter, r *http.Request) {
	// ...
}

// ANCHOR_END: handler
```

- 在 markdown 用 `{{#include}}` 引入，路徑相對於該 markdown 檔，並放在 ```` ```go,ignore ```` 區塊裡（`checkcode` 不會抽出它；真正的驗證由 `go vet` 與 `go test` 負責）：

````markdown
```go,ignore
{{#include ../../../examples/chapter1/01_slug/before/handler.go:handler}}
```
````

- anchor 名稱用描述內容的英文小寫（`handler`、`test`），同一個檔案內不可重複，也不要留下沒被引用的 anchor。
- `// ANCHOR: x` 與 `// ANCHOR_END: x` 各佔一行，並且與相鄰的宣告或文件註解之間**空一行**。anchor 緊貼著宣告時，它會變成該宣告的文件註解，`go doc` 與 IDE 都會顯示 `ANCHOR: x`。
- 引入的片段因此會多一行開頭與結尾的空白，這可以接受；檔案本身必須維持 `gofmt` 乾淨，不要為了去掉空白而改格式。
- CI 與 `README.md` 的本機建置指令會檢查：`go doc -all` 列出任何 `examples/` 套件時，輸出不可含 `ANCHOR`。
- 只引入需要讓讀者看的片段，不要整檔貼上；完整程式碼讓讀者去 `examples/` 看。

## before/after 的驗證標準

- **before**：只需要能編譯並通過 `go vet`。它是用來展示問題的，不要求有測試，也不要求沒有 bug。before 可以有 `Example`（或測試），用來把「問題在哪」的輸出鎖住，例如不好的錯誤訊息實際長什麼樣子；`go test -race` 一樣要通過。
- **after**：必須有測試，`go test -race` 要通過。測試要能說明「較好的做法」好在哪裡，而不只是跑過一次。
- **before 套件本身無法編譯時**（例如 import cycle、從外部 import `internal`）：不建 `before` 目錄，以免 `go vet ./...` 讓建置流程失敗。改用套件結構圖（`text` 區塊或簡單的目錄樹）呈現 before，並以「編譯錯誤：」加 `text` 區塊貼出 Go 1.27 實際的錯誤訊息，可省略檔案路徑前綴。錯誤訊息由實作者在 repo 外的暫存目錄手動重現，不進建置流程。`after` 照常是有測試的多檔案範例。
- **只有呼叫端壞掉時**（破壞相容性的修改、刪除舊名稱）：before 套件本身能編譯，照常建 `before` 目錄並用 anchor 引入。壞掉的呼叫端完整程式與它的編譯錯誤，用 `text` 區塊貼出（錯誤前標「編譯錯誤：」），錯誤裡的行號要和貼出的呼叫端一致，由實作者在 repo 外的暫存目錄以同一份呼叫端重現。
- 書中的錯誤訊息、錯誤鏈與程式輸出，來自 `Example` 的 `// Output:`，用 anchor 連同 `// Output:` 註解一起 include（anchor 範圍要包含整個 `Example` 函式）。只有 `Example` 無法表達時才手動貼。
- **整合測試**：需要真實資源的測試加上 `//go:build integration`，CI 以 `-tags integration` 編譯並執行。`integration` 是本書唯一的測試 tag；不要寫 `//go:build !integration`，否則帶 tag 時那些檔案會被漏掉。
- 書中宣稱的行為（效能差距、錯誤訊息、競爭條件）要能被範例重現。效能數字要寫明量測方式與環境，不寫成保證。
- **效能**：時間（ns/op、延遲）每次執行、每台機器都不同，不放進 `// Output:`，也不在測試裡斷言。
  - 確定的結果（配置次數）用 `testing.AllocsPerRun` 在 after 的測試斷言，`-race` 下也要成立；經過 `sync.Pool` 的路徑不斷言精確次數（`-race` 下 Pool 會隨機丟掉物件）。
  - 效能工具的輸出（`go test -bench`、`benchstat`、`go tool pprof -top`／`-list`、`go build -gcflags=-m`、從 trace 轉出的 pprof）在 repo 裡實際執行後手動貼上，前一行標「執行結果（某一次）：」，保留 `goos`／`goarch`／`cpu` 標頭，省略的部分要在文中說明。
  - 正文的結論只談相對差異與 `benchstat` 的判斷，「快了多少」只引用貼出的 `benchstat` 結果；沒有顯著差異就照實寫。不放截圖。
  - `benchstat` 一律寫成固定版本的 `go run golang.org/x/perf/cmd/benchstat@v0.0.0-20261009192801-be2c69fb417e`，不加進 `go.mod`。
  - benchmark 用 `b.Loop`，準備工作放在迴圈外。CI 以 `-benchtime=1x` 讓每個 benchmark 跑一次。
  - profile 與 trace 檔不進 repo；唯一例外是 PGO 用的 `default.pgo`，放在 main 套件旁。
  - 最佳化的 after 可以 import 同集的 before 當比對基準（oracle），證明改寫前後結果一致。
- 省略「改寫模式」三段的集可以沒有 `before`／`after`，但書中出現的 Go 程式碼仍然要走單檔範例或多檔案範例其中一條驗證路徑。

## 第三方套件

- **標準庫與 `golang.org/x` 優先**。能用標準庫解決的就不要引入套件。
- OpenTelemetry 只在第 9 章「可觀測性」使用；`golangci-lint` 只在第 12 章「建置與發布」使用，並以 `go.mod` 的 `tool` 指令鎖定版本（Go 1.24 起），不是可 import 的相依。第 4、11、16 章可以用 `modernc.org/sqlite`（純 Go、不需要 cgo，與前作第 15 章相同）。第 16 章綜合實戰可以沿用上述套件，但不新增其他相依。
- 不教框架（Web 框架、ORM、相依注入框架）。要談取捨時，用標準庫寫出同樣的事，再說明框架解決了什麼、換來了什麼。
- 新增相依要在該集的 ticket 說明理由；單檔範例用到就同步鎖定在 `tools/checkcode/deps/go.mod`，多檔案範例鎖定在根目錄 `go.mod`。

## 正確性

- 以 Go 1.27 為準。Go 1.22 起迴圈變數每次迭代各一份；1.23 起有 range over func；1.26 起 `new(expr)`、`errors.AsType`；1.27 起泛型方法、struct literal 可用巢狀欄位選擇器、`encoding/json/v2`、`uuid` 套件。
- 不要寫過時的網路說法（例如 Go 1.22 前的迴圈變數陷阱、`time.After` 在迴圈中洩漏）。
- 不確定時用 `go doc <套件>.<名稱>` 或實際寫程式驗證，不要憑印象。
- 完成後執行 `go run ./tools/checkcode zh-TW/src/<章節目錄>`，再執行 `go vet ./... && go vet -tags integration ./... && go test -race -tags integration ./...` 與 `go test -run='^$' -bench=. -benchtime=1x ./...`，全部通過才算完成。

## 完成一章時

- 更新 `zh-TW/src/GUIDE.md` 第 1 節「已出版與未出版的章節」的表格，把新章節列為已出版。AI 助教靠這張表判斷哪些內容書中已經談過。
