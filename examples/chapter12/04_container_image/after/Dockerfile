# multi-stage：golang 映像只負責建置，執行映像是 scratch。在 repo 根目錄執行：
#   docker build -f examples/chapter12/04_container_image/after/Dockerfile -t ch12-ep04:after .

# ANCHOR: dockerfile
FROM golang:1.27-alpine AS build
WORKDIR /src
# 先只複製 go.mod 與 go.sum：原始碼改了，這一層與下載相依的那一層仍然命中快取
COPY go.mod go.sum ./
RUN go mod download
COPY examples/chapter12/04_container_image/after ./examples/chapter12/04_container_image/after
RUN CGO_ENABLED=0 go build -trimpath -o /out/app ./examples/chapter12/04_container_image/after

FROM scratch
COPY --from=build /out/app /app
# scratch 沒有 /etc/passwd，所以用數字 UID:GID
USER 65532:65532
EXPOSE 8080
# exec 形式：執行檔是 PID 1，直接收得到 SIGTERM
ENTRYPOINT ["/app"]
# ANCHOR_END: dockerfile
