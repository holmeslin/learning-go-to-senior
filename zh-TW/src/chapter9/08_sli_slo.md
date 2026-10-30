# SLI 與 SLO

## 本集目標

分清 SLI、SLO、SLA；判斷延遲的直方圖 bucket 邊界要放在哪裡才算得出確定的 SLI；用錯誤預算與消耗速度，而不是「出現任何 5xx」決定什麼時候告警。

## 情境

搜尋服務的 API 有兩個目標：99.9% 的請求不回 5xx，99% 的請求在 300 ms 內完成。三個名詞常被混著用：

- **SLI**（service level indicator）：量出來的數字，通常是好請求數除以全部請求數。
- **SLO**（service level objective）：SLI 要達到的目標，附帶一個窗口，例如 30 天內 99.9%。
- **SLA**（service level agreement）：對客戶的合約，沒達到要賠償或退費。SLO 通常訂得比 SLA 嚴，讓團隊在違約之前先發現。

延遲是用直方圖收的。指標的種類與屬性怎麼選，本章〈[指標（metrics）設計與高基數問題](03_metrics_cardinality.md)〉談過；第 8 章〈[`runtime/metrics`](../chapter8/06_runtime_metrics.md)〉提到直方圖的 bucket 要自己決定怎麼彙整，這集回答邊界怎麼挑、怎麼彙整。

## 常見寫法

延遲用毫秒記進直方圖，沒有指定 bucket，用 SDK 的預設值；狀態碼類別（`5xx`）當屬性：

```go,ignore
{{#include ../../../examples/chapter9/08_sli_slo/before/slo.go:recorder}}
```

要算「300 ms 內的比例」，就去直方圖裡數 bucket。告警規則是「出現任何 5xx 就通知值班的人」。完整程式碼在 `examples/chapter9/08_sli_slo/before/`。

## 問題在哪

直方圖只保存每一格的請求數，不保存每個請求的延遲。SDK 的預設邊界是 `0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000`（bucket 包含上界）。300 不在裡面，落在 (250, 500] 那一格。`Within` 能做的，是算出下限（不含那一格）與上限（含整格）：

```go,ignore
{{#include ../../../examples/chapter9/08_sli_slo/before/slo.go:within}}
```

下面用兩批固定的資料，各 100 個請求：96 個花 120 ms，其餘 4 個，A 批花 260 ms，B 批花 400 ms。

```go,ignore
{{#include ../../../examples/chapter9/08_sli_slo/before/example_test.go:example}}
```

兩批資料在直方圖裡一模一樣：96 個在 (100, 250]，4 個在 (250, 500]。實際上 A 批全部達標，B 批只有 96%，不到 99%。範圍 96～100% 橫跨了 SLO 的 99%，SLI 到底有沒有達標，直方圖回答不了。要給單一數字，只能假設那一格裡的請求平均分布，這是假設，不是量測。

告警的問題在另一邊。`Alert` 只要看到 5xx 的資料點就回報：

```go,ignore
{{#include ../../../examples/chapter9/08_sli_slo/before/slo.go:alert}}
```

一萬個請求裡只有一個 500：

```go,ignore
{{#include ../../../examples/chapter9/08_sli_slo/before/example_test.go:alertexample}}
```

99.9% 的 SLO 本來就允許一萬個請求裡有十個失敗，這一個不需要叫醒任何人。反過來，這條規則看不出「每分鐘一個 5xx，持續一整個月」這種慢慢用光預算的情況，它只看有沒有，不看多快。

## 較好的做法

先讓 bucket 邊界包含 SLO 的門檻。直方圖由 `NewRecorder` 建立，邊界不寫在記錄的程式裡，而是用 View 在建立 `MeterProvider` 時指定；延遲改記秒，單位依指標命名的慣例。門檻與邊界用同一個常數 `LatencyThreshold`，不會各改各的：

```go,ignore
{{#include ../../../examples/chapter9/08_sli_slo/after/slo.go:recorder}}
```

`Compute` 從 `ManualReader` 收到的資料算 SLI。邊界相同的直方圖可以彙整：把每個資料點（這裡是每個狀態碼類別）對應的格子逐格相加，就得到整體的直方圖；邊界不同的直方圖不能相加。門檻本身是邊界時，「門檻內的請求」就是這個邊界以下所有格子的總和，是數出來的。沒有這個邊界時回傳 `ErrNoBoundary`，不去估：

```go,ignore
{{#include ../../../examples/chapter9/08_sli_slo/after/slo.go:compute}}
```

測試用固定的請求資料，不量真實耗時。同一批請求在沒有 View 時得到 `ErrNoBoundary`：

```go,ignore
{{#include ../../../examples/chapter9/08_sli_slo/after/slo_test.go:traffic}}
```

```go,ignore
{{#include ../../../examples/chapter9/08_sli_slo/after/slo_test.go:noboundary}}
```

加上 View 之後，前面 A、B 兩批資料的 SLI 就分開了：

```go,ignore
{{#include ../../../examples/chapter9/08_sli_slo/after/example_test.go:example}}
```

錯誤預算（`Budget`）是「SLO 允許的壞請求數」。目標 99.9%、一萬個請求，預算是 10 個。消耗速度是實際的壞請求比例除以允許的比例：1 代表剛好在窗口結束時用完預算，2 代表兩倍快，窗口走到一半就沒了：

```go,ignore
{{#include ../../../examples/chapter9/08_sli_slo/after/slo.go:result}}
```

告警看消耗速度，`ShouldPage` 在速度達到 `FastBurn` 時回報要叫醒人。`FastBurn` 是 14.4：30 天的窗口，以這個速度燒 1 小時，用掉 2% 的預算（14.4 ÷ 720 小時）；數字出自 Google SRE Workbook 的〈[Alerting on SLOs](https://sre.google/workbook/alerting-on-slos/)〉。表格測試鎖住 SLI、剩餘預算與消耗速度，包含「剛好 300 ms」算達標、沒有請求不除以零：

```go,ignore
{{#include ../../../examples/chapter9/08_sli_slo/after/slo_test.go:table}}
```

同樣是 5xx，一個 500 只燒掉 0.1 倍的速度，不告警；兩百個 503 是 20 倍，告警：

```go,ignore
{{#include ../../../examples/chapter9/08_sli_slo/after/example_test.go:burn}}
```

## 取捨

- **SLO 不是越高越好。** 以 30 天計，99.9% 允許約 43 分鐘的壞請求，99.99% 只剩約 4.3 分鐘，預算少了 10 倍：一次部署、一次依賴的抖動就可能用完，團隊只能凍結功能、補冗餘與自動化，成本跟著上去。先看歷史資料與使用者真正在意的程度，再訂；訂得比使用者感受得到的更高，是白花力氣。
- **SLI 在哪裡量，會量到不同的東西。** 在伺服器端量，看不到被負載平衡器擋掉、DNS 失敗、使用者端逾時而根本沒到達的請求；在使用者端（瀏覽器、App、定時的探測）量得到，但雜訊多、收集與隱私的成本也高。常見的折衷是在最靠近使用者的元件（閘道、負載平衡器）量，再輔以探測。
- **延遲與可用性要一起看。** 這集的延遲 SLI 把所有請求都算進去，所以很快就失敗的 503 在延遲上算「好」（測試資料裡的 50 ms 的 503）；只看延遲會漏掉它，只看 5xx 又漏掉慢。要不要把失敗的請求從延遲 SLI 排除，是個要明確決定的定義。
- **多窗口的告警只提概念。** `Compute` 只算收到的資料，窗口由誰收集、收多久決定。實務上常同時看長、短兩個窗口的消耗速度：長窗口確認不是一時的雜訊，短窗口確認問題還在發生，事故結束後告警就會停。這集沒有實作；整套服務的應用，第 16 章〈可觀測性與韌性〉會談。
- **邊界要跟著 SLO 維護，也有成本。** 改門檻就要改邊界，新舊邊界的資料不能直接相加。每多一格，後端若把每格存成一條時間序列，成本就多一份，邊界只放需要的幾個。
- **before 不是沒有用。** 只想看延遲的分布（儀表板、找尾巴在哪），估算就夠。沒有承諾 SLO 的內部工具、流量極低的服務，「任何 5xx 就通知」也簡單可靠；要拿它去證明達標，或決定該不該停下功能，就需要這集的做法。

## 重點整理

- SLI 是量出來的好請求比例，SLO 是 SLI 的目標與窗口，SLA 是違約要賠償的合約；SLO 比 SLA 嚴。
- 延遲 SLI 要數出來而不是估：bucket 邊界必須包含 SLO 的門檻，用 View 指定，門檻與邊界共用同一個常數。
- 邊界相同的直方圖逐格相加就能彙整；沒有門檻邊界時 `Compute` 回傳錯誤，不給估計值。
- 錯誤預算是允許的壞請求數，消耗速度是實際比例除以允許的比例；告警看速度，單一個 5xx 不該叫醒人。
- 測試用固定的請求資料斷言 SLI、預算與消耗速度，不斷言耗時。
