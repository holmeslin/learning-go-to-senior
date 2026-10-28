# 要啟動真的 server，還是只測 handler

## 本集目標

寫 HTTP 的測試時，判斷該測哪一層：大部分案例測組好的 `http.Handler`，只有需要真實連線行為時才啟動 server。

## 情境

一個筆記服務：`GET /notes/{id}` 讀取筆記，`DELETE /notes/{id}` 刪除筆記，刪除必須帶 `Authorization: Bearer <token>`。兩個 handler 函式各管一件事，token 檢查是 `requireToken` 這個 middleware，最後由 `NewHandler` 用 `http.ServeMux` 把路由與 middleware 組起來，回傳一個 `http.Handler`。

`httptest` 的用法前作已經教過：〈[`httptest`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter14/09_httptest.html)〉。這集不教 API，只談測哪一層。「組好的 handler」是什麼，第 1 章〈[套件切分的依據（依職責，不依型別）](../chapter1/01_package_by_responsibility.md)〉的 `taskhttp.NewHandler` 就是一例。

## 常見寫法

每個 handler 函式各寫一個測試，直接呼叫，用 `httptest.NewRecorder` 接結果；middleware 也單獨測：

```go,ignore
{{#include ../../../examples/chapter5/09_server_or_handler/before/handlers_test.go:test}}
```

三個測試都過，每一條路徑也都有測到。完整程式碼在 `examples/chapter5/09_server_or_handler/before/`。

## 問題在哪

這三個測試測的都是零件，沒有一個走過 `NewHandler` 組出來的 handler。而零件之間怎麼接，寫在這裡：

```go,ignore
{{#include ../../../examples/chapter5/09_server_or_handler/before/notes.go:routes}}
```

讀取的 pattern 少打了一個 `s`，刪除的路由忘了包上 `requireToken`。換成 client 的角度，對組好的 handler 發請求：

```go,ignore
{{#include ../../../examples/chapter5/09_server_or_handler/before/example_test.go:example}}
```

`GET /notes/1` 拿到 405：`/notes/1` 只註冊了 DELETE，所以 mux 回「方法不允許」，而不是 200。沒有 token 的 `DELETE` 回 204，筆記被刪了。before 的測試全過，因為 `TestGetNote` 自己用 `SetPathValue` 把 `{id}` 填好，繞過路由；`TestRequireToken` 測的是 middleware，沒有人檢查它掛上了沒。

## 較好的做法

測組好的 `http.Handler`，用 `NewRecorder` 跑大部分案例。請求從路由進去、穿過 middleware，和線上走同一條路；沒有網路、沒有 port，一個案例就是一次函式呼叫。案例形狀一致，放進一張表：

```go,ignore
{{#include ../../../examples/chapter5/09_server_or_handler/after/handler_test.go:table}}
```

表裡刻意有「錯路徑」「錯方法」「錯 token」，這些都是路由與 middleware 才決定得了的行為。每個測試用 `newHandler()` 建自己的 Handler，沒有共用狀態。把 after 的路由改回 before 的樣子，這張表與 `Example` 會失敗；只拿掉 middleware 的話，執行 `go test` 的輸出如下（省略了每個測試與套件後面的耗時）。

執行結果：

```text
--- FAIL: TestRoutes
    --- FAIL: TestRoutes/delete_without_token
        handler_test.go:41: DELETE /notes/1 = 204, want 401
    --- FAIL: TestRoutes/delete_with_wrong_token
        handler_test.go:41: DELETE /notes/1 = 204, want 401
--- FAIL: TestUnauthorizedDeleteKeepsNote
    handler_test.go:63: GET /notes/1 after unauthorized delete = 404, want 200
--- FAIL: ExampleNewHandler
got:
GET /notes/1 = 200
DELETE /notes/1 without token = 204
want:
GET /notes/1 = 200
DELETE /notes/1 without token = 401
FAIL
FAIL	github.com/holmeslin/learning-go-to-senior/examples/chapter5/09_server_or_handler/after
FAIL
```

同樣的 `Example` 在 after 印出 `GET /notes/1 = 200` 與 `DELETE /notes/1 without token = 401`。

什麼時候才啟動 server？after 多了一個 `Client`，它用 `http.Client` 呼叫這個服務。client 的程式會經過 `net/http` 的 `Transport`：真的建立連線、真的送出請求、真的收到回應，`NewRecorder` 這一層都沒有。所以這裡用 `httptest.NewServer`：

```go,ignore
{{#include ../../../examples/chapter5/09_server_or_handler/after/handler_test.go:client}}
```

兩個測試都用 `defer srv.Close()` 關掉 server。第二個測試測取消：handler 先通知「請求收到了」，然後等到 client 斷線；測試等到通知才呼叫 `cancel()`，全程沒有 `time.Sleep`，也不賭逾時要設多短。

## 取捨

`NewRecorder` 不經過網路堆疊，所以這些行為它測不到：連線的建立與關閉、client 的逾時與取消、`Flush` 之後的串流、`Transport` 的行為（重用連線、重導向）。需要這些，就得用 `NewServer`。

`NewServer` 的代價：

- 比較慢。每次要開 listener、建連線，案例多的表格不該每一列都開一個 server。
- 有東西要關。忘了 `srv.Close()` 會留下 goroutine 與 port；handler 卡住時，`Close` 也會等到它結束。
- 失敗時的訊息多半是連線錯誤，不像比對狀態碼那樣直接。

before 的寫法也不是沒有用處。零件各自有複雜邏輯時，單獨測 handler 函式或 middleware 是合理的，例如 `requireToken` 要比對多種 token 格式。但它不能取代測組好的 `http.Handler`，兩者看的是不同的 bug。

`httptest.NewServer` 只監聽 loopback，不依賴外部服務，CI 可以直接跑。要連真實資源（資料庫、檔案）的整合測試怎麼和單元測試分開，本章〈[整合測試與 build tags](05_integration_tests_build_tags.md)〉談過。

## 重點整理

- 直接呼叫 handler 函式的測試繞過了路由與 middleware：pattern 寫錯、middleware 沒掛上，測試照過。
- 測組好的 `http.Handler`，請求從路由進去，才測得到這些接線。
- 大部分案例用 `httptest.NewRecorder`：快、沒有 port、不用關。
- 需要真實連線行為（client 程式、取消與逾時、串流）時才用 `httptest.NewServer`，並且記得 `Close`。
- 用「通知」而不是 `time.Sleep` 來排定先後，測試才不依賴機器的快慢。
