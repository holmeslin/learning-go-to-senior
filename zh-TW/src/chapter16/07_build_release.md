# 建置與發布

## 本集目標

提醒服務送上線之前，判斷建置與發布要先確定哪幾件事：版本怎麼進到執行檔、停舊與起新之間的寬限期怎麼和服務的關機期限對上，以及版本、送出的開關與回復的順序怎麼配合。

## 情境

提醒服務要發第一個版本，`v1.0.0`（版本號是教材虛構的）。依本章〈[設計文件](01_design_doc.md)〉，它只跑一個實例，發布時先停舊、再起新，新舊行程不重疊；送出的開關是環境變數，預設關閉。建置與設定的做法，第 8 章與第 12 章都談過，這集看的是它們湊在一起時，哪些地方沒有人替你檢查。Dockerfile 與 `docker` 指令不進 CI（同第 12 章〈[容器映像（multi-stage）](../chapter12/04_container_image.md)〉），本書也沒有實際建過這個映像，所以不貼 `docker build` 與 `docker run` 的輸出；貼出的輸出來自 Dockerfile 裡 `go build` 那一行的等效建置，做法在下面說明。

## 映像裡放什麼

形狀照第 12 章〈[容器映像（multi-stage）](../chapter12/04_container_image.md)〉：`golang` 映像建置，`scratch` 執行。下面是這個服務的 Dockerfile（檔頭另有幾行註解，寫的是後面〈發布〉的指令）：

```dockerfile
{{#include ../../../examples/chapter16/reminder/Dockerfile:dockerfile}}
```

和第 12 章不同的幾處，都來自這個服務的條件：

- **`ARG VERSION=dev` 與 `-X main.version`。** 建置脈絡沒有 `.git`（`Dockerfile.dockerignore` 排除了它），版本只能由建置參數帶進來；沒帶參數也建得成，版本是 `dev`。
- **`CGO_ENABLED=0`。** `modernc.org/sqlite` 是純 Go，靜態執行檔就夠，不必帶 libc。
- **`/data` 與數字 UID。** 資料庫是一個 sqlite 檔，放在掛載的 volume；`scratch` 沒有 `mkdir`，目錄在建置階段建好，用 `COPY --chown` 交給 65532。
- **只 `EXPOSE 8080`。** 那是內部 API；診斷端點只聽 `127.0.0.1:6060`，刻意不開出去（見〈取捨〉）。

`GOMEMLIMIT` 與 `GOMAXPROCS` 不寫在 Dockerfile：容器的上限是部署時才決定的。`GOMAXPROCS` 什麼都不設，預設值會跟著 cgroup 的 CPU 配額（第 8 章〈[容器內的 `GOMAXPROCS`](../chapter8/05_container_gomaxprocs.md)〉）；`GOMEMLIMIT` 在部署時用環境變數設成容器上限的九成左右（第 8 章〈[`GOMEMLIMIT` 與容器記憶體上限](../chapter8/03_gomemlimit_container.md)〉），下面的部署腳本照做。`GOMEMLIMIT` 是軟上限，活著的資料它救不了；防止失控的是本章〈[並行](05_concurrency.md)〉的逾時與並行上限，`--memory` 讓失控時被終止的只有提醒服務的容器，不再是報名服務。

設定的做法是第 12 章〈[設定與環境變數](../chapter12/08_config_env.md)〉的：只有環境變數，空字串視為沒設，每個有問題的變數一次報出來，錯誤不帶供應商網址。`Load` 的後半是這個服務自己的規則：

```go,ignore
{{#include ../../../examples/chapter16/reminder/internal/config/config.go:load}}
```

並行上限至少 1；關機期限不小於最壞的一則送出加餘裕，算式與理由本章〈[並行](05_concurrency.md)〉談過；診斷位址的主機只能是 loopback，否則拒絕啟動。這些都是行程自己看得到的條件，平台給的寬限期它看不到，見〈取捨〉。

## 版本：從哪裡來，在哪裡看

版本的規則是第 12 章〈[`go build` 與用 `-ldflags` 注入版本](../chapter12/01_build_version_info.md)〉的：commit、時間與有沒有未提交的修改，`go build` 已經記在執行檔裡，用 `debug.ReadBuildInfo` 讀。`main` 只多一個不匯出的字串變數，留給 `-ldflags -X` 注入；`buildinfo.From` 把兩個來源整理成同一個 `Info`，注入的版本號優先，沒注入就用主模組的版本，commit 與時間一律取自建置資訊。它收 `*debug.BuildInfo`，不自己呼叫 `ReadBuildInfo`，測試才能傳假的：

```go,ignore
{{#include ../../../examples/chapter16/reminder/cmd/reminder/main.go:version}}
```

```go,ignore
{{#include ../../../examples/chapter16/reminder/internal/buildinfo/buildinfo.go:info}}
```

`go test` 的測試執行檔沒有 vcs 資訊，所以 `From` 的測試用假的 `BuildInfo`（`TestFrom`：注入優先、沒注入、`go run`、什麼都沒有）。啟動日誌與診斷端點的 `/version` 用同一份 `Info`，一個在 bubble 外、用真的 socket 的測試同時斷言兩處：

```go,ignore
{{#include ../../../examples/chapter16/reminder/service_version_test.go:version}}
```

映像裡實際看得到什麼？把 Dockerfile 會 `COPY` 的檔案放進一個沒有 `.git` 的目錄，用 `RUN` 那一行的旗標建置，再看 `go version -m`。這是本機（macOS）的建置，不是映像裡的；映像裡 `GOOS` 會是 `linux`：

```bash
export GOTOOLCHAIN=go1.27.0
mkdir -p ctx/examples/chapter16 && cp go.mod go.sum ctx/ && cp -R examples/chapter16/reminder ctx/examples/chapter16/
cd ctx
CGO_ENABLED=0 go build -trimpath -ldflags "-X main.version=v1.0.0" -o ../reminder ./examples/chapter16/reminder/cmd/reminder
go version -m ../reminder | grep -v -e $'^\tdep'
```

執行結果（某一次）：平台是 `Darwin arm64`（`uname -sm`），`go version` 是 `go1.27.0 darwin/arm64`。`dep` 開頭的相依套件行省略了，行尾的空白也去掉了：

```text
../reminder: go1.27.0
	path	github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/cmd/reminder
	mod	github.com/holmeslin/learning-go-to-senior	(devel)
	build	-buildmode=exe
	build	-compiler=gc
	build	-trimpath=true
	build	CGO_ENABLED=0
	build	GOARCH=arm64
	build	GOOS=darwin
	build	GOARM64=v8.0
```

接著用 `cmd/seed` 灌一場 3 位報名者的活動，讓檢視有東西，再以預設的關閉開關啟動，這是上線的第 1 步。供應商網址必填，開關關閉時不會有請求送出；實際重現時先起 `cmd/slowprovider` 並等它的啟動日誌，下面省略。日誌的 `time` 欄位每次不同，也省略了：

```bash
../seed -db ../reminder.db -event 1 -registrants 3
REMINDER_PROVIDER_URL=http://127.0.0.1:9707 REMINDER_DB_PATH=../reminder.db \
REMINDER_API_ADDR=127.0.0.1:8787 REMINDER_DIAG_ADDR=127.0.0.1:6767 ../reminder 2> ../service.log &
curl -s http://127.0.0.1:6767/version
grep 'reminder service started' ../service.log
```

執行結果（某一次）：

```text
{"version":"v1.0.0","modified":false,"go_version":"go1.27.0"}
{"level":"INFO","msg":"reminder service started","send_enabled":false,"version":"v1.0.0 (no vcs info) go1.27.0"}
```

三件事一起看：`mod` 是 `(devel)` 而且沒有 `vcs.*`，所以 `/version` 沒有 `revision` 與 `time`，日誌是 `(no vcs info)`；沒有 `-ldflags` 那一行，是 `-trimpath` 的緣故，同樣的建置拿掉它，`go version -m` 才多出 `build -ldflags="-X main.version=v1.0.0"`；`v1.0.0` 只在 `/version` 與日誌看得到。

## 發布：先停舊、再起新

發布流程只寫在這裡，本書的 repo 不建立 tag 或 release（版本號與 tag 的規則見第 12 章〈[發布流程與版本號](../chapter12/07_release_and_versioning.md)〉）。要互相配合的數字在部署腳本裡只寫一次，另一個由它算出來：

```bash
# deploy.sh（示意，書附範例沒有這個檔案）
: "${SEND_ENABLED:?要明寫 true 或 false}"          # 開關每次都要明寫，沒寫就在動到舊容器之前停下來
VERSION=${VERSION:-$(git describe --tags --always --dirty)}   # 回復時由外面指定，例如 VERSION=v1.0.0
[[ $VERSION =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo "不是乾淨的 tag：$VERSION" >&2; exit 1; }
SHUTDOWN=30                                       # REMINDER_SHUTDOWN_TIMEOUT 的秒數，只在這裡寫一次
docker image inspect reminder:$VERSION >/dev/null 2>&1 ||   # 映像已經有了就不重建：回復用的是當初建好的那一個
  docker build -f examples/chapter16/reminder/Dockerfile --build-arg VERSION=$VERSION -t reminder:$VERSION .
docker stop reminder && docker rm reminder        # 先停舊：等舊行程自己結束，最久 --stop-timeout
docker run -d --name reminder -v reminder-data:/data \
  --memory=256m -e GOMEMLIMIT=230MiB \
  --stop-timeout $((SHUTDOWN + 10)) -e REMINDER_SHUTDOWN_TIMEOUT=${SHUTDOWN}s \
  -e REMINDER_SEND_ENABLED=$SEND_ENABLED -e REMINDER_PROVIDER_URL=... reminder:$VERSION
docker logs reminder 2>&1 | grep 'reminder service started'
```

最後一行是確認：啟動日誌的 `version` 要等於要發的版本，`send_enabled` 要等於預期。`/version` 在容器外問不到（見〈取捨〉），`docker logs` 不需要網路。版本再對回 commit：`git rev-list -n 1 v1.0.0`；tag 不移動、不重用。

版本與開關各是一次「停舊、再起新」，開關是環境變數，啟動時讀一次，所以改開關也要重啟。兩件事分開走，順序照設計文件：

| 步驟 | 動作 | 日誌的 `version` | 日誌的 `send_enabled` |
| --- | --- | --- | --- |
| 1 | 新服務以關閉的開關部署：`SEND_ENABLED=false ./deploy.sh` | 新版本 | `false` |
| 2 | 關掉報名服務裡 worker 的開關 | 不變 | 不變 |
| 3 | 打開新服務的開關：同一個版本，`SEND_ENABLED=true` 再跑一次 | 不變 | `true` |
| 回復開關 | 先 `SEND_ENABLED=false` 重啟新服務，再打開報名服務的 worker | 不變 | `false` |
| 回復版本 | `VERSION=<前一個版本> SEND_ENABLED=<目前的值> ./deploy.sh` | 前一個版本 | 不變 |

## 取捨

這集的取捨都來自前面各章的判斷湊在一起：

- **映像沒有 `.git`，版本號成了沒人檢查的字串。** 第 12 章要 `go build` 自己記 commit，`-X` 只留給建置系統才知道的值；同一章的 `.dockerignore` 又把 `.git` 擋在建置脈絡外。兩個都照做，就是上面的輸出：沒有 `vcs.*`，`v1.0.0` 只是一個建置參數。`ARG VERSION=dev` 讓忘了帶參數的建置也建得成，看到 `dev` 至少知道它不是發布出來的，看到 `v1.0.0` 卻不一定是真的。所以檢查放在流程：腳本只接受乾淨的 tag（`git describe --dirty` 帶 `-dirty` 或 `-N-g…` 就停），發布後比對日誌的版本。另一條路是把 `.git` 帶進建置脈絡拿回 `vcs.*`，連沒碰到程式碼的 commit 也讓編譯的那一層重做。靠流程比靠編譯器弱，單一實例、人手發布的規模用得起。
- **從容器外問版本，只剩 `docker logs`。** 為了可重現的建置（第 12 章〈[可重現的建置](../chapter12/02_reproducible_builds.md)〉）用了 `-trimpath`，`go version -m` 看不到 `-ldflags`，版本只能從 `/version` 與日誌讀。診斷端點只聽 `127.0.0.1:6060`，因為有 pprof，不能對外（第 9 章〈[`/debug/pprof` 與診斷端點](../chapter9/06_debug_pprof.md)〉），`Load` 拒絕非 loopback 的位址；容器的 loopback 屬於容器自己的網路命名空間，`-p` 轉發不到，`scratch` 裡又沒有 `curl` 與 shell，所以 `/version` 在容器外問不到。要打端點，得另起一個帶 `curl` 的映像、共用這個容器的網路（`docker run --rm --network container:reminder <帶 curl 的映像> curl -s http://127.0.0.1:6060/version`），或用平台的除錯容器。代價是診斷多一步，平台的健康檢查也探測不到 `/readyz`。把位址改成 `0.0.0.0` 能省掉這一步，pprof 就對整個容器網路開放，`Load` 的檢查也得跟著拿掉。
- **寬限期寫在兩個地方，沒有東西檢查它們一致。** `Load` 保證 `REMINDER_SHUTDOWN_TIMEOUT` 不小於最壞的一則送出加餘裕（預設下限 25 秒），平台給的寬限期它看不到：`docker stop` 預設 10 秒就送 `SIGKILL`，比最壞的 15 秒還短，進行中的送出被切斷，紀錄停在「寄送中」，1 小時後記為失敗，就是本章〈[並行](05_concurrency.md)〉的 before。Dockerfile 沒有設定寬限期的指令，只能在 `docker run --stop-timeout` 或平台設定裡給，所以腳本只寫一次 `SHUTDOWN`，`--stop-timeout` 取它再加 10 秒。代價是腳本成了正確性的一部分，直接手打 `docker run` 就繞過它；停舊最久的 40 秒也算進發布的空窗，相對預定時間後的 1 小時用得起（本章〈[設計文件](01_design_doc.md)〉）。
- **開關預設關閉，只對第一次部署是對的。** 預設的 `false` 讓步驟 1 安全；但之後每一次發布，漏寫開關就是悄悄把送出關掉，連回復版本也一樣，關著超過 1 小時，到期的提醒打開後記為失敗（本章〈上線檢查表〉會談）。所以腳本不給預設值，沒寫就停下來，動到舊容器之前。
- **回復版本，是舊的執行檔面對新的資料庫。** 單一實例先停舊、再起新，沒有新舊同時跑，第 4 章〈[migration](../chapter4/06_migration.md)〉為滾動更新訂的「先擴充，再收斂」看起來用不上。但回復時舊版本要在新版本 migrate 過的資料庫上啟動：`Migrate` 只套用自己有的檔案，資料庫多出來的版本紀錄它不管，所以舊版本不會拒絕啟動，能不能跑，取決於新的 migration 有沒有動到舊版本讀寫的欄位。`store/schema` 目前只有 `001_deliveries.sql`，沒有這個問題；之後的 migration 照同一條規則寫，理由從「新舊同時跑」換成「回復得了」。回復用的是當初建好的映像，不是重建：基底映像的標籤會移動（第 12 章〈[容器映像（multi-stage）](../chapter12/04_container_image.md)〉），重建出來的不一定是原來那一份。

這些都不做，也有對的時候：只在一台機器上跑、由維護的人在那台機器上建置的工具，不需要映像、`VERSION` 參數與部署腳本，`go build` 記下的 commit 與 `go version -m` 就夠。這個服務有映像、建置脈絡沒有 `.git`，停舊與起新之間還有一個要做對的順序，所以才需要這些。

## 重點整理

- 容器的建置脈絡沒有 `.git`，執行檔裡就沒有 `vcs.*`，版本只剩 `-X` 注入的字串；發布腳本只接受乾淨的 tag，發布後拿日誌的版本對照，再用 tag 找回 commit。
- `-trimpath` 讓 `go version -m` 看不到 `-ldflags`；診斷端點只聽 loopback、`scratch` 沒有 `curl`，從容器外問版本只剩 `docker logs`。
- `--stop-timeout` 要比 `REMINDER_SHUTDOWN_TIMEOUT` 長，`Load` 看不到平台的寬限期，兩個數字由腳本從同一處算出。
- 版本與開關各是一次停舊再起新：先以關閉的開關部署並確認版本，再依序關舊 worker、開新服務；開關每次明寫，回復用當初建好的映像。
