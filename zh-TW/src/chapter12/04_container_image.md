# 容器映像（multi-stage）

## 本集目標

判斷服務的映像該怎麼建：用 `golang` 映像建置、只把執行檔放進最小的執行映像、不用 root 執行；也知道最小的映像拿掉了什麼，哪些問題要自己補。

## 情境

一個小的 HTTP 服務（`GET /livez` 與 `GET /hello`），要打包成容器部署。Go 編譯出來是單一執行檔，執行時不需要工具鏈。`go.mod` 前作教過：〈[`go mod init` 與 `go.mod`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/01_go_mod.html)〉。這集的指令都在 repo 根目錄執行。

下面的輸出來自 macOS（arm64）上的 OrbStack（docker 29.8.2）：容器是 linux/arm64、cgroup v2，建置用的 image 是 `golang:1.27-alpine`（Go 1.27.3）。

## 常見寫法

單階段：建置用的映像就是執行用的映像，`COPY . .` 把整個目錄帶進去：

```dockerfile
{{#include ../../../examples/chapter12/04_container_image/before/Dockerfile:3:}}
```

寫法直白，也真的跑得起來。完整程式碼在 `examples/chapter12/04_container_image/before/`。

## 問題在哪

建出映像，進去看裡面有什麼（`head -4` 只截前四行，省略其餘檔案）：

```bash
docker build -f examples/chapter12/04_container_image/before/Dockerfile -t ch12-ep04:before .
docker run --rm ch12-ep04:before sh -c 'id -u; go version; ls /src | head -4'
```

執行結果（某一次）：

```text
0
go version go1.27.3 linux/arm64
AGENTS.md
CLAUDE.md
GLOSSARY.md
LICENSE
```

三件事一次看到：使用者是 UID 0（root），服務被攻破時攻擊者在容器裡就是 root；映像帶著完整的 Go 工具鏈，執行時用不到，卻成了額外的攻擊面與要更新的東西；原始碼整包都在 `/src`。`COPY . .` 複製的是整個建置情境（build context）：目錄裡有 `.env` 就一起進了映像，和第 11 章〈[秘密管理](../chapter11/05_secret_management.md)〉的做法正好相反：那裡的秘密在執行時才從環境變數或掛載的檔案讀。另外，任何一個檔案改動都會讓這一層與後面的 `go build` 重做；有相依的服務每次建置都會重新下載模組。

## 較好的做法

multi-stage：第一個階段建置，第二個階段只放執行檔。

```dockerfile
{{#include ../../../examples/chapter12/04_container_image/after/Dockerfile:3:}}
```

- **模組快取層。** 先只複製 `go.mod` 與 `go.sum` 再 `go mod download`；原始碼改了，這兩層命中快取。本書的 `go.mod` 是整個 repo 共用的，所以這層會下載其他章的相依；你的服務只會下載自己的。
- **`-trimpath` 與 `CGO_ENABLED=0`。** 前者讓執行檔不帶建置機器的路徑，本章〈可重現的建置〉談過；後者建出不依賴 libc 的靜態執行檔，本章〈交叉編譯〉談過。`scratch` 裡沒有 libc；明寫出來，換到帶 C 編譯器的基底映像時，建出的仍是能放進 `scratch` 的靜態執行檔。
- **`scratch` 加數字 UID。** `scratch` 是空的映像，只有 `COPY` 進去的東西；沒有 `/etc/passwd`，所以 `USER` 寫 `65532:65532`。
- **exec 形式的 `ENTRYPOINT`。** 執行檔直接是 PID 1，`docker stop` 送的 `SIGTERM` 它收得到。

在 `after/main.go` 結尾加一行註解後重建，只看建置階段的步驟（`CACHED` 表示命中快取，後面沒有的是重新執行的步驟，第一行是本機已有的基底映像）：

```bash
docker build --progress=plain -f examples/chapter12/04_container_image/after/Dockerfile -t ch12-ep04:after . 2>&1 | grep -E '^#[0-9]+ (\[build|CACHED)'
```

執行結果（某一次）：

```text
#3 [build 1/6] FROM docker.io/library/golang:1.27-alpine
#5 [build 2/6] WORKDIR /src
#5 CACHED
#6 [build 3/6] COPY go.mod go.sum ./
#6 CACHED
#7 [build 4/6] RUN go mod download
#7 CACHED
#8 [build 5/6] COPY examples/chapter12/04_container_image/after ./examples/chapter12/04_container_image/after
#9 [build 6/6] RUN CGO_ENABLED=0 go build -trimpath -o /out/app ./examples/chapter12/04_container_image/after
```

兩個映像的大小，在同一台機器上比較（只看相對關係，數字依平台與版本而不同）：

```bash
docker images ch12-ep04 --format 'table {{.Repository}}:{{.Tag}}\t{{.Size}}'
```

執行結果（某一次）：

```text
REPOSITORY:TAG     SIZE
ch12-ep04:after    9.18MB
ch12-ep04:before   344MB
```

這一次單階段的映像大約是 multi-stage 的三十七倍。被打包的服務在 `service` 套件，測試只測行為，映像是不是對的由上面的指令確認，測試不呼叫 `docker`：

```go,ignore
{{#include ../../../examples/chapter12/04_container_image/after/service/example_test.go:example}}
```

`main` 收 `SIGTERM` 後先停止接收新連線再結束；等什麼、等多久，第 10 章〈[關機時該等什麼、等多久（graceful shutdown）](../chapter10/06_graceful_shutdown.md)〉談過，這裡只是最小版本：

```go,ignore
{{#include ../../../examples/chapter12/04_container_image/after/main.go:main}}
```

執行、探測、停止（`-p` 把容器的 8080 對應到本機的 18080，`EXPOSE` 本身不會開放任何連接埠）。省略的輸出：`docker run -d` 印出的容器 ID、`docker stop` 與 `docker rm` 印出的容器名稱，以及日誌每行開頭的日期與時間：

```bash
docker run -d --name ch12-ep04 -p 18080:8080 ch12-ep04:after
curl -s localhost:18080/livez
docker stop ch12-ep04
docker logs ch12-ep04
docker inspect --format '{{.State.ExitCode}}' ch12-ep04
docker rm ch12-ep04
```

執行結果（某一次）：

```text
ok
INFO listening addr=:8080 uid=65532
INFO shutting down
0
```

`ok` 是 `curl` 的輸出。服務以 UID 65532 執行，`docker stop` 之後走完關機流程，結束碼 0。

## 取捨

- **`scratch` 缺的東西。** 沒有 CA 憑證，對外的 HTTPS 會以 `x509: certificate signed by unknown authority` 失敗；沒有時區資料，`time.LoadLocation("Asia/Taipei")` 回 `unknown time zone`（兩者都在 `scratch` 裡實測過）。補法：`COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/`，程式 `import _ "time/tzdata"`（文件說執行檔約增加 450 KB）。懶得逐項補，就用 distroless 的 `static` 映像：實測 `gcr.io/distroless/static-debian12:nonroot` 內有 CA 憑證、時區資料與 `/etc/passwd`，預設使用者是 65532。
- **除錯工具。** `scratch` 沒有 shell：`docker run --rm --entrypoint sh ch12-ep04:after` 的第一行錯誤是 `exec: "sh": executable file not found in $PATH`（省略最後一行 `docker run --help` 的提示）。要進去看，得靠平台的除錯容器（例如 `kubectl debug`）、另開帶工具的映像共用行程，或在除錯時換用帶 shell 的基底。把整套工具塞進正式映像，等於回到 before。
- **標籤與 digest。** `golang:1.27-alpine` 是會移動的標籤：Go 的小版本或 Alpine 更新後，同一份 Dockerfile 建出的內容就不同。要固定就寫成 `FROM golang:1.27-alpine@sha256:…`，代價是要有人（或 Dependabot、Renovate 這類工具）定期更新。自己的映像也一樣：部署用 commit 或版本做標籤，不用 `latest`，要完全確定時用 digest 引用。
- **`.dockerignore`。** `after/` 旁的 `Dockerfile.dockerignore`（BuildKit 會讀與 Dockerfile 同目錄、名為 `<Dockerfile>.dockerignore` 的檔案，實測有效；沒有這個檔案就讀根目錄的 `.dockerignore`）：

```text
{{#include ../../../examples/chapter12/04_container_image/after/Dockerfile.dockerignore}}
```

after 沒有 `COPY . .`，所以不靠它也不會帶進這些檔案；它防的是日後有人改成 `COPY . .`，也讓建置情境小一點。

- **容器裡的 runtime 設定。** Dockerfile 不要寫 `ENV GOMAXPROCS=…`：第 8 章〈[容器內的 `GOMAXPROCS`](../chapter8/05_container_gomaxprocs.md)〉說明預設值會跟著 cgroup 的配額，寫死反而關掉；記憶體上限怎麼交給 runtime，見第 8 章〈[`GOMEMLIMIT` 與容器記憶體上限](../chapter8/03_gomemlimit_container.md)〉。容器的上限是部署時決定的（`docker run --memory` 或平台），寫進映像會和部署脫節。
- **健康檢查與秘密。** `scratch` 裡沒有 `curl`，Dockerfile 的 `HEALTHCHECK` 沒有東西可執行；多數平台由外面探測 `/livez`，端點怎麼設計見第 9 章〈[健康檢查](../chapter9/07_health_check.md)〉。`ENV`、`ARG`、`COPY .env` 的內容都留在映像裡：實測 `docker history` 看得到 `ENV` 與 `ARG` 的值，`docker inspect` 看得到 `ENV`，映像內的檔案一取出就是明文；秘密在執行時才交給容器。
- **before 什麼時候是對的？** 本機開發、一次性的工具、需要在容器裡跑 `go test` 或 `go run`，而且映像不會離開你的機器時，單階段加完整工具鏈很方便。用了 cgo 就不能用 `scratch` 這種做法，要選帶 libc 的執行映像。

## 重點整理

- 建置與執行分兩個階段：工具鏈與原始碼留在建置階段，執行映像只放執行檔；單階段映像帶著工具鏈、原始碼與 root，在這次是 multi-stage 的三十七倍大。
- `go.mod`、`go.sum` 先複製並下載，原始碼改動時模組層命中快取；`CGO_ENABLED=0` 加 `-trimpath` 建出可放進 `scratch` 的靜態執行檔。
- 用數字 UID 不以 root 執行；`ENTRYPOINT` 用 exec 形式，執行檔當 PID 1 才收得到 `SIGTERM`。
- `scratch` 缺 CA 憑證、時區資料與 shell，要自己補或改用 distroless；標籤會移動，要固定用 digest；秘密不進映像。
