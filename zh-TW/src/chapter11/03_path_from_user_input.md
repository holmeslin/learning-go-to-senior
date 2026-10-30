# 路徑來自使用者輸入時，用 `os.Root` 還是自己檢查

## 本集目標

檔名有任何一部分來自使用者時，能認出「`filepath.Join` 加 `strings.HasPrefix`」擋不住哪些輸入，並依目錄裡放的是什麼，決定用 `os.Root`，還是只用 `filepath.IsLocal` 就夠。

## 情境

你在寫報表下載：每晚的排程把 CSV 寫進 `reports` 目錄，使用者在網址帶上檔名，服務把檔案回傳。目錄旁邊還有一個 `reports-private`，放不能外流的檔案（範例裡用明顯是假的字串代替）。`reports` 裡也有一個排程自己用的 `.state`，還有一個指到目錄外的符號連結 `link.csv`。這個目錄配置，before 與 after 的測試共用：

```go,ignore
{{#include ../../../examples/chapter11/03_path_from_user_input/before/setup_test.go:setup}}
```

路徑穿越是什麼、`os.Root` 怎麼用，前作〈[`os.Root`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter12/04_os_root.html)〉與〈[`path/filepath` 與 `io/fs`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter12/05_filepath_fs.html)〉已經講過。這集談的是判斷：自己檢查哪裡會漏，什麼時候該換工具。

## 常見寫法

先 `Join` 成完整路徑，再檢查結果有沒有還在報表目錄底下：

```go,ignore
{{#include ../../../examples/chapter11/03_path_from_user_input/before/reports.go:read}}
```

最常見的測試是 `../outside.csv`，它被擋下，檢查看起來有用。這個套件的文件註解已經寫明是刻意示範的漏洞，完整程式碼在 `examples/chapter11/03_path_from_user_input/before/`。

## 問題在哪

同一個目錄配置，把六種檔名各送一次。`// Output:` 由 `go test` 逐字比對，只印結果，不印暫存目錄的路徑：

```go,ignore
{{#include ../../../examples/chapter11/03_path_from_user_input/before/example_test.go:example}}
```

- **`..` 加相同開頭的目錄會漏。** `Join` 之後是 `…/reports-private/payroll.csv`，而 `HasPrefix` 比的是字串，不是路徑：`…/reports-private` 以 `…/reports` 開頭，檢查就過了。只在前綴後補一個路徑分隔符，這一種可以補起來，但擋不住下一種。
- **符號連結會漏。** 檢查只看字串，`link.csv` 看起來完全在目錄裡；`os.ReadFile` 照著連結走到目錄外。先用 `filepath.EvalSymlinks` 解開再比對，檢查與開檔之間仍有空檔，有人能在這段時間把檔案換成連結（TOCTOU，time-of-check to time-of-use）。
- **絕對路徑沒有漏，但原因不是檢查。** `Join` 把 `/…/payroll.csv` 當成相對路徑接在目錄後面，指到一個不存在的地方。換成「收到絕對路徑就直接使用」的寫法，同樣的檢查就擋不住。
- **只測 `..` 看不出問題。** 這是最常被拿來測的輸入，也是唯一被擋下的一個；其餘幾種要刻意設計才會出現在測試裡。

## 較好的做法

兩道關卡，各擋各的：

```go,ignore
{{#include ../../../examples/chapter11/03_path_from_user_input/after/reports.go:read}}
```

- **`filepath.IsLocal` 先在字串上拒絕。** 它擋掉 `..`、絕對路徑與空字串（Windows 上還有 `NUL` 這類保留名稱），而且不碰檔案系統，錯誤能指出是檔名不合法。它只做字串分析，不知道符號連結。
- **`os.Root` 負責檔案系統這一層。** `Open` 時抓住目錄，之後每次開檔都在目錄之內解析路徑，連結指到目錄外就失敗，沒有檢查與開檔之間的空檔。
- **副檔名是這個服務自己的規則。** `os.Root` 只管「不出目錄」，不管目錄裡哪些檔案該給人看。

測試把每種輸入的結果寫成表，並斷言是哪一關擋下的，符號連結要由 `os.Root` 擋下，不是檔名檢查：

```go,ignore
{{#include ../../../examples/chapter11/03_path_from_user_input/after/reports_test.go:test}}
```

另一個測試證明 `os.Root` 讀得到 `.state`，所以副檔名檢查不是多餘的。`Example` 與 before 用同一組輸入：

```go,ignore
{{#include ../../../examples/chapter11/03_path_from_user_input/after/example_test.go:example}}
```

after 的測試把關卡逐一拿掉，都會失敗：改用 `os.ReadFile(filepath.Join(...))`，符號連結那列讀到目錄外的內容；拿掉 `IsLocal`，`..` 與絕對路徑不再回傳 `ErrInvalidName`（`os.Root` 仍擋得住內容）；拿掉副檔名檢查，`.state` 讀得到。

## 取捨

- **`os.Root` 的限制寫在文件上。** 它跟隨目錄裡的符號連結，但連結不能指到目錄外、也不能是絕對路徑；不禁止穿越檔案系統邊界、Linux bind mount、`/proc` 特殊檔案與裝置檔。Unix 上 `Chmod`、`Chown`、`Chtimes` 有競爭條件；`GOOS=js` 的符號連結驗證有 TOCTOU，擋不住逃逸；plan9 與 `js` 不追蹤目錄改名。服務只讀檔、又只跑在 Linux 或 macOS 時，這些限制多半碰不到；要跨平台，或要用到寫入與改權限的方法，就要逐條對照。
- **它持有一個檔案描述符。** 用完要 `Close`；目錄被搬走後，`Root` 仍指向原來那個目錄。部署時整個目錄被換掉的服務，要重新 `OpenRoot`，不能一直用舊的。
- **`filepath.IsLocal` 什麼時候就夠。** 目錄裡的檔案全由自己的程式寫、沒有任何路徑能產生符號連結，檢查的又只是「這個名字能不能安全接在目錄後面」。這時 `IsLocal` 加 `Join` 比 `os.Root` 簡單。只要有使用者上傳、解壓縮別人給的壓縮檔、或別的程式也能寫這個目錄，就該用 `os.Root`。
- **檔名驗證仍然要做。** `os.Root` 不知道 `.state` 不該給人讀，也不限制長度、字元與副檔名。本集用副檔名白名單，白名單比「擋掉已知壞的」好，本章〈輸入驗證〉談過。
- **`os.DirFS` 擋不住符號連結。** 文件寫得很清楚：它只保證開檔從目錄開頭，不是 chroot 的替代品；要把目錄交給 `fs.FS` 的使用者，用 `Root.FS`。
- **程式這一層以外。** 檔案權限、服務的執行身分、唯讀掛載，是另外的防線；`os.Root` 不取代它們。
- **before 在什麼情況下是對的。** 目錄與檔名都由自己控制、不含使用者輸入時，直接 `os.ReadFile` 就好，不必檢查。

## 重點整理

- `Join` 加 `HasPrefix` 比的是字串：相同開頭的兄弟目錄與符號連結都能繞過，只測 `..` 會得到錯誤的信心。
- 先用 `filepath.IsLocal` 在字串上拒絕，再用 `os.Root` 在檔案系統這一層擋住符號連結；各擋各的，測試要斷言是哪一關。
- 目錄只有自己寫的檔案時 `IsLocal` 就夠；使用者能影響目錄內容時用 `os.Root`。
- `os.Root` 不管同一個目錄裡哪些檔案該給人讀，檔名驗證仍然要做；平台差異與限制要看文件。
