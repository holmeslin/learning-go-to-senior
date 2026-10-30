# 日誌裡的敏感資料

## 本集目標

判斷哪些資料不能進日誌、由誰負責擋：型別用 `LogValuer` 決定自己被記成什麼，`ReplaceAttr` 依鍵名遮蔽當最後一道防線，其他的乾脆不記。

## 情境

一個電子錢包服務：`POST /cards` 讓使用者綁定信用卡，本文有使用者代號、密碼（再次確認身分）與完整卡號，`Authorization` 標頭帶著 token。客服要查「為什麼綁卡失敗」，所以大家習慣把整個請求記下來，好還原現場。

`log/slog`、JSON API 與 `httptest` 前作教過：〈[`log/slog`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter12/10_slog.html)〉、〈[JSON API](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter14/04_json_api.html)〉、〈[`httptest`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter14/09_httptest.html)〉。`LogValuer` 與 `ReplaceAttr` 的用法，第 9 章〈[結構化日誌：`log/slog` 的欄位與 handler 怎麼選](../chapter9/01_structured_logging.md)〉談過，那一集也預告了這個問題；欄位怎麼設計見〈[日誌層級與欄位設計](../chapter9/02_log_levels_and_fields.md)〉。秘密型別本身（`String`、`GoString`、`MarshalJSON`）本章〈秘密管理〉談過，這集只用到它的 `LogValue`。

## 常見寫法

把請求 struct 與標頭整組交給 `slog`：

```go,ignore
{{#include ../../../examples/chapter11/08_sensitive_data_in_logs/before/wallet.go:handler}}
```

沒有人寫「記錄密碼」，只寫了「記錄請求」。完整程式碼在 `examples/chapter11/08_sensitive_data_in_logs/before/`，它是刻意示範的漏洞，不要照用。

## 問題在哪

送一個綁卡請求（密碼、token 都是明顯的假值），看日誌的實際輸出：

```go,ignore
{{#include ../../../examples/chapter11/08_sensitive_data_in_logs/before/example_test.go:example}}
```

密碼、完整卡號、`Bearer` token 全部在同一行裡。`TextHandler` 用 `%+v` 印 struct，`JSONHandler` 則把 struct 序列化成 JSON，兩種都一樣。

- 漏洞是間接的：程式碼裡沒有任何一行寫著 `password`，code review 看不出來。
- 以後 `BindRequest` 加一個 `CVV` 欄位，它會自動進日誌，沒有人需要同意。
- 日誌一旦寫出去，就會被收集、索引、備份，也會被更多人看到。資料庫可以逐欄管權限，日誌行通常做不到。

## 較好的做法

**第一道：讓型別決定自己怎麼被記錄。** `Secret` 永遠記成 `[REDACTED]`；`Card` 只留末四碼，除錯夠用；`BindRequest` 自己逐欄決定輸出；`Headers` 用允許清單，只有名單上的標頭才記值，其他只留名稱：

```go,ignore
{{#include ../../../examples/chapter11/08_sensitive_data_in_logs/after/wallet.go:types}}
```

`BindRequest` 必須自己實作 `LogValue`。`slog` 只在屬性的值上呼叫 `LogValue`，不會鑽進 struct 的欄位；`JSONHandler` 序列化 struct 時，`Secret` 照樣輸出原本的字串。把 `BindRequest.LogValue` 拿掉，測試就會失敗。它也是允許清單：新欄位不會自己進日誌。

**第二道：handler 依鍵名遮蔽。** 有人隨手寫了 `slog.String("password", pw)`，型別幫不上忙，`ReplaceAttr` 在 handler 輸出之前攔下來：

```go,ignore
{{#include ../../../examples/chapter11/08_sensitive_data_in_logs/after/wallet.go:redact}}
```

`Redact` 匯出，是因為應用程式的進入點要把它放進 `HandlerOptions`，不是為了測試。handler 這樣用，輸出（`Example` 的 `redactNoTime` 先呼叫 `Redact`，再拿掉每次都不同的 `time`）：

```go,ignore
{{#include ../../../examples/chapter11/08_sensitive_data_in_logs/after/wallet.go:handler}}
```

```go,ignore
{{#include ../../../examples/chapter11/08_sensitive_data_in_logs/after/example_test.go:example}}
```

`bind` 是下面測試檔的 helper，送出和 before 一模一樣的請求，但日誌只剩使用者代號、`[REDACTED]` 與末四碼 `3456`。

**第二道有它看不到的地方。** `ReplaceAttr` 只看得到屬性的鍵名，包在 `slog.Any` 裡的 struct 對它是一整個不透明的值：

```go,ignore
{{#include ../../../examples/chapter11/08_sensitive_data_in_logs/after/example_test.go:opaque}}
```

所以第一道（型別）是主力，第二道只負責接住漏網的 `slog.String("password", ...)`。測試分別證明兩道：第一個測試的 logger 沒有 `ReplaceAttr`，只靠型別；`TestRedactMasksByKeyName` 則直接寫敏感的鍵名。測試解析日誌輸出，斷言三個假值都不出現、用於除錯的欄位仍在，並且拿同一個請求打 before，確認這些假值在 before 確實會漏出來，否則斷言證明不了什麼：

```go,ignore
{{#include ../../../examples/chapter11/08_sensitive_data_in_logs/after/wallet_test.go:test}}
```

另有測試鎖住卡號的邊界（含連字號、不足四碼）、標頭的允許清單與 `Redact`。把任何一道防線拿掉，對應的測試都會失敗。

## 取捨

- **遮蔽和不記，是兩種成本。** 遮蔽留下「這個請求帶了密碼」的訊號，但每個型別、每個鍵名都要有人維護，漏一個就漏一個。不記最安全，也最可能在排查時讓你說「早知道就記下來」。判斷的依據是排查需不需要這個值：密碼不需要，永遠不記；卡號只需要辨識是哪一張，末四碼就夠。末四碼本身仍是個人資料，卡號能顯示幾碼，要看你適用的支付卡規範（例如 PCI DSS）。
- **雜湊後的識別碼可能被反推。** 想用 `sha256(卡號)` 當關聯用的識別碼，搜尋空間太小：卡號的前綴與檢查碼有規則，電話、Email 也一樣，攻擊者拿到日誌就能逐一試算。要用的話，改成帶金鑰的 HMAC，金鑰本身是秘密，不能進日誌，也不能和日誌放在一起。換金鑰之後，新舊日誌就對不起來。
- **允許清單和拒絕清單。** `Headers` 用允許清單，因為沒人預料到的 `X-Api-Key` 也不會漏。代價是排查時看不到沒列的標頭值，要用再加。`Redact` 是拒絕清單：鍵名要列出來，寫成 `pwd` 或 `passwd2` 的欄位照樣漏。所以它只能是最後一道。
- **程式這一層擋不住的。** 第三方函式庫自己寫的日誌、`fmt.Errorf("... %+v", req)` 包進錯誤訊息的整個 struct、panic 的堆疊、反向代理的 access log（token 放在網址的 query string 就會被記下來）。這些要靠審視錯誤訊息、不把秘密放進網址、檢查收集端的設定來補。
- **保存期限與存取權限。** 日誌寫出去就收不回來。使用者要求刪除資料時，資料庫刪得掉，散在日誌裡的刪不掉，所以要設保存期限，層級越低（Debug）期限越短。誰能查日誌也要像誰能查資料庫一樣管，日誌平台通常開放給比資料庫更多的人。
- **before 什麼時候不算錯。** 本機開發、資料全是假的，整個請求記下來最省事。風險在這段程式碼會原封不動上線，所以預設的日誌就該是安全的。

## 重點整理

- 敏感資料多半是間接漏進日誌的：記整個請求 struct 或整組標頭，沒有人寫過 `password`。
- 第一道防線是型別：`LogValuer` 讓 `Secret` 全遮、`Card` 留末四碼，含敏感欄位的 struct 自己實作 `LogValue`，用允許清單決定輸出。
- 第二道防線是 `ReplaceAttr` 依鍵名遮蔽，只看得到屬性的鍵名，看不進 `slog.Any` 包著的 struct，只能接住漏網的。
- 測試解析日誌輸出，斷言假的秘密不出現、除錯用的欄位還在，並用 before 當對照證明斷言真的會失敗。
- 遮蔽、不記、雜湊各有成本；雜湊可被反推，日誌也要有保存期限與存取權限。
