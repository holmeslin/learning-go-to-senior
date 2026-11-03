# 提醒服務的映像：golang 映像只負責建置，執行映像是 scratch。不進 CI，在 repo 根目錄執行：
#   docker build -f examples/chapter16/reminder/Dockerfile --build-arg VERSION=$(git describe --tags --always --dirty) -t reminder:$(git describe --tags --always --dirty) .
#
# 單一實例：資料庫是一個 sqlite 檔，同一時間只能有一個行程在服務它。
# 發布新版本時先停舊的、再起新的，兩個行程不重疊：
#   docker stop reminder && docker rm reminder
#   docker run -d --name reminder --stop-timeout 40 -v reminder-data:/data -e REMINDER_PROVIDER_URL=... reminder:<新版本>
# 停止的寬限期（--stop-timeout）要比服務的關機期限（REMINDER_SHUTDOWN_TIMEOUT，預設 30 秒）長：
# docker 預設 10 秒就送 SIGKILL，進行中的送出做不完。Dockerfile 沒有設定寬限期的指令，要在 docker run 或平台的設定裡給。
# GOMEMLIMIT 與 GOMAXPROCS 也不寫在這裡：容器的上限是部署時決定的。

# ANCHOR: dockerfile
FROM golang:1.27-alpine AS build
WORKDIR /src
# 先只複製 go.mod 與 go.sum：原始碼改了，這一層與下載相依的那一層仍然命中快取
COPY go.mod go.sum ./
RUN go mod download
COPY examples/chapter16/reminder ./examples/chapter16/reminder
# 建置脈絡沒有 .git（見 Dockerfile.dockerignore），執行檔裡沒有 vcs 資訊，版本只能由建置參數帶進來
ARG VERSION=dev
# modernc.org/sqlite 是純 Go，CGO_ENABLED=0 的靜態執行檔就夠了
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-X main.version=${VERSION}" -o /out/reminder ./examples/chapter16/reminder/cmd/reminder
# scratch 沒有 mkdir，資料目錄在這裡先建好，交給非 root 的使用者
RUN mkdir /data && chown 65532:65532 /data

FROM scratch
# 推播供應商是 HTTPS，scratch 沒有 CA 憑證
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /out/reminder /reminder
COPY --from=build --chown=65532:65532 /data /data
# scratch 沒有 /etc/passwd，所以用數字 UID:GID
USER 65532:65532
ENV REMINDER_DB_PATH=/data/reminder.db
VOLUME /data
# 內部 API 的預設位址 :8080；診斷端點預設只聽 127.0.0.1:6060，容器外連不到，這是刻意的，不 EXPOSE
EXPOSE 8080
# exec 形式：執行檔是 PID 1，直接收得到 SIGTERM
ENTRYPOINT ["/reminder"]
# ANCHOR_END: dockerfile
