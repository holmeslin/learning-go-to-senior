# SSRF 與對外請求

## 本集目標

判斷伺服器替使用者抓取網址時，防線該放在哪：不是網址的字串，而是連線建立那一刻實際要連的 IP；同時限制轉址、逾時與回應大小。

## 情境

文件協作服務有「貼上連結就顯示預覽」的功能：使用者貼一個網址，伺服器去抓頁面。這個請求是伺服器發的，伺服器在內網裡，連得到使用者連不到的東西：本機的管理介面、內網的服務、雲端主機的 metadata 端點。使用者能指定網址，就能叫伺服器替他去連，這就是 SSRF（server-side request forgery）。

HTTP client 與 `httptest` 前作教過：〈[HTTP client](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter14/06_http_client.html)〉、〈[逾時設定](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter14/07_timeouts.html)〉、〈[`httptest`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter14/09_httptest.html)〉。網址只是輸入的一種，一般的輸入驗證，本章〈[輸入驗證](01_input_validation.md)〉談過；這集處理的是「字串合法，連到的地方不合法」。對外請求失敗要不要重試，第 10 章〈[重試、退避與 jitter](../chapter10/02_retry_backoff_jitter.md)〉談過，這集的 `Fetch` 只發一次請求。

## 常見寫法

先檢查網址：主機不能是 `localhost`，也不能是 loopback、私有或 link-local 的 IP 字面值；通過就直接請求，`client` 傳 `http.DefaultClient` 就等於 `http.Get`：

```go,ignore
{{#include ../../../examples/chapter11/07_ssrf_outbound_requests/before/preview.go:fetch}}
```

這個檢查用 `net/netip` 解析 IP，比比對字串前綴嚴謹。完整程式碼在 `examples/chapter11/07_ssrf_outbound_requests/before/`，套件的文件註解寫明這是刻意示範的漏洞。

## 問題在哪

檢查的是字串，連線連的是解析之後的 IP，兩者中間隔著 DNS 與轉址。下面的測試只用 `httptest` 的 loopback 伺服器：`internal` 當作內網的管理介面，`attacker` 是攻擊者的網站，任何路徑都轉址到 `internal`。`dnsClient` 模擬 DNS 紀錄，讓兩個看起來公開的名稱指向它們：

```go,ignore
{{#include ../../../examples/chapter11/07_ssrf_outbound_requests/before/example_test.go:example}}
```

直接寫 `127.0.0.1` 被擋下，但另外兩個都讀到內部頁面。攻擊者註冊自己的網域，把 A 紀錄指向內部位址，主機名稱看起來完全正常；或者給一個公開的網址，伺服器回 302，`http.Client` 預設會跟著轉址最多 10 次，檢查只在第一個網址做過。更狡猾的 DNS rebinding 讓檢查時查到公開位址、連線時查到內部位址。

`http.Get` 也沒有逾時，回應全部讀進記憶體：

```go,ignore
{{#include ../../../examples/chapter11/07_ssrf_outbound_requests/before/example_test.go:limits}}
```

## 較好的做法

防線放在繞不過去的地方：連線建立之前，對實際要連的 IP。`net.Dialer` 的 `Control` 在名稱解析完、socket 建立後、真正連線前被呼叫，拿到的是已解析的 `ip:port`。檢查與使用的是同一個值，rebinding 沒有空窗；每次轉址都會新建連線，也會經過這裡。先看判斷規則：不是公開單播位址的一律拒絕，`IsGlobalUnicast` 把私有位址當成單播，所以要另外排除，再加上 `netip` 沒涵蓋的特殊範圍：

```go,ignore
{{#include ../../../examples/chapter11/07_ssrf_outbound_requests/after/preview.go:policy}}
```

`Fetcher` 把所有限制放在欄位上，零值就是安全的預設：只連公開位址、5 秒、不跟隨轉址、1 MiB。`Allow` 與 `Resolver` 為了測試而匯出：httptest 的伺服器在 loopback，預設規則一定擋下，測試要用 `Allow` 放行指定的假伺服器，用 `Resolver` 把名稱解析到 loopback，才能在不連外網的前提下測「允許的可以連」與「名稱指向內部」。`Allow` 只增不減：公開位址本來就能連，`Allow` 回傳 true 的位址是額外的例外，所以正式環境可以用 `Allow` 放行少數已知的內部位址，不必重寫整套規則。

```go,ignore
{{#include ../../../examples/chapter11/07_ssrf_outbound_requests/after/preview.go:fetcher}}
```

`Fetch` 用 `io.LimitReader` 多讀一個位元組，超過上限就回錯誤，不截斷後假裝成功。`Client.Timeout` 涵蓋整個請求，包含轉址與讀取回應：

```go,ignore
{{#include ../../../examples/chapter11/07_ssrf_outbound_requests/after/preview.go:fetch}}
```

after 的測試放在外部測試套件。`serve` 數出假伺服器收到幾個請求，`only` 是注入的允許清單，`Allowed` 由 `export_test.go` 匯出、只給測試用，讓測試直接問「這個位址能不能連」，不必連到真的公開位址。`TestAllowAddsToPublic` 鎖住 `Allow` 只增不減：放行一個內部位址之後，公開位址仍然可連，另一個內部位址仍被擋。證明「被擋下」不能只看錯誤，還要看伺服器收到 0 個請求：

```go,ignore
{{#include ../../../examples/chapter11/07_ssrf_outbound_requests/after/fetch_test.go:helpers}}
```

```go,ignore
{{#include ../../../examples/chapter11/07_ssrf_outbound_requests/after/fetch_test.go:blocked}}
```

轉址到內部位址同樣被擋，內部伺服器沒收到請求；轉址次數超過上限就停：

```go,ignore
{{#include ../../../examples/chapter11/07_ssrf_outbound_requests/after/fetch_test.go:redirect}}
```

名稱指向內部位址的測試，用 loopback 上的 UDP 小伺服器當 DNS，把所有名稱解析成 `127.0.0.1`：

```go,ignore
{{#include ../../../examples/chapter11/07_ssrf_outbound_requests/after/dns_test.go:fakedns}}
```

對照組先放行那個位址，證明名稱解析成功，後面的拒絕才有意義：

```go,ignore
{{#include ../../../examples/chapter11/07_ssrf_outbound_requests/after/dns_test.go:dns}}
```

逾時與回應大小，以及判斷規則的各種位址。`special` 的每個範圍各有一列，拿掉任何一個範圍，測試都會失敗；範圍外一格的位址也要測，確認沒有誤擋：

```go,ignore
{{#include ../../../examples/chapter11/07_ssrf_outbound_requests/after/fetch_test.go:limits}}
```

```go,ignore
{{#include ../../../examples/chapter11/07_ssrf_outbound_requests/after/fetch_test.go:ispublic}}
```

拿掉 `Control` 或轉址上限，對應的測試與 `Example` 會失敗。`Example` 把兩種攻擊放在一起：

```go,ignore
{{#include ../../../examples/chapter11/07_ssrf_outbound_requests/after/example_test.go:example}}
```

## 取捨

- **允許清單與拒絕清單。** 只能連事先登記的主機（允許清單）最穩，webhook 的對象固定時就該這樣做。預覽功能要抓任意網址，只能用拒絕清單：`IsPublic` 排除已知的特殊範圍。拒絕清單的代價是會漏：新的保留範圍、你自己的網路裡用了公開位址卻不該被連的服務。
- **IPv6 與特殊位址。** IPv4-mapped 位址要先還原；NAT64、6to4、Teredo 可以內嵌 IPv4，所以整段排除。CGNAT 常被內部網路與 VPN 使用。範圍清單要隨標準更新，測試表是它的規格。
- **程式這一層擋不住的。** `Control` 只管 TCP 連線：DNS 查詢本身照樣發出；走 proxy 時 `Control` 看到的是 proxy 的位址，所以範例不用 `ProxyFromEnvironment`，需要 proxy 時檢查要放在 proxy 上。網路層的隔離是另一道獨立的防線：讓抓取的工作跑在沒有內網路由的網段，或強制經過只放行公開位址的 egress proxy，即使程式有漏洞也連不到。
- **回應與錯誤也是出口。** 連得上的公開頁面，內容不要原樣轉給使用者，預覽只回標題與摘要；回給使用者的錯誤要一致，否則「被擋」與「連線被拒」的差別，會讓人用來探測內網（本章〈[輸入驗證](01_input_validation.md)〉談過錯誤訊息不洩漏內部細節）。`BlockedError` 帶著位址，是給日誌看的。
- **成本。** 每次 `Fetch` 建立新的 `http.Client` 且關閉連線重用，換來連線不會跨使用者共用，代價是每次都重新握手。
- **before 什麼時候夠用？** 網址是設定檔裡寫死的，不是使用者給的。即使如此，逾時與大小上限還是要有。

## 重點整理

- SSRF 是伺服器替使用者連到使用者連不到的地方；只檢查網址字串，會被 DNS 與轉址繞過。
- 防線放在 `net.Dialer.Control`：連線前、對實際的 IP、每次轉址都檢查，檢查與使用是同一個值。
- 同時設定轉址次數、整體逾時與回應大小的上限，零值要是安全的預設。
- 測試用 loopback 伺服器、注入的允許清單與 resolver；斷言錯誤型別，也斷言被擋下的伺服器收到 0 個請求。
- 允許清單最穩，拒絕清單會漏；程式這一層之外，還要有網路層的隔離。
