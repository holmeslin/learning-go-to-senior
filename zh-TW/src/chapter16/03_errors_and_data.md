# 錯誤與資料層

## 本集目標

判斷寄送紀錄表要怎麼設計，才能做到「寧可不寄，也不重複」；推播的錯誤要分成哪幾類，掃描迴圈才能只看分類決定重試或放棄；資料層的查詢逾時、連線池與 migration 各歸誰。

## 情境

提醒服務每 5 分鐘掃描一次，找出預定時間（活動開始前 24 小時）已到、還沒寄的報名，向推播供應商送出，預定時間後 1 小時內要送完；供應商限制每秒 50 次，一場活動約 2 萬人（平台與數字都是教材範例）。報名資料來自報名服務擁有的唯讀檢視，寄送紀錄表歸提醒服務，這個資料歸屬本章〈[設計文件](01_design_doc.md)〉決定過。

管道原本是推播與 email；書附範例只實作推播，email 走同一套寄送紀錄與重試，差別在供應商的限制是每日額度，範例省略。這集看兩個元件：資料層 `store` 與推播客戶端 `push`。套件怎麼切、怎麼依賴，本章〈套件結構與 API〉會談；它們的測試怎麼分層，本章〈測試〉會談。

## 寄送紀錄怎麼設計

主鍵是「活動 + 報名者」，這個唯一限制就是冪等鍵（第 10 章〈[冪等性](../chapter10/03_idempotency.md)〉的做法落成資料表）：寫得進去的才送，寫不進去代表已經有人處理過。

```sql
{{#include ../../../examples/chapter16/reminder/internal/store/schema/001_deliveries.sql:table}}
```

狀態有四個，資料表的 `CHECK` 擋住其他值：

```go,ignore
{{#include ../../../examples/chapter16/reminder/internal/store/store.go:states}}
```

每個轉換由 `store` 的一個方法負責：

| 從 | 到 | 方法 | 條件 |
| --- | --- | --- | --- |
| （無紀錄） | 寄送中 | `Begin` | 第一次嘗試：`INSERT ... ON CONFLICT DO NOTHING`，寫進一筆才算 |
| 待重試 | 寄送中 | `Begin` | 重試：`UPDATE ... WHERE state = 'retrying'`，次數加一 |
| 寄送中 | 已寄出、待重試、失敗 | `MarkSent`、`MarkRetry`、`MarkFailed` | 只轉換寄送中的紀錄，否則回傳 `ErrStateChanged` |
| 寄送中、待重試 | 失敗 | `FailOverdue` | 超過預定時間 1 小時；原本的失敗原因保留 |
| （無紀錄） | 失敗 | `FailOverdue` | 從未嘗試、期限已過、活動還沒開始：直接記失敗，不送出 |

失敗是終點，晚到的結果蓋不掉它（`TestMarkFailedAndTransitionsFromOtherStates`）。`Begin` 聽起來像開交易，其實是一個陳述式：

```go,ignore
{{#include ../../../examples/chapter16/reminder/internal/store/store.go:begin}}
```

「有沒有人處理過」與「寫入」在同一個陳述式裡完成，沒有先查再寫的空隙。`RowsAffected` 是 1 才能送；0 代表別人處理過了，例如兩邊的開關同時打開時，另一邊先寫了這一筆（`TestBeginWritesTheRecordOnlyOnce` 鎖住第二次回傳 `false`，`TestScanSkipsReminderRecordedByAnotherWriter` 驗證這一則被略過、沒有送出）。

一則提醒的路徑是三步：`Begin` 寫下寄送中、`Send` 送出、`Mark*` 記結果。行程在中間中斷，紀錄就停在寄送中。`Due` 只挑沒有紀錄的與待重試的，這一筆不會再被挑出，也就是不補寄：補寄得先確定上一次有沒有寄出，確定不了就可能重複（第 14 章〈[設計文件的 review](../chapter14/05_design_review.md)〉）。期限到了，由 `FailOverdue` 記為失敗；每次掃描先做這件事，再挑到期的：

```go,ignore
{{#include ../../../examples/chapter16/reminder/internal/store/store.go:expire}}
```

「從未嘗試」是服務停機超過 1 小時的情況，只限活動還沒開始的，否則掃描會替已經結束的活動寫出一堆失敗紀錄。兩種都記為失敗，掃描結果分別用 `Expired` 與 `Missed` 計數（`TestFailOverdue`）。

## 錯誤怎麼分類

推播客戶端回答一個問題：這個失敗，重送有沒有用。答案是 `push` 與呼叫端之間的契約（第 2 章〈[錯誤是 API 契約的一部分](../chapter2/07_error_contract.md)〉）：呼叫端只用 `errors.Is` 判斷兩個哨兵錯誤，不看狀態碼。`Send` 的文件註解逐條列出規則，程式照著寫：

```go,ignore
{{#include ../../../examples/chapter16/reminder/internal/push/push.go:errors}}
```

```go,ignore
{{#include ../../../examples/chapter16/reminder/internal/push/push.go:send}}
```

- 429 是 4xx 裡唯一值得再試的，供應商忙，等一下可能就好；其他 4xx 重送同樣的內容不會有不同的結果。3xx 也算永久失敗：客戶端不跟轉址，請求本文裡的裝置 token 不送到沒設定過的地方。
- 逾時與連線錯誤是暫時失敗：請求可能已經到了供應商。同一則的每次重試帶同一個 `Idempotency-Key`（`reminder-<活動>-<報名者>`），這是第二層冪等：主鍵保證提醒服務不會送兩次，標頭讓供應商在分不出有沒有送到時去重（範例假設供應商認這個標頭）。
- 逾時用每個請求的 `context.WithTimeout`，不設 `http.Client.Timeout`。期限只有一個來源，錯誤同時符合 `ErrTransient` 與 `context.DeadlineExceeded`（`TestSendTimesOutPerRequest` 在 `synctest` 的假時間裡驗證）。

確定的輸出交給 `Example`，由 `go test` 逐字比對。`status` 是固定回一個狀態碼的假供應商，`send` 用它送出一則通知：

```go,ignore
{{#include ../../../examples/chapter16/reminder/internal/push/push_test.go:fake}}
```

```go,ignore
{{#include ../../../examples/chapter16/reminder/internal/push/example_test.go:example}}
```

掃描迴圈只看分類：

```go,ignore
{{#include ../../../examples/chapter16/reminder/internal/dispatch/dispatch.go:deliver}}
```

`ErrTransient` 排重試，時間點寫進 `next_attempt_at`，由之後的掃描挑出；其餘一律記為失敗，包括分不出類別的錯誤：無法證明重送是安全的，寧可不寄（`TestScanTreatsUnclassifiedErrorAsPermanent`）。退避與 jitter 用第 10 章〈[重試、退避與 jitter](../chapter10/02_retry_backoff_jitter.md)〉的做法，限流與並行本章〈並行〉會談。

## 資料層的四個判斷

**查詢逾時在 `dispatch` 設，`store` 只用收到的 `ctx`。** 掃描迴圈是背景工作，沒有請求可以繼承 deadline，它自己就是邊界（第 4 章〈[查詢逾時該設在哪一層](../chapter4/03_query_timeout_layer.md)〉）。`dispatch` 為每個查詢設 `QueryTimeout`（預設 5 秒）；`store` 沒有自己的逾時，取消的 `ctx` 讓查詢立刻結束（`TestQueriesHonorTheContext`）。

**連線池上限是 4，送出的並行上限預設是 20。** 並行上限算同時在路上的請求，連線池算同時在資料庫的陳述式。一則提醒只在 `Begin` 與 `Mark*` 各占一條連線一小段時間，呼叫供應商的期間不占，所以池子不必跟著並行上限放大；sqlite 又只有一個寫入者，連線再多也只是排隊。池子是 `reminder` 套件的常數 `databaseMaxConns`，理由寫在 `store.OpenDB` 的文件註解；怎麼設、怎麼觀察，第 4 章〈[連線池要設多大、設錯會怎樣](../chapter4/04_pool_size.md)〉談過。

**migration 只管自己的表，檢視歸報名服務。** `Migrate` 只建 `reminder_deliveries`，版本表叫 `reminder_migrations`，帶前綴，因為示範的資料庫檔和報名服務共用。報名那一側由 `demoreg` 建出示範用的版本，文件註解標明「示範用，正式環境由報名服務維護」：

```sql
{{#include ../../../examples/chapter16/reminder/internal/demoreg/schema.sql:view}}
```

檢視是兩個服務之間的契約：`CheckCandidates` 在啟動時讀一次它的欄位，檢視不見了或欄位被改掉就拒絕啟動，開關關閉的部署也一樣，這就是上線第一步的「讀得到檢視」（`TestCheckCandidates`）。`Migrate` 是第 4 章〈[migration](../chapter4/06_migration.md)〉的機制，每個檔案與它的版本紀錄在同一個交易裡，是 `store` 唯一開交易的地方：

```go,ignore
{{#include ../../../examples/chapter16/reminder/internal/store/migrate.go:migrate}}
```

沒有鎖是那一集談過的限制。服務只跑一個實例、發布時先停舊再起新，兩個行程不會同時 migrate，所以用 `shortcut:` 記下；改成多副本時要先還這筆債。

**所有查詢都用參數。** SQL 是 `const`，值走具名參數或 `?`，第 11 章〈[SQL injection 與參數化查詢](../chapter11/02_sql_injection.md)〉的判斷照用。這裡的輸入不是請求：裝置 token 是報名服務那一側的資料，`last_error` 存的是錯誤訊息，內容都不由提醒服務決定。`TestQueriesAreParameterized` 把 `x'); DROP TABLE reminder_deliveries; --` 當作兩者寫進去，讀出來是原樣的文字。

兩個挑選的查詢與 `Due` 如下，`queryReminders` 用 `defer rows.Close()` 歸還連線（沒關的 `rows` 會占住連線，第 4 章〈[連線池要設多大、設錯會怎樣](../chapter4/04_pool_size.md)〉談過）；一整批載入記憶體是 `shortcut:`，一場約 2 萬筆是數 MB，大到放不下時改成分頁：

```go,ignore
{{#include ../../../examples/chapter16/reminder/internal/store/store.go:due}}
```

## 哪裡會衝突

各章的建議湊在這條路徑上，會互相拉扯：

- **交易包不住「寫紀錄、送出、記結果」。** 第 10 章〈[冪等性](../chapter10/03_idempotency.md)〉建議把冪等鍵與動作寫在同一個交易，動作若是呼叫外部，就把同一個鍵一併傳給對方；第 4 章〈[交易邊界放在哪裡](../chapter4/02_transaction_boundary.md)〉建議交易內不做網路呼叫。這裡的動作就是網路呼叫，回滾不了，也不該讓交易開著等供應商。折衷是把交易邊界縮到單一陳述式：條件式的 `INSERT` 與 `UPDATE ... WHERE state = ...`，用「寄送中」這個中間狀態橫跨網路呼叫。代價是縫隙還在：中斷的那一筆停在寄送中、不補寄。
- **硬要包住，連線池就得跟著並行上限。** 若 `Begin` 到 `Mark*` 在同一個交易裡，連線在整個呼叫供應商的期間被占住：池子 4、並行上限 20，同時在路上的請求最多 4 個；供應商一慢，連線被占滿，連寫紀錄的查詢也得排隊等連線。單一陳述式的寫法讓這兩個數字可以各自調。
- **`WithoutCancel` 之後，deadline 也沒了。** 進行中的送出不能被取消切斷，所以 `deliver` 通過限流之後用 `context.WithoutCancel`；它拿掉的不只取消，還有 deadline（第 10 章〈[逾時預算](../chapter10/01_timeout_budget.md)〉談過）。第 4 章〈[查詢逾時該設在哪一層](../chapter4/03_query_timeout_layer.md)〉要 `store` 只用收到的 `ctx`，而這時收到的 `ctx` 沒有任何期限，所以 `dispatch` 要為每個查詢與每個請求各設自己的期限，否則資料庫卡住時，進行中的送出不會結束，關機期限先到。取消與關機的整體順序，本章〈並行〉會談。

## 取捨

- **不補寄的代價是真的。** 行程在寫完紀錄、記結果之前結束，這則提醒就少寄一次；提醒漏一則，使用者只是少收一次，通知若攸關金錢或安全，優先順序可能相反（第 14 章〈[設計文件的 review](../chapter14/05_design_review.md)〉談過）。即使帶同一個冪等鍵也不補寄：供應商認不認這個標頭、保存多久，不在這個服務的掌握中。供應商明確保證保存的時間長過預定時間後的 1 小時，補寄停在寄送中的那一筆才值得重新評估。
- **分不出類別就放棄，錯誤沒包好時會默默丟提醒。** 新增一種錯誤卻忘了包 `ErrTransient`，那一類失敗就不再重試；契約的外部測試、`Example` 與掃描結果裡的失敗數是防線。反過來預設重試，永久失敗會一直打供應商（第 10 章〈[重試、退避與 jitter](../chapter10/02_retry_backoff_jitter.md)〉）。
- **收到 429 沒有讀 `Retry-After`，只用自己的退避。** 這是刻意留下的簡化；供應商會在回應裡明說要等多久時，才值得讀它。
- **先寫紀錄、再送出，多一次寫入。** 先送出、再記結果可以省一次寫入，但寄出之後、記結果之前中斷，下一輪會再寄一次。重複無害又不能漏寄時，那樣寫才是對的；提醒選了「寧可不寄」，所以付這次寫入。

## 重點整理

- 寄送紀錄的主鍵就是冪等鍵：先寫「寄送中」再送出，結果用只轉換預期狀態的條件式 `UPDATE` 記下，失敗是終點。
- 中斷的那一筆停在寄送中，不補寄，期限到了記為失敗；從未嘗試而逾期的直接記失敗、不送出。
- 推播的錯誤分成永久與暫時，是 `push` 與呼叫端的契約：429、5xx、逾時可重試，其他 4xx 與 3xx 放棄，分不出類別也放棄；掃描迴圈只用 `errors.Is` 判斷。
- 交易不該包住網路呼叫：交易邊界縮到單一陳述式，連線池才不必跟著並行上限放大；`WithoutCancel` 之後，查詢要自己設期限。
- migration 只管自己的表，檢視歸報名服務、啟動時檢查契約；所有查詢都用參數，輸入不只來自請求。
