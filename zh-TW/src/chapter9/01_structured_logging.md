# 結構化日誌：`log/slog` 的欄位與 handler 怎麼選

## 本集目標

寫日誌時判斷：變數要寫成欄位，不要拼進訊息；型別自己決定怎麼被記錄；函式庫接收 `*slog.Logger`，`TextHandler` 或 `JSONHandler` 由程式的進入點決定。

## 情境

一個折價券服務：使用者用券號核銷折價券，券不存在或已經用過就失敗。客服會問「`u42` 昨天為什麼核銷失敗」，維運會問「核銷失敗的原因各佔多少」。這些問題的答案都在日誌裡，所以日誌要查得出來。

`slog` 的基本用法前作已經教過：〈[`log/slog`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter12/10_slog.html)〉。這集不重講 API，只談欄位怎麼寫、handler 怎麼選。日誌的層級與鍵名約定，本章〈日誌層級與欄位設計〉會談；讓日誌帶上 trace ID，本章〈在 `context` 裡傳遞 trace〉會談。

## 常見寫法

把使用者、券號這些變數用 `fmt.Sprintf` 拼進訊息，寫給全域的 logger：

```go,ignore
{{#include ../../../examples/chapter9/01_structured_logging/before/voucher.go:redeem}}
```

換成 `log.Printf` 也是同一種寫法。人讀起來一點問題都沒有。完整程式碼在 `examples/chapter9/01_structured_logging/before/`。

## 問題在哪

日誌用 `JSONHandler` 輸出，下面的 `Example` 核銷三次，再找「`u4` 的所有失敗」，唯一的辦法是比對字串：

```go,ignore
{{#include ../../../examples/chapter9/01_structured_logging/before/example_test.go:example}}
```

`u4` 只失敗一次，卻撈出兩行：`user u4` 同時是 `user u42` 的開頭。比對成 `user u4 ` 可以躲掉這一次，但使用者代號剛好排在訊息最後面時又不成立。輸出雖然是 JSON，裡面只有 `level` 與 `msg`，使用者、券號、原因全部擠在 `msg` 這一個字串裡，要查詢就得寫綁定措辭的比對或正規表達式；有人把訊息改成 `redeem failed for user ...`，所有查詢跟著壞掉。「各種失敗原因各幾次」更沒有欄位可以彙總。

另一個問題在 `Redeem` 直接呼叫全域的 `slog.Warn`：呼叫端換不掉輸出位置。上面的 `Example` 得用 `slog.SetDefault` 改全域狀態再還原，這種測試不能和別的測試並行。

## 較好的做法

變數寫成欄位，訊息只留一句固定的話，函式庫用建構函式接收的 `*slog.Logger`：

```go,ignore
{{#include ../../../examples/chapter9/01_structured_logging/after/voucher.go:redeem}}
```

`slog.String`、`slog.Any` 這些 `slog.Attr` 帶著型別，`LogAttrs` 只接受 `Attr`，不會少一個參數。`voucher` 欄位的值是自訂型別，它實作 `LogValuer`，由型別決定自己被記成什麼：

```go,ignore
{{#include ../../../examples/chapter9/01_structured_logging/after/voucher.go:logvalue}}
```

呼叫端只寫 `slog.Any("voucher", v)`，之後要加欄位，改這一個方法，所有記錄 `Voucher` 的地方一起改。`TextHandler` 把群組欄位印成 `voucher.code`，`JSONHandler` 印成巢狀物件。下面的 `Example` 用 `ReplaceAttr` 拿掉 `time`，輸出才會固定：

```go,ignore
{{#include ../../../examples/chapter9/01_structured_logging/after/example_test.go:notime}}
{{#include ../../../examples/chapter9/01_structured_logging/after/example_test.go:example}}
```

同樣的資料換成 `JSONHandler`，每個欄位在輸出裡都有名字與型別：

```go,ignore
{{#include ../../../examples/chapter9/01_structured_logging/after/example_test.go:json}}
```

測試把 `JSONHandler` 的輸出解析回來，直接依欄位篩選，斷言層級、欄位與型別。同一組資料，`u4` 這次只有一筆；`cents` 在 JSON 裡是數字，解析後是 `float64`，不是字串：

```go,ignore
{{#include ../../../examples/chapter9/01_structured_logging/after/voucher_test.go:test}}
```

另外兩個測試鎖住設計：注入 logger 之後，全域的 logger 什麼都收不到；傳入 `nil` 時，函式庫丟棄日誌，不會偷偷改用全域的 logger。函式庫不碰全域狀態，所以斷言日誌的測試各寫各的 buffer，可以並行。

handler 在程式的進入點選。本機看日誌的是人，`TextHandler` 好讀；正式環境讀日誌的是日誌收集系統，`JSONHandler` 的欄位與型別不需要再靠規則解析。選好之後交給函式庫：

```go,ignore
{{#include ../../../examples/chapter9/01_structured_logging/after/cmd/voucherd/main.go:main}}
```

## 取捨

`slog.Attr` 比交替的「鍵、值」囉唆，換來型別：數字與字串不會混在一起，也不會漏一個值。交替參數寫起來短，但少一個值或鍵不是字串時，`slog` 不會失敗，而是印出 `!BADKEY`：

```go,ignore
{{#include ../../../examples/chapter9/01_structured_logging/after/example_test.go:badkey}}
```

`go vet` 的 slog 檢查會抓這類寫法：缺最後一個值時回報 `call to slog.Info missing a final value`，鍵不是字串或 `slog.Attr` 時也會回報。所以團隊要跑 `go vet`。兩種寫法也可以混用，例如 `logger.Info("m", slog.String("a", "b"), "k", 1)`。

自訂 handler 的成本比看起來高。`Handler` 有四個方法，`WithAttrs` 與 `WithGroup` 的語意（群組要套用到之後的屬性）容易寫錯，標準庫提供 `testing/slogtest` 檢查，但維護是你的責任。多數需求先用 `HandlerOptions`（層級、`ReplaceAttr`）解決；真的要寫，也只包住標準 handler，改 `Handle` 加欄位或轉送。

日誌量是成本。每一行日誌都佔儲存與索引，JSON 的鍵名每行重複一次，欄位越多、越常記，成本越高。欄位化讓日誌查得到，不代表什麼都該記；哪些事件值得一行，本章〈日誌層級與欄位設計〉會談。記下來的欄位本身也要小心：`Voucher` 的 `LogValue` 可以決定不輸出哪些部分，但日誌裡的敏感資料怎麼處理，第 11 章〈日誌裡的敏感資料〉會談。

before 的寫法也不是永遠錯。一次性的小工具、沒有人會查詢的訊息，一行文字更直接。要注意的是 `slog.SetDefault` 會讓舊程式的 `log.Printf` 也走新的 handler，但整行仍是一個 `msg`，不會因此變成欄位。

## 重點整理

- 變數寫成欄位，訊息留固定的一句話；拼進訊息的日誌只能比對字串，會誤判，也無法彙總。
- `slog.Attr` 帶型別，不會有 `!BADKEY`；交替參數要靠 `go vet` 檢查。
- 自訂型別實作 `LogValuer`，讓型別決定自己怎麼被記錄，欄位改一處就好。
- 函式庫接收 `*slog.Logger`，不用全域的 logger；`TextHandler` 或 `JSONHandler` 在進入點選。
- 自訂 handler 與日誌量都有成本：先用 `HandlerOptions`，欄位化之後仍要想哪些事件值得記。
