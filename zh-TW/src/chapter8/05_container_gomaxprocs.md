# 容器內的 `GOMAXPROCS`

## 本集目標

Go 1.25 起，容器裡的 `GOMAXPROCS` 預設會依 cgroup 的 CPU 配額決定，也會定期更新；讀完這集，能判斷自己的服務是不是被舊時代的寫法關掉了這個機制，並把覆寫拿掉。

## 情境

訂單服務跑在容器裡，部署時的 CPU limit 是 4 顆。幾年前的 Go 在容器裡看到的是整台機器的 CPU 數，`GOMAXPROCS` 的預設值就是那個數字，遠大於容器實際分到的配額。當時的建議是把 `GOMAXPROCS` 設成 limit：在程式碼裡呼叫 `runtime.GOMAXPROCS`、在 Dockerfile 寫 `ENV GOMAXPROCS=4`，或 import `go.uber.org/automaxprocs`。後來 limit 改成 2 顆，這些設定還留在原地。`GOMAXPROCS` 限制的是什麼，本章〈[排程器（GMP）怎麼看](01_scheduler_gmp.md)〉談過。

## 常見寫法

在 `main` 開頭寫死：

```go,ignore
{{#include ../../../examples/chapter8/05_container_gomaxprocs/before/main.go:main}}
```

在舊版 Go 它是對的。完整程式碼在 `examples/chapter8/05_container_gomaxprocs/before/`。

## 問題在哪

`go doc runtime.GOMAXPROCS` 說得很清楚：預設值取邏輯 CPU 數、CPU 親和性遮罩與 cgroup 配額（配額除以週期，不是整數就進位）三者的最小值，並且定期更新；但設了環境變數 `GOMAXPROCS`，或呼叫 `runtime.GOMAXPROCS(n)`，自動調整與更新就都停了。另外 `go.mod` 的 `go` 版本在 1.24 以下時，`GODEBUG` 的 `containermaxprocs` 與 `updatemaxprocs` 預設是 0，同樣沒有這個機制。

下面的輸出來自 macOS（arm64）上的 OrbStack：容器是 Linux、cgroup v2、看得到 14 顆 CPU，image 是 `golang:1.27-alpine`（Go 1.27.3）。每一行 `GOMAXPROCS` 就是 P 的數量。指令都在 repo 外的副本目錄、用 `go run` 執行，省略了 `-v "$PWD":/src -w /src` 掛載旗標，也省略了結束碼：每一條都是 0。

執行結果（某一次）：

```text
$ docker run --rm --cpus=2 golang:1.27-alpine go run .
NumCPU=14 GOMAXPROCS=4
$ docker run --rm --cpus=2 -e GOMAXPROCS=8 golang:1.27-alpine go run ./cmd/probe
NumCPU=14 GOMAXPROCS=8
warning: GOMAXPROCS=8 is set in the environment: the runtime keeps it instead of following the cgroup CPU limit
$ docker run --rm --cpus=2 golang:1.27-alpine go run ./cmd/probe
NumCPU=14 GOMAXPROCS=14
warning: GODEBUG containermaxprocs=0 turns off the cgroup-aware default: use go 1.25 or later in go.mod and drop the setting
GODEBUG updatemaxprocs=0 turns off the cgroup-aware default: use go 1.25 or later in go.mod and drop the setting
```

三條指令的限制都是 `--cpus=2`：第一條跑 before，寫死的 4 蓋過了配額，也不會再跟著配額變；第二條跑 after 的程式，環境變數把值固定成 8；第三條也是 after，但 `go.mod` 的 `go` 行是 1.24，`GOMAXPROCS` 等於 `NumCPU`，完全沒看配額。三種做法的結果都不是 2，而且程式裡沒有任何一行告訴你原因。

## 較好的做法

拿掉覆寫，讓 runtime 自己決定。服務該做的只剩啟動時把結果印出來，並檢查有沒有東西關掉了自動調整。`Check` 只讀不改：它看環境變數 `GOMAXPROCS` 與 `GODEBUG`，也看執行檔編譯時由 `go` 版本決定的預設值（`debug.ReadBuildInfo` 的 `DefaultGODEBUG`）。執行檔裡讀不到 `go.mod`，但編譯進去的預設值讀得到：

```go,ignore
{{#include ../../../examples/chapter8/05_container_gomaxprocs/after/check.go:check}}
```

呼叫端只有幾行，印出 `NumCPU` 與 `GOMAXPROCS(0)`（零代表只讀），有問題就警告：

```go,ignore
{{#include ../../../examples/chapter8/05_container_gomaxprocs/after/cmd/probe/main.go:main}}
```

下面兩條在同一台機器、同一個 image 上執行，省略的旗標與結束碼和上面相同。

執行結果（某一次）：

```text
$ docker run --rm golang:1.27-alpine go run ./cmd/probe
NumCPU=14 GOMAXPROCS=14
$ docker run --rm --cpus=2 golang:1.27-alpine go run ./cmd/probe
NumCPU=14 GOMAXPROCS=2
```

沒有限制時等於 CPU 數，限制成 2 顆就是 2。測試只斷言確定的事，不看這台機器有幾顆 CPU。假的編譯資訊與環境讓每種情況都可以驗證：

```go,ignore
{{#include ../../../examples/chapter8/05_container_gomaxprocs/after/check_test.go:table}}
```

這個 module 的 `go.mod` 是 `go 1.27`，所以真正編譯出來的測試執行檔必須通過檢查，這等於把「`go` 版本不低於 1.25」鎖進 CI；程式只讀不改這件事，用哨兵值證明（改全域設定的測試用 `t.Cleanup` 還原，也不呼叫 `t.Parallel`）：

```go,ignore
{{#include ../../../examples/chapter8/05_container_gomaxprocs/after/check_test.go:build}}
{{#include ../../../examples/chapter8/05_container_gomaxprocs/after/check_test.go:readonly}}
```

測試用到的兩個輔助函式，`buildInfo` 與 `env`，在同一個檔案裡：

```go,ignore
{{#include ../../../examples/chapter8/05_container_gomaxprocs/after/check_test.go:helpers}}
```

警告的文字用 `Example` 鎖住：

```go,ignore
{{#include ../../../examples/chapter8/05_container_gomaxprocs/after/check_test.go:example}}
```

## 取捨

- **配額小於 1 顆，預設值仍是 2。** 依 `go doc`，只要邏輯 CPU 數與親和性遮罩不低於 2，預設就不會小於 2。`--cpus=0.5` 的實測也是 2；P 的數量降不到 1，配額小於 1 顆的服務要另外評估自己能不能接受。
- **CPU 親和性也算。** 預設值取親和性遮罩與配額的較小者。下面用 `--cpuset-cpus` 把容器綁在特定 CPU 上（同一台機器、同一個 image），`NumCPU` 也跟著遮罩縮小；第一條是上一點的 `--cpus=0.5`，放在一起對照：

  執行結果（某一次）：

  ```text
  $ docker run --rm --cpus=0.5 golang:1.27-alpine go run ./cmd/probe
  NumCPU=14 GOMAXPROCS=2
  $ docker run --rm --cpuset-cpus=0-3 golang:1.27-alpine go run ./cmd/probe
  NumCPU=4 GOMAXPROCS=4
  $ docker run --rm --cpuset-cpus=0 golang:1.27-alpine go run ./cmd/probe
  NumCPU=1 GOMAXPROCS=1
  ```

  綁在 1 顆時才會得到 1：低於 2 的例外就是這一種。
- **真的需要固定值的少數情況。** 例如依第 7 章〈[先量測，再最佳化](../chapter7/01_measure_before_optimizing.md)〉的流程量測之後，確定固定值比預設值好，或壓測時要在同一台機器上比較不同的 P 數。固定值寫在部署設定（環境變數）裡，不要寫在程式碼中，這樣改配額時改的是同一個地方；之後想回到預設，可以呼叫 `runtime.SetDefaultGOMAXPROCS`。
- **舊版 Go 的建議。** `automaxprocs` 這類套件是為 Go 1.24 以下補的洞。升級後先確認它做了什麼：只要它還是呼叫 `runtime.GOMAXPROCS` 設定固定值，就會關掉定期更新，這時該移除。
- **before 在舊版 Go 上是對的。** 如果服務還不能升級到 1.25，寫死或用套件仍是合理的暫時做法。
- **想在執行時觀察。** `runtime/metrics` 的 `/sched/gomaxprocs:threads` 就是目前的值，本章〈`runtime/metrics`〉會談。

## 重點整理

- Go 1.25 起，`GOMAXPROCS` 的預設值取 CPU 數、親和性、cgroup 配額的最小值，並定期更新；不要在程式碼、Dockerfile 或第三方套件裡再蓋一次。
- 設環境變數、呼叫 `runtime.GOMAXPROCS(n)`、`go.mod` 的 `go` 版本在 1.24 以下，三種都會關掉自動調整。
- 啟動時印出 `GOMAXPROCS(0)` 並檢查這三項，比猜更快；測試只斷言檢查邏輯，不斷言這台機器的 CPU 數。
- 配額小於 1 顆時預設仍是 2；CPU 親和性也會限制預設值。
- 真的需要固定值時放在部署設定裡，並知道 `runtime.SetDefaultGOMAXPROCS` 可以回到預設。
