[![CC BY-NC-ND 4.0](https://img.shields.io/badge/content-CC%20BY--NC--ND%204.0-blue)](https://creativecommons.org/licenses/by-nc-nd/4.0/)
[![CC0 1.0](https://img.shields.io/badge/code-CC0%201.0-blue)](https://creativecommons.org/publicdomain/zero/1.0/)

# 邁向資深 Go 工程師

寫給已經會 Go 的工程師：套件怎麼切、錯誤怎麼設計、效能從哪裡查起、服務怎麼上線與維運（Go 1.27），請閱讀[線上版](https://holmeslin.github.io/learning-go-to-senior/zh-TW/)。

本書是《[從零開始學 Go](https://holmeslin.github.io/learning-go-from-zero/zh-TW/)》的續作，前置知識就是前作的內容。

## 本機建置

需要 Go 1.27 與 [mdBook](https://github.com/rust-lang/mdBook) 0.5.4。以下指令與 CI 相同：

```bash
go run ./tools/checkcode zh-TW/src   # 驗證書中的單檔 Go 範例
go vet ./... && go test -race ./...  # 驗證多檔案範例
cd zh-TW && mdbook build             # 建置（mdbook serve 可本機預覽）
```

## 範例程式碼

範例有兩種：

- **單檔範例**：寫在 markdown 的 ```` ```go ```` 區塊，由 `checkcode` 抽出驗證。預設會跑 `go vet` 與 `go run`，必須成功結束。可在 info string 加上標註：

  | 標註 | 意義 |
  | --- | --- |
  | `go,compile_fail` | 必須編譯失敗 |
  | `go,stdin=3\n5` | 執行時餵入 stdin（`\n` 代表換行，值內不可有空白或逗號） |
  | `go,exit=2` | 預期的結束碼（例如 panic、死結） |
  | `go,norun` | 只編譯與 `go vet`，不執行（伺服器、signal） |
  | `go,ignore` | 不驗證（`{{#include}}` 引入的程式碼與不完整的片段） |

  這些範例共用 `tools/checkcode/deps/go.mod`；需要第三方套件時在那裡鎖定版本（目前沒有任何相依）。

- **多檔案範例**：放在 repo 根目錄 module 底下的 `examples/`（隨第 1 章的第一個範例建立），按章、集分目錄，每集再分 `before` 與 `after`。書中用 mdBook 的 `{{#include}}` 引入片段，所以書上顯示的就是被 `go vet` 與 `go test -race` 驗證過的程式碼。

## 授權

除非另有標示，本書的文字、圖片與非程式碼內容以 [CC BY-NC-ND 4.0](https://creativecommons.org/licenses/by-nc-nd/4.0/) 授權。

本書中的程式碼範例以 [CC0 1.0](https://creativecommons.org/publicdomain/zero/1.0/) 釋出，可自由複製、修改與使用。
