# 作者的自我審查

## 本集目標

站在作者這一側：送出 PR 之前，自己先讀完整個 diff、在本機跑過建置流程、把描述寫清楚，讓 reviewer 的時間花在作者看不到的地方；也要知道自我審查抓不到什麼，不拿它取代 review。

## 情境

本章〈[先看設計，再看細節](01_design_before_details.md)〉與〈[怎麼寫 review 意見](04_writing_review_comments.md)〉都站在 reviewer 一側，這集換成作者。「加入候補名單」PR 寫完了，測試全部通過，動了 12 個檔案，功能、改名與搬動的程式碼混在一起。標題寫好了，描述欄是空的。作者按下送出。

## 沒看過的 diff

reviewer 打開 PR，第一步是讀描述；沒有描述，只好先問。除了設計，這個 PR 還有三件事，作者自己看一遍就會處理，reviewer 卻得花時間確認、等回覆。

**沒有描述。** reviewer 只有標題與 12 個檔案：為什麼要做、哪些是改名、有沒有刻意不做的事，全靠猜。先留一則「請補描述」，作者回覆之後才開始看。

**留下除錯輸出。** 取消端點每次成功都印一行活動的名單，連 `fmt` 的 import 都是為了這一行才加的：

```go,ignore
{{#include ../../../examples/chapter13/01_design_before_details/before/handler.go:debug}}
```

在作者的機器上，它用來確認遞補有沒有生效；進了主幹，就是每次取消都多一行雜訊，名單裡還有報名者的 id。

**測試沒走過新行為。** 測試全部通過，但只走成功的路徑。測試檔的 diff 看得出來：在範例目錄下用 `-U0` 只列出改動的行，下面是第三個 hunk（其餘五個是 import 路徑、`NewEvent` 多出的參數與兩個方法的改名，標頭一併省略）：

```bash
git diff --no-index -U0 base/event_test.go before/event_test.go
```

執行結果：

```text
@@ -18,2 +18,2 @@ func TestRegisterUntilFull(t *testing.T) {
-	if err := ev.Register("u3"); !errors.Is(err, signup.ErrFull) {
-		t.Errorf("Register(u3) = %v, want ErrFull", err)
+	if got, want := ev.Confirmed(), []string{"u1", "u2"}; !slices.Equal(got, want) {
+		t.Errorf("Confirmed() = %v, want %v", got, want)
```

`-` 兩行是 `base/` 裡額滿回傳 `ErrFull` 的斷言，`+` 兩行是 `Attendees()` 改名之後原本就有的名單檢查，git 把它們配成同一個 hunk。額滿的斷言就這樣消失，沒有補新的。

## 送出前自己審一遍

1. **用 reviewer 的順序讀完整的 diff。** 看整個 PR 對主幹的差異，不是最後一個 commit。先看改了哪些 API、責任放在哪一層，再看細節，順序同本章〈[先看設計，再看細節](01_design_before_details.md)〉。再掃一遍：除錯輸出、註解掉的程式碼、`TODO`、不相干的格式變動、被刪掉的斷言。
2. **在本機跑 CI 會跑的那串指令。** `gofmt -l`、`go vet`、`go test -race`、lint，指令和 CI 一樣，第 12 章〈[`golangci-lint` 與 CI](../chapter12/05_golangci_lint_ci.md)〉談過。紅燈在 CI 才出現，reviewer 收到通知時看到的是一個還不能合併的 PR。
3. **對每個新行為問：它壞掉時，哪個測試會失敗？** 用覆蓋率找沒被跑到的分支，第 5 章〈[覆蓋率的意義與限制](../chapter5/11_coverage_meaning_limits.md)〉談過；測試該釘住的是公開 API 看得到的行為，第 5 章〈[測什麼、不測什麼](../chapter5/01_what_to_test.md)〉談過。
4. **寫描述，標出請 reviewer 特別看的地方。** 見下面的〈PR 描述〉。

第 3 步在 `before/` 上的做法如下；輸出省略了 `go test` 那行的耗時，`grep` 只留下三個函式：

```bash
go test -count=1 -coverprofile=cover.out ./examples/chapter13/01_design_before_details/before
go tool cover -func=cover.out | grep -E 'Register|Cancel|PromoteNext'
```

執行結果：

```text
ok  	github.com/holmeslin/learning-go-to-senior/examples/chapter13/01_design_before_details/before	coverage: 85.4% of statements
github.com/holmeslin/learning-go-to-senior/examples/chapter13/01_design_before_details/before/event.go:38:	Register	92.9%
github.com/holmeslin/learning-go-to-senior/examples/chapter13/01_design_before_details/before/event.go:63:	Cancel		75.0%
github.com/holmeslin/learning-go-to-senior/examples/chapter13/01_design_before_details/before/waitlist.go:27:	PromoteNext	100.0%
```

整個套件是 85.4%。`Cancel` 只有 75.0%：沒被跑到的區塊之一，是移除候補者的那一支，沒有任何測試是候補者本人取消（R6）。`Register` 缺的是候補名單也滿時的 `ErrFull`。`PromoteNext` 是 100.0%，但覆蓋率只量「有執行」，量不出遞補該不該放在那裡。

## PR 描述

描述回答四個問題：為什麼做、做了什麼、怎麼驗證、風險。下面是為 `before/` 的 PR 補寫的版本（除錯輸出已刪）：

```text
加入候補名單

為什麼
活動額滿之後，新的報名者目前只會收到 409。改成可以排進候補，
有人取消時依排隊順序遞補第一位，並通知他。

做了什麼
- 額滿時，Register 把人排進候補名單（上限 MaxWaitlist，預設 50）；
  候補也滿才回 ErrFull。
- 取消端點在 Cancel 之後呼叫 PromoteNext：遞補第一位，
  並用呼叫端提供的 Notifier 通知。
- GET 活動多回傳 waitlist 欄位；ErrFull 的訊息從 event is full
  改成 event and waitlist are full。
- 順手改名（capacity -> seats、Capacity() -> Seats()、
  Attendees() -> Confirmed()），並把 normalize 搬到 validate.go。
  兩者都沒有行為變更，JSON 欄位名稱不變。

怎麼驗證
- gofmt、go vet、go test -race、golangci-lint 在本機通過。
- 新增測試：依排隊順序遞補（直接呼叫與經由 HTTP 取消）。
- 還沒有測試：候補者本人取消、候補名單也滿時回 ErrFull。

風險
- 額滿時 POST 報名從 409 變成 201（排進候補）。
- 匯出方法 Capacity()、Attendees() 改名，signup 以外的呼叫端要跟著改。
- 通知失敗由 Notifier 的實作處理，這個 PR 不重試。

請特別看
- 行為變更：event.go 的 Register、Cancel，waitlist.go，
  handler.go 的取消端點（呼叫 PromoteNext）與 GET（多一個 waitlist 欄位），
  errors.go 的 ErrFull 訊息。
- 只有改名與搬動的部分：event.go、store.go、handler.go 與測試檔裡的改名，
  normalize 搬到 validate.go。
```

每一欄都要作者把原本只在腦中的事寫成句子。寫「怎麼驗證」時，沒測到的地方自己浮出來；寫「風險」時，額滿回 201 這個契約變更也得寫成一句話。寫出來不等於看出問題，但 reviewer 不必再猜。「請特別看」把注意力放在行為變更上，12 個檔案才不會把它淹沒。

長度以 reviewer 不打開 diff 就能回答這四個問題為止。改一個錯字、升一個相依，一兩句就夠；超過一個螢幕，多半是 PR 太大，描述寫得再清楚也不能取代拆分，本章〈[PR 大小與拆分](03_pr_size_and_split.md)〉談過。描述講「為什麼」與「去哪裡看」，不逐行翻譯 diff，diff 自己會說。

## 抓得到與抓不到的

自我審查抓得到的，是機械性的疏漏與沒做完的工作：除錯輸出、隔天再讀就看不懂的縮寫（`PromoteNext` 裡的 `cf`、`wl`、`nx`，R7）、沒測到的分支。這些作者看得到，只是送出前沒有人要求他看。

抓不到的是自己的設計與假設。遞補寫在取消端點，是因為作者腦中的「取消」就是那個 `DELETE` 請求；同一個人用同一個前提再讀十遍 diff，得到的還是同一個答案，測試全綠，`PromoteNext` 的覆蓋率是 100.0%，沒有任何訊號會發問（R1）。`ExampleEvent` 的 `// Output:` 把 `u3 <nil>` 印在作者眼前：名額 2 的活動，第三位報名者的回傳值和前兩位一樣。它每次執行都在，但得先問「呼叫端怎麼分辨報名成功與排進候補」，才看得出那是個問題（R2）。介於兩者之間的，例如持鎖時呼叫 `Notify`，要靠一份清單逐項問才有機會自己抓到，本章〈[Go 程式碼的 review 檢查清單](02_go_review_checklist.md)〉談過。

補強盲點的做法：隔一段時間再讀；挑一個 domain 的入口（直接呼叫 `Cancel`）自己走一遍；方向有疑問就在動手前找人談，不要等到 PR。設計層級的討論該寫在設計文件裡，第 14 章〈[什麼時候需要寫](../chapter14/01_when_to_write.md)〉談過。

## 取捨

- 自我審查花的時間要和 PR 的大小成比例。十幾行的改動，讀一遍 diff 加一兩句描述就夠；每一欄都寫滿，是在浪費作者與 reviewer 的時間。
- 想早點拿到方向回饋的草稿 PR，可以直接送，在描述寫明「還沒完成，想先確認方向」。這時跳過自我審查是對的，前提是講明白。
- 「請特別看」會引導 reviewer 的注意力，也可能讓他忽略沒標出的地方。reviewer 仍照本章〈[先看設計，再看細節](01_design_before_details.md)〉的順序看，標出的地方是補充，不是範圍。
- 把沒測到的地方寫進描述，是誠實，不是免責：reviewer 仍可能要求在這個 PR 補上，還是之後再補，本章〈什麼時候該擋、什麼時候可以放行〉會談。

## 重點整理

- 送出前用 reviewer 的順序讀完整個 diff：先看 API 與責任放在哪一層，再掃除錯輸出、被拿掉的斷言與不相干的變動。
- 在本機跑 CI 那串指令，用覆蓋率找沒被測到的分支；這些留給 reviewer 找，是浪費他的時間。
- PR 描述回答為什麼、做了什麼、怎麼驗證、風險，再標出請特別看的地方；長度以不打開 diff 就能回答這四題為止。
- 自我審查抓得到疏漏與沒做完的工作，抓不到自己的設計盲點：同一個人、同一個前提，得到同一個答案。
- 作者看過不能取代 reviewer 看。
