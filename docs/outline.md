# 邁向資深 Go 工程師 — 全書大綱（已定案）

Go 1.27 · 繁中 · 讀者已會前作《從零開始學 Go》的內容（不要求讀過前作）· 標準庫與 `golang.org/x` 優先，OpenTelemetry 與 `golangci-lint` 只在對應章節使用，不教框架

章名與 `zh-TW/src/SUMMARY.md` 一致。每章列出要涵蓋的主題；一個主題對應一集或多集，實際切分在各集的 ticket 決定，單集超過篇幅上限就拆集（見風格指南）。

# 第一部　程式碼設計

## 第 1 章 套件與 API 設計
01 套件切分的依據（依職責，不依型別）· 02 套件命名與 `util` 陷阱 · 03 `internal` 與公開面積 · 04 匯出名稱與 API 穩定性 · 05 建構函式與 functional options · 06 讓零值可用的設計 · 07 循環相依與依賴方向 · 08 棄用（deprecation）與相容性

## 第 2 章 錯誤設計
01 哨兵錯誤、自訂型別、不透明錯誤的選擇 · 02 包裝錯誤：`%w` 該不該暴露給呼叫端 · 03 在哪一層處理、哪一層只回傳 · 04 錯誤訊息怎麼寫 · 05 `errors.Join` 與多重錯誤 · 06 `panic` 的使用邊界 · 07 錯誤是 API 契約的一部分

## 第 3 章 抽象的取捨
01 在使用端定義介面 · 02 接受介面、回傳 struct · 03 介面何時是多餘的 · 04 泛型、介面、重複三者的取捨 · 05 過早抽象的代價 · 06 嵌入與組合 · 07 不用框架的相依注入

## 第 4 章 資料存取的設計
01 要不要有 repository 這一層 · 02 交易邊界放在哪裡 · 03 `context` 與查詢逾時 · 04 連線池設定 · 05 N+1 與查詢組裝 · 06 migration · 07 測試資料層：fake 與真實資料庫

# 第二部　測試

## 第 5 章 測試策略
01 測什麼、不測什麼 · 02 表格驅動測試的邊界 · 03 fake、stub、mock 的選擇 · 04 測試替身放在哪裡 · 05 整合測試與 build tags · 06 golden file · 07 fuzzing 的實務用法 · 08 `testing/synctest` 測時間與並行 · 09 `httptest` · 10 不穩定（flaky）測試 · 11 覆蓋率的意義與限制

# 第三部　並行實務

## 第 6 章 結構化並行
01 goroutine 的生命週期歸誰管 · 02 `errgroup` · 03 `errgroup` 的並行上限 · 04 取消與 `context` 的傳遞 · 05 fan-out / fan-in 的取捨 · 06 worker pool 與每個任務一個 goroutine · 07 channel 與 mutex 怎麼選 · 08 背壓 · 09 偵測 goroutine 洩漏

# 第四部　效能

## 第 7 章 效能調查流程
01 先量測，再最佳化 · 02 benchmark 與 `b.Loop` · 03 `benchstat` 比較結果 · 04 `pprof`：CPU 與 heap · 05 `runtime/trace` · 06 常見熱點：配置、拷貝、鎖競爭 · 07 用逃逸分析結果找配置 · 08 PGO · 09 把效能回歸擋在 CI 之外

## 第 8 章 runtime
01 排程器（GMP）怎麼看 · 02 GC 與 `GOGC` · 03 `GOMEMLIMIT` 與容器記憶體上限 · 04 記憶體配置與 heap 成長 · 05 容器內的 `GOMAXPROCS` · 06 `runtime/metrics` · 07 阻塞、syscall 與排程 · 08 讀 GC trace

# 第五部　上線與維運

## 第 9 章 可觀測性
01 `log/slog` 結構化日誌 · 02 日誌層級與欄位設計 · 03 指標（metrics）設計與高基數問題 · 04 分散式追蹤與 OpenTelemetry · 05 在 `context` 裡傳遞 trace · 06 `/debug/pprof` 與診斷端點 · 07 健康檢查 · 08 SLI 與 SLO

## 第 10 章 韌性
01 逾時預算 · 02 重試、退避與 jitter · 03 冪等性 · 04 `golang.org/x/time/rate` 限流 · 05 斷路器 · 06 graceful shutdown · 07 負載卸載與降級 · 08 bulkhead

## 第 11 章 安全實務
01 輸入驗證 · 02 SQL injection 與參數化查詢 · 03 `os.Root` 與路徑穿越 · 04 `govulncheck` 與相依弱點 · 05 秘密管理 · 06 `crypto/rand` 與 TLS 設定 · 07 SSRF 與對外請求 · 08 日誌裡的敏感資料

## 第 12 章 建置與發布
01 `go build` 與用 `-ldflags` 注入版本 · 02 可重現的建置 · 03 交叉編譯 · 04 容器映像（multi-stage）· 05 `golangci-lint` 與 CI · 06 `go.mod` 與相依升級 · 07 發布流程與版本號 · 08 設定與環境變數

# 第六部　在團隊裡寫 Go

## 第 13 章 code review
01 先看設計，再看細節 · 02 Go 程式碼的 review 檢查清單 · 03 PR 大小與拆分 · 04 怎麼寫 review 意見 · 05 作者的自我審查 · 06 什麼時候該擋、什麼時候可以放行

## 第 14 章 設計文件
01 什麼時候需要寫 · 02 結構：背景、目標、非目標、方案、取捨 · 03 寫給誰看 · 04 替代方案怎麼寫 · 05 設計文件的 review · 06 設計文件與 ADR 的關係

## 第 15 章 事故處理
01 事故中的角色與溝通 · 02 先止血，再找原因 · 03 用 `pprof` 與 trace 在現場診斷 · 04 事後檢討（blameless postmortem）· 05 行動項目與追蹤 · 06 runbook

# 第七部　綜合實戰

## 第 16 章 一個服務從設計到上線
01 設計文件 · 02 套件結構與 API · 03 錯誤與資料層 · 04 測試 · 05 並行 · 06 可觀測性與韌性 · 07 建置與發布 · 08 壓測與調校 · 09 上線檢查表
