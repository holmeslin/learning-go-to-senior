# 測試資料層：fake 與真實資料庫

## 本集目標

測資料層時，判斷什麼該對真的資料庫跑、什麼可以用 fake；用 fake 的時候，知道怎麼讓它不要和真的資料庫愈離愈遠。

## 情境

行銷團隊要匯入一份邀請名單。`Invite` 把每個信箱註冊進 `users` 表：已經註冊過的略過，並告訴呼叫端略過了誰；其他錯誤就中止。`users.email` 有唯一約束，而且不分大小寫。

`Service` 靠 `Store` 介面存取資料，介面宣告在使用端（第 3 章〈[在使用端定義介面](../chapter3/01_define_interface_at_consumer.md)〉）。`database/sql` 與 SQLite driver 怎麼用，前作〈[`sql.Open` 與 SQLite driver](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter15/01_open_sqlite.html)〉教過；store 這一層怎麼來，本章〈要不要有 repository 這一層〉談過。這集只談它們的測試。

## 常見寫法

用例要測，又不想為了單元測試架資料庫，所以用 map 寫一個 fake，交給 `Service`：

```go,ignore
{{#include ../../../examples/chapter4/07_fake_or_real_database/before/signup/signup.go:service}}
```

```go,ignore
{{#include ../../../examples/chapter4/07_fake_or_real_database/before/signup/fake_test.go:fake}}
```

```go,ignore
{{#include ../../../examples/chapter4/07_fake_or_real_database/before/signup/service_test.go:test}}
```

測試跑得快、不用裝任何東西，也通過了。真正接資料庫的 `SQLStore` 沒有測試：要測就得架資料庫，而用例的測試看起來已經把 `Add` 涵蓋了：

```go,ignore
{{#include ../../../examples/chapter4/07_fake_or_real_database/before/signup/store.go:add}}
```

完整程式碼在 `examples/chapter4/07_fake_or_real_database/before/`。

## 問題在哪

同一份名單、同一個 `Invite`，分別接上 fake 與真的 SQLite。`// Output:` 是實際輸出，由 `go test` 逐字比對：

```go,ignore
{{#include ../../../examples/chapter4/07_fake_or_real_database/before/signup/example_test.go:example}}
```

兩個 store 對同一個呼叫給了不同的答案：

- **fake 收下重複的信箱。** 第三筆的 ann 也成功了，`added=3`。map 不會替你執行唯一約束；外鍵、交易回滾也一樣（回滾本章〈交易邊界放在哪裡〉會用真的資料庫示範）。fake 只做你寫進去的事。
- **真的資料庫拒絕了，但 `SQLStore` 沒把錯誤翻成 `ErrEmailTaken`。** `Invite` 的「略過」分支永遠進不去，名單在第三筆中止，`skipped` 是空的。

這個 bug 通過了所有測試。fake 不會回 `ErrEmailTaken`，「略過重複」這條路徑就沒有測到；想測它，得先教 fake 回 `ErrEmailTaken`，而 fake 怎麼回，是寫的人憑印象決定的。沒有任何東西把 fake 和真的資料庫放在一起比過。

## 較好的做法

分三件事。

**測 store 時，直接對真的資料庫跑。** 每個測試拿到一個全新的檔案資料庫（放在 `t.TempDir()` 底下，測試結束自動清掉），套用的是 `signup.Schema`，也就是部署用的那份 SQL，不是測試自己抄一份。各測試用各的檔案，並行也不會互相干擾。資料表怎麼隨版本演進，本章〈migration〉會談。

```go,ignore
{{#include ../../../examples/chapter4/07_fake_or_real_database/after/signup/store_test.go:db}}
```

**把 driver 錯誤翻成領域錯誤。** 真的資料庫一被測，第一個失敗就是 before 看不到的那個。唯一約束的錯誤碼要在 `Add` 裡翻成 `ErrEmailTaken`（做法見第 2 章〈[包裝錯誤：`%w` 該不該暴露給呼叫端](../chapter2/02_wrap_or_expose.md)〉）：

```go,ignore
{{#include ../../../examples/chapter4/07_fake_or_real_database/after/signup/store.go:add}}
```

**fake 與真的 store 跑同一組契約測試。** 契約寫成一個函式，收一個「造出全新 store」的函式；fake 與 SQLite 各呼叫一次：

```go,ignore
{{#include ../../../examples/chapter4/07_fake_or_real_database/after/signup/store_test.go:contract}}
```

fake 現在也得照契約行事：信箱不分大小寫，重複就回 `ErrEmailTaken`。

```go,ignore
{{#include ../../../examples/chapter4/07_fake_or_real_database/after/signup/fake_test.go:fake}}
```

契約是兩個 store 之間唯一的連結：fake 少了大小寫處理，`TestFakeStore` 會失敗；schema 拿掉 `COLLATE NOCASE`，`TestSQLStore` 會失敗。有了這份保證，用例測試才能放心用 fake 測「略過重複」：

```go,ignore
{{#include ../../../examples/chapter4/07_fake_or_real_database/after/signup/service_test.go:test}}
```

同一份名單，兩個 store 現在給出一樣的答案：

```go,ignore
{{#include ../../../examples/chapter4/07_fake_or_real_database/after/signup/example_test.go:example}}
```

`Service` 一個字都沒改。完整程式碼在 `examples/chapter4/07_fake_or_real_database/after/`。

## 取捨

對真的資料庫跑測試，要付出：

- 比 map 慢，也多了依賴：每個測試要開檔案、建表。本章的 driver 是純 Go，不必安裝任何東西；換成需要外部程式的資料庫，CI 就得多起一個服務。
- 契約測試是多寫的程式，而且只涵蓋你列出來的案例。fake 仍是第二份實作：它用 `strings.ToLower`，連非 ASCII 字母都轉，而 SQLite 的 `NOCASE` 只轉 ASCII 字母。契約沒有這個案例，差異就沒人發現。

SQLite 也不是正式環境，常見的差異：

- **型別**：SQLite 的欄位型別只是「親和性」，不是強制；`INTEGER` 欄位可以存進文字 `'abc'`（`STRICT` 資料表可以避開）。PostgreSQL 會回報型別錯誤。
- **鎖**：SQLite 同一時間只有一個寫入者。正式資料庫有列鎖、隔離層級與死結，SQLite 重現不了。
- **方言**：`COLLATE NOCASE` 是 SQLite 的寫法；PostgreSQL 要用 `citext` 或 `lower(email)` 的唯一索引。契約沒變，實作是另一套。
- **錯誤碼**：`Add` 認的是 SQLite 的 `SQLITE_CONSTRAINT_UNIQUE`，PostgreSQL 的唯一約束錯誤是 SQLSTATE `23505`。SQLite 的測試全綠，不代表正式環境那條翻譯分支跑過。

所以這幾種情況要對正式環境用的資料庫跑測試：用到方言功能的查詢、取決於鎖與隔離層級的行為、migration 套上真資料庫的結果，以及 driver 錯誤的翻譯。做法是在 CI 起一個真的資料庫（例如測試容器 testcontainers，或 CI 提供的服務容器），`testStore` 多呼叫一次就行。整合測試與 build tags、替身的一般選擇，第 5 章會談。

before 在這些情況下反而是對的：

- 用例要驗證的是「store 失敗時我怎麼辦」。資料庫很難指定在哪一刻失敗，直接回傳 `errors.New("database down")` 的替身最省事，`TestInviteStopsOnOtherErrors` 就是這樣寫的。
- 小工具、只有一兩個查詢的 store，對真的資料庫手動跑一次就夠，不值得維護契約。
- 正式資料庫在本機根本起不來時，fake 加上一份能在 CI 對真資料庫跑的契約，好過什麼都不測。

判斷的方法：這個測試要驗證的行為，是你程式碼的，還是資料庫的？用例的分支是你的，fake 夠用；約束、排序規則、鎖是資料庫的，要問資料庫本人。

## 重點整理

- fake 只做你寫進去的事：map 不會執行唯一約束、外鍵或回滾，它和資料庫的差異不會有人通知你。
- 測 store 就直接對真的資料庫跑：每個測試一個全新的資料庫，套用和部署同一份 schema；SQLite 不必安裝，成本很低。
- 用例測試可以繼續用 fake，但 fake 與真的 store 要跑同一組契約測試，契約是兩者之間唯一的連結。
- SQLite 不是正式環境：型別、鎖、排序規則、錯誤碼都不同；碰到這些行為，就要用正式環境那種資料庫來測。
- 先問測的是誰的行為：自己的程式碼用 fake，資料庫的行為用真的資料庫。
