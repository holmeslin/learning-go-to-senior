# 重試、退避與 jitter

## 本集目標

下游呼叫失敗時，判斷哪些錯誤值得重試、每次等多久、什麼時候該放棄，並說得出重試為什麼本身也是負載。

## 情境

結帳服務要向匯率服務查匯率。匯率服務偶爾逾時，過載時回 503；結帳服務有很多個實例，每個實例同時處理很多請求，每個請求都帶著呼叫端願意等的 deadline。

HTTP client 與 `context` 的逾時前作教過：〈[HTTP client](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter14/06_http_client.html)〉、〈[`WithTimeout` 與 `WithDeadline`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter10/03_with_timeout.html)〉。逾時怎麼分配，本章〈逾時預算〉談過，這集只談失敗之後的重試。操作不是冪等的時候重試會做兩次，本章〈冪等性〉會談；下游已經掛了還要不要打，本章〈斷路器〉會談。

## 常見寫法

失敗就再試，直到成功，`ctx` 是唯一的煞車。`base` 是 0 就立刻重試；發現立刻重試會壓垮下游的人，通常會加上指數退避，第 n 次失敗後等 `base<<n`：

```go,ignore
{{#include ../../../examples/chapter10/02_retry_backoff_jitter/before/rates.go:fetch}}
```

完整程式碼在 `examples/chapter10/02_retry_backoff_jitter/before/`。

## 問題在哪

**每種錯誤都重試，而且沒有上限。** 400 代表請求本身有問題，送再多次結果也一樣。下面的假下游對每個請求回 400，並在第 100 次呼叫時取消 `ctx`，否則 `Fetch` 永遠不會停：

```go,ignore
{{#include ../../../examples/chapter10/02_retry_backoff_jitter/before/example_test.go:example}}
```

**立刻重試，下游在同一個瞬間被打 50 次以上。** 測試用假下游：不開 port，直接當 `http.Client` 的 Transport，記下每次呼叫的假時間。時間用 `testing/synctest`（見第 5 章〈[測時間與並行：用 `testing/synctest`，還是自己注入時鐘](../chapter5/08_synctest_or_clock.md)〉），bubble 裡的時間是確定的，所以可以斷言每次呼叫落在哪個時間點：

```go,ignore
{{#include ../../../examples/chapter10/02_retry_backoff_jitter/before/retry_test.go:fake}}
```

10 個 client 遇到過載的下游，前 50 次呼叫都失敗。立刻重試不會讓時間往前走，50 次失敗全擠在同一個時間點：

```go,ignore
{{#include ../../../examples/chapter10/02_retry_backoff_jitter/before/retry_test.go:immediate}}
```

**加了退避、沒有 jitter，重試還是同步的。** 同一刻失敗的 10 個 client 算出同樣的等待，每一輪都同時抵達。剛恢復的下游，每一輪都被同時湧入的 10 次呼叫再打一次：

```go,ignore
{{#include ../../../examples/chapter10/02_retry_backoff_jitter/before/retry_test.go:lockstep}}
```

下游重啟或網路抖動一過，所有 client 一起失敗，也就一起重試。重試是額外的負載，而且在最不該有負載的時候到達。

## 較好的做法

把重試拆成三個決定：值不值得重試、等多久、還等不等得起。

`Policy` 收著規則。等待時間用 full jitter：第 n 次失敗後，在 `[0, min(Max, Base<<n))` 之間隨機取值，下游用 `Retry-After` 要求的秒數當下限，加在上面：

```go,ignore
{{#include ../../../examples/chapter10/02_retry_backoff_jitter/after/rates.go:policy}}
```

`Jitter` 是為了測試才匯出的接縫：正式環境留空，用 `math/rand/v2` 的全域亂數（前作〈[`math/rand/v2`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter12/11_rand_v2.html)〉）；測試注入固定 seed 的 `*rand.Rand`，或固定的值，讓等待時間可以斷言。`StatusError` 匯出，是因為呼叫端要用 `errors.AsType` 分辨 400 與 503。

`Fetch` 只重試暫時性的錯誤：單次嘗試逾時、連線被拒、429 與 503，其他錯誤立刻回傳。每次等待之前先看 deadline，等不起就現在回錯誤，不要睡到 deadline 才失敗；等待時 `ctx` 結束也立刻回傳：

```go,ignore
{{#include ../../../examples/chapter10/02_retry_backoff_jitter/after/rates.go:fetch}}
```

after 的測試放在外部測試套件，只用公開 API。假下游可以指定每次回什麼：`hang` 卡到 `ctx` 結束，`refused` 連線被拒，`status` 回指定的狀態碼與 `Retry-After`；`half` 是測試用的 jitter，永遠取上限的一半，讓等待時間是確定值：

```go,ignore
{{#include ../../../examples/chapter10/02_retry_backoff_jitter/after/retry_test.go:fake}}
```

第一個測試斷言等待的上限依序是 1 秒、2 秒、3 秒（之後被 `Max` 截斷），每次呼叫的時間點都是確定值；第二個用固定 seed 的亂數，斷言每次等待都落在 `[0, 上限)`：

```go,ignore
{{#include ../../../examples/chapter10/02_retry_backoff_jitter/after/retry_test.go:backoff}}
```

逾時與連線被拒會重試：第一次卡到單次逾時的 2 秒，第二次被拒，第三次成功。400 只呼叫一次，也不等待：

```go,ignore
{{#include ../../../examples/chapter10/02_retry_backoff_jitter/after/retry_test.go:classify}}
```

`Retry-After` 是下限，jitter 加在上面。deadline 只剩 5 秒、下游卻要求等 10 秒時，立刻回錯誤，讓呼叫端有時間做別的事；等待中 `ctx` 被取消，在取消的那一刻就回傳：

```go,ignore
{{#include ../../../examples/chapter10/02_retry_backoff_jitter/after/retry_test.go:wait}}
```

jitter 的效果也要證明。10 個 client 各自用固定 seed 的 `*rand.Rand`。把 jitter 換成直接回傳上限（等於沒有 jitter），10 次重試落在同一個時間點；有 jitter 時，同一個時間點最多一次：

```go,ignore
{{#include ../../../examples/chapter10/02_retry_backoff_jitter/after/retry_test.go:spread}}
```

次數也可以用 `Example` 看：503 重試到成功，400 只呼叫一次（`Base` 是 0，所以不用真的等）：

```go,ignore
{{#include ../../../examples/chapter10/02_retry_backoff_jitter/after/example_test.go:example}}
```

## 取捨

**重試放大負載。** 呼叫鏈上每一層都重試，次數會相乘：三層各自最多嘗試 3 次，最底層的下游最壞要承受 27 次。重試只在一層做，通常是最靠近使用者、知道整體 deadline 的那一層，其他層把錯誤往上回。

**重試預算。** 下游整個壞掉時，每個請求都重試，流量變成好幾倍。可以限制重試占全部請求的比例（例如一成），超過就不重試、直接回錯誤。這份預算要跨請求共用，是有狀態的，這集只提概念；要看得出放大了幾倍，得先把每次重試記成指標，指標的設計見第 9 章〈[指標（metrics）設計與高基數問題](../chapter9/03_metrics_cardinality.md)〉。

**jitter 的變體。** 這裡用 full jitter。equal jitter 保留一半固定的退避，decorrelated jitter 用上一次的等待決定下一次，差別見 AWS Architecture Blog 的〈Exponential Backoff And Jitter〉。只要有 jitter，同步的重試就會散開；選哪一種是次要的。

**重試會拉長延遲。** 重試成功的請求比第一次就成功的慢；有人在等的請求，deadline 要夠放得下一次重試，否則重試只是讓失敗晚一點才被看見。

**`Retry-After` 與 deadline。** 範例只讀秒數的寫法，RFC 9110 也允許 HTTP-date。沒有 deadline 的 `ctx` 會乖乖等下游要求的時間，所以呼叫端一定要設 deadline。

**before 什麼時候夠用？** 一個 client 的批次作業，錯誤確定是暫時的，下游不會被它的重試影響，沒有人在等。即使如此，上限與 deadline 還是要有。要更多功能（預算、各種退避）時，可以看 `cenkalti/backoff` 這類套件，或把重試交給 service mesh。

## 重點整理

- 只重試暫時性的錯誤（逾時、連線被拒、429、503）；400 這類請求本身的問題，重試也沒用。
- 重試要有上限；等待用指數退避加 jitter，下游的 `Retry-After` 當下限。
- 沒有 jitter，同一刻失敗的 client 會在同一刻重試；jitter 讓它們散開。
- 等待之前先看 deadline，等不起就現在回錯誤；`ctx` 結束立刻回傳。
- 重試也是負載：只在一層重試，並考慮重試預算。
- 退避的測試用 `testing/synctest` 斷言假時間，jitter 用注入的亂數與固定 seed，不需要 `time.Sleep`。
