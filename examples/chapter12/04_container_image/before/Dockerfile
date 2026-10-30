# 單階段：建置用的映像就是執行用的映像。在 repo 根目錄執行：
#   docker build -f examples/chapter12/04_container_image/before/Dockerfile -t ch12-ep04:before .

# ANCHOR: dockerfile
FROM golang:1.27-alpine
WORKDIR /src
COPY . .
RUN go build -o /usr/local/bin/app ./examples/chapter12/04_container_image/before
EXPOSE 8080
CMD ["app"]
# ANCHOR_END: dockerfile
