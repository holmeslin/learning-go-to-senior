# `crypto/rand` 與 TLS 設定

## 本集目標

看到「給別人猜不到的值」就用 `crypto/rand`、比對秘密就用常數時間；看到 `InsecureSkipVerify: true` 就知道它關掉的是整個身分檢查，並改成讓 client 信任指定的 CA，伺服器則依自己的 client 決定最低版本。

## 情境

內部有一支帳號服務，要發密碼重設碼，也要呼叫另一個內部服務查資料。那個服務的憑證是公司私有 CA 簽的，第一次連的時候，client 回報 `certificate signed by unknown authority`。

`math/rand/v2` 與 HTTP client、`httptest` 前作教過：〈[`math/rand/v2`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter12/11_rand_v2.html)〉、〈[HTTP client](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter14/06_http_client.html)〉、〈[`httptest`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter14/09_httptest.html)〉。這集談的是哪些地方不能用亂數套件、TLS 的錯誤訊息該怎麼處理。

## 常見寫法

重設碼用 `math/rand/v2`，種子取服務啟動的時間；比對 token 用 `==`；client 被憑證錯誤擋住，網路上最快的解法是關掉驗證：

```go,ignore
{{#include ../../../examples/chapter11/06_crypto_rand_tls/before/authkit.go:token}}
```

```go,ignore
{{#include ../../../examples/chapter11/06_crypto_rand_tls/before/authkit.go:client}}
```

三段都很短，也都「能動」。這個套件刻意有漏洞，完整程式碼在 `examples/chapter11/06_crypto_rand_tls/before/`，不要照用。

## 問題在哪

**重設碼可以預測。** 攻擊者替自己的帳號要一組碼，再用「服務是這一天之內啟動的」這點資訊，逐秒試種子，找出產生那組碼的種子，下一組就是受害者的：

```go,ignore
{{#include ../../../examples/chapter11/06_crypto_rand_tls/before/authkit_test.go:predict}}
```

Go 的文件對 `math/rand/v2` 的說法比這個例子更保守：它「不該用在與安全有關的工作」，而且輸出「無論怎麼設種子都可能容易預測」，其中 PCG 明寫「不適合」。所以問題不只是種子猜得到，這個套件本來就不是為了不被猜到而設計的。

**`==` 比對 token。** `==` 在第一個不同的位元組就回傳，花的時間會洩漏「前面幾個位元組對了」。實際上網路抖動遠大於這個差距，要量得出來很難，但常數時間比對一行就能換掉，沒有理由賭。這一點沒辦法用測試展示：時間不能斷言，所以 after 的測試只驗證比對結果，不驗證耗時。

**關掉驗證的 client 會連上任何人。** 憑證驗證做兩件事：確認憑證是信任的 CA 簽的，以及確認它屬於你要連的那個名稱。`InsecureSkipVerify` 兩件都不做，加密還在，但你不知道對面是誰。下面用 `httptest.NewTLSServer` 當冒充的伺服器，它的憑證不是任何 CA 簽的：

```go,ignore
{{#include ../../../examples/chapter11/06_crypto_rand_tls/before/example_test.go:example}}
```

client 照樣收到「balance: 0」，連線成功，沒有任何警告。

## 較好的做法

**token 用 `crypto/rand`。** `rand.Text` 回傳至少 128 位元的亂數，用 base32 字元編碼，直接能放進網址或信件；要原始位元組用 `rand.Read`，它從 Go 1.24 起不會回傳錯誤。比對用 `subtle.ConstantTimeCompare`，它對長度不同的輸入直接回 0，token 長度本來就不是秘密：

```go,ignore
{{#include ../../../examples/chapter11/06_crypto_rand_tls/after/authkit.go:token}}
```

測試不斷言值，只斷言契約：長度至少 26 個字元（128 位元除以 base32 每字元 5 位元）、字元都在 A-Z 與 2-7 裡。把 `rand.Text` 換成 `math/rand/v2` 的 `Uint32`，這個測試就失敗。不可預測這件事本身測不出來，它來自 `crypto/rand` 的文件保證：

```go,ignore
{{#include ../../../examples/chapter11/06_crypto_rand_tls/after/authkit_test.go:token}}
```

**client 信任指定的 CA，不是關驗證。** 私有 CA 的憑證錯誤，解法是把那張 CA 憑證交給 client，驗證保持開啟。`NewClient` 只接受 PEM，裡面沒有憑證就回錯誤，不會悄悄退回「信任空集合」；`Timeout` 是前作〈HTTP client〉教過的。最低版本設 TLS 1.2 其實就是 Go 目前的預設，寫出來是讓意圖看得見：

```go,ignore
{{#include ../../../examples/chapter11/06_crypto_rand_tls/after/authkit.go:tls}}
```

測試用的憑證在記憶體裡產生，私鑰不寫進檔案，也不放進 repo；`httptest` 內建的憑證所有伺服器共用，沒辦法用來扮演兩台不同的伺服器：

```go,ignore
{{#include ../../../examples/chapter11/06_crypto_rand_tls/after/helpers_test.go:helpers}}
```

信任 CA 的伺服器連得上；另一台名稱與位址都一樣、但私鑰不同的伺服器被拒絕，錯誤型別是 `x509.UnknownAuthorityError`。把 `NewClient` 改回 `InsecureSkipVerify: true`，冒充者的測試就失敗：

```go,ignore
{{#include ../../../examples/chapter11/06_crypto_rand_tls/after/authkit_test.go:client}}
```

**伺服器依 client 決定最低版本。** `ServerConfig` 要求 TLS 1.3，因為呼叫它的都是自己控制的服務。只支援到 TLS 1.2 的 client 會收到 `protocol version not supported`。拿掉 `MinVersion`，預設值是 TLS 1.2，這個測試就失敗，所以它證明的是「1.3 這個選擇」，不是預設值：

```go,ignore
{{#include ../../../examples/chapter11/06_crypto_rand_tls/after/authkit_test.go:version}}
```

## 取捨

- **Go 的 TLS 預設值已經合理。** 依 `crypto/tls` 的文件，最低版本預設是 TLS 1.2；`CipherSuites` 是 `nil` 時用安全的預設清單，而且清單會隨 Go 的版本改變。多數服務不設 `CipherSuites`，升級 Go 就會跟上；自己列密碼套件，等於接下追蹤更新的工作。
- **什麼時候才改。** 所有 client 都是自己控制的內部服務，可以把最低版本拉到 1.3；對外、client 不受控的服務保留預設的 1.2。為了相容舊 client 把版本降到 1.2 以下，要先確認真的有那樣的 client，以及它們多久之後會升級。
- **只信任私有 CA，就連不到公開網站。** `NewClient` 的信任集合只有傳進去的 CA；同一個 client 還要連公開服務，就從 `x509.SystemCertPool()` 起頭再加。能分成兩個 client，攻擊面比較小。
- **CA 檔和私鑰要管。** 哪些檔案是秘密、放在哪、怎麼輪替，本章〈秘密管理〉談過；憑證過期是另一個常見的事故，提早輪替比事後補救便宜。
- **密碼不是 token。** 使用者密碼要用雜湊儲存，用 `golang.org/x/crypto` 的 bcrypt 或 argon2；這集的 `crypto/rand` 與 `subtle` 管的是隨機的 token，不是密碼。
- **token 還有很多議題。** 有效期限、用過即作廢、錯誤次數限制，這集都沒談。6 位數的重設碼再怎麼不可預測，沒有次數限制也能被窮舉；`math/rand/v2` 仍然適合模擬、抽樣、測試用的資料這些與安全無關的地方。

`InsecureSkipVerify` 在本機的一次性實驗、連的是自己剛啟動的伺服器時，不會出大事；問題是它很容易跟著程式碼一起進正式環境。

## 重點整理

- 要給別人猜不到的值（token、重設碼、session ID），用 `crypto/rand` 的 `rand.Text` 或 `rand.Read`；`math/rand/v2` 的文件明說不適合與安全有關的工作。
- 比對秘密用 `subtle.ConstantTimeCompare`；這個差別測試量不出來，理由來自它的契約。
- `InsecureSkipVerify: true` 同時關掉 CA 與名稱的檢查；憑證錯誤的解法是讓 client 信任正確的 CA。
- Go 的 TLS 預設值已經合理；只在 client 全是自己控制的服務時往上調，對外的服務保留預設。
- 測試用 `httptest` 與記憶體裡產生的憑證，斷言握手成功或失敗，不斷言亂數的值。
