# `GOMEMLIMIT` 與容器記憶體上限

## 本集目標

判斷服務在有記憶體上限的容器裡該不該設 `GOMEMLIMIT`：`GOGC` 只看比例、不知道容器的上限；由 cgroup 的 `memory.max` 算出略低於上限的值並在啟動時設定，GC 才會在撞到上限之前收垃圾。也要知道它救不了什麼。

## 情境

一個服務跑在 `--memory=256m` 的容器裡，把一份資料集（快取）一直留在記憶體，每個請求另外配置一塊暫存緩衝區，用完就丟。上線後容器偶爾被 OOM 殺掉、重啟，heap 看起來並沒有洩漏。`GOGC`、`GOMEMLIMIT` 與 `debug.SetMemoryLimit` 的用法前作教過：〈[GC 概念](https://holmeslin.github.io/learning-go-from-zero/zh-TW/appendix2/c_gc.html)〉。這集談的是容器的上限該怎麼交給 runtime。

## 常見寫法

什麼都不設，用預設的 `GOGC=100`。下面用 160 MiB 的快取加 2000 個請求模擬這個服務，每個請求配置 1 MiB：

```go,ignore
{{#include ../../../examples/chapter8/03_gomemlimit_container/before/workload.go:run}}
{{#include ../../../examples/chapter8/03_gomemlimit_container/before/main.go:main}}
```

這是一支吃大量記憶體的示範，只放在 `main` 套件，CI 只編譯不執行。完整程式碼在 `examples/chapter8/03_gomemlimit_container/before/`。

## 問題在哪

用官方 image 在容器裡重現。編譯不加限制，執行才套上 `--memory`；`--memory-swap` 與 `--memory` 相同，關掉 swap。在 macOS（arm64）上用 OrbStack 執行，容器是 `golang:1.27-alpine`（Go 1.27.3、linux/arm64、cgroup v2），`--cpus=2` 所以容器內的 P 是 2 個：

```bash
docker run --rm -v "$PWD":/src:ro -v "$PWD/bin":/out -w /src -e CGO_ENABLED=0 golang:1.27-alpine \
  sh -c 'go build -o /out/before ./examples/chapter8/03_gomemlimit_container/before'
docker run --name ep03-before --memory=256m --memory-swap=256m --cpus=2 -v "$PWD/bin":/out:ro golang:1.27-alpine /out/before
echo "exit code: $?"
docker inspect --format '{{.State.OOMKilled}}' ep03-before
```

執行結果（某一次）：

```text
memory limit: 9223372036854775807
exit code: 137
OOMKilled: true
```

`memory limit` 是程式啟動時印出的 `debug.SetMemoryLimit(-1)`，`math.MaxInt64` 就是「沒有上限」，runtime 不知道有 256 MiB 這回事。`done` 沒有印出來，因為行程被 kernel 殺了：137 是 128 加上 `SIGKILL` 的 9，`OOMKilled` 是 Docker 記下的原因。`exit code` 與 `OOMKilled` 兩行是 `docker run` 結束後另外查的，`docker run` 本身沒有印別的。

原因在 `GOGC=100`：下次 GC 的目標是上次 GC 之後活著資料的兩倍。活著的有 160 MiB，目標超過 256 MiB，heap 在 GC 啟動前就把容器的上限用完了。這個比例只和自己的資料有關，容器有多大它不看。

沒有加 `--memory-swap` 時，Docker 預設連 swap 一起給（是 `--memory` 的兩倍）；在這台 OrbStack 上，同一支程式沒有被殺，跑完才結束（見 `repro.sh` 的 `noswap.txt`）。允許 swap 的環境看不到 OOM，但記憶體其實已經超過上限了；要重現就要關掉 swap。

## 較好的做法

啟動時讀容器自己的 cgroup 檔案 `memory.max`，算出該設的 `GOMEMLIMIT`。`Limit` 只做計算，檔案從參數的 `fs.FS` 讀，所以測試可以用 `fstest.MapFS` 提供假檔案：

```go,ignore
{{#include ../../../examples/chapter8/03_gomemlimit_container/after/memlimit/memlimit.go:limit}}
```

取上限的 90%，留下 10%：`GOMEMLIMIT` 管的是 runtime 自己管理的記憶體（heap、堆疊、runtime 的內部結構），不含執行檔的映射、C 程式配置的記憶體、`syscall.Mmap` 與 kernel 代替行程持有的部分，容器的上限卻是整個行程加總。10% 是起點，不是定律：用了 cgo 或 mmap 的服務要留更多，看實際的 RSS 與上限差多少再調。`memory.max` 的內容是 `max` 表示沒有上限，這不是錯誤，所以用 `ErrNoLimit` 讓呼叫端區分。

`Apply` 把結果套到全域設定。行程已經有 memory limit（例如部署時設了環境變數）就不覆蓋：

```go,ignore
{{#include ../../../examples/chapter8/03_gomemlimit_container/after/memlimit/memlimit.go:apply}}
```

`main` 在啟動時呼叫一次，沒有 cgroup、沒有上限或已經設過都維持原狀；負載與 before 逐字相同（兩邊的 `workload.go` 內容一樣，由 `repro.sh` 用 `cmp` 確認）：

```go,ignore
{{#include ../../../examples/chapter8/03_gomemlimit_container/after/main.go:main}}
```

同樣的容器參數，改跑 after：

執行結果（某一次）：

```text
memory limit: 241591911
done, cached MiB: 160
exit code: 0
```

上限是 `memory.max` 的 268435456 減去十分之一（整數除法）。GC 現在知道總量要壓在這個值以下，會比 before 更早、更常收掉暫存緩衝區，同一個負載就跑完了。

測試只斷言確定的事：`Limit` 的數字、`max`、檔案不存在與各種格式錯誤，`Apply` 有沒有改到全域設定。錯誤訊息的內容由 `Example` 鎖住：

```go,ignore
{{#include ../../../examples/chapter8/03_gomemlimit_container/after/memlimit/memlimit_test.go:limit}}
{{#include ../../../examples/chapter8/03_gomemlimit_container/after/memlimit/example_test.go:example}}
```

`Apply` 改的是整個行程的設定，所以測試用 `t.Cleanup` 還原，也不呼叫 `t.Parallel`：

```go,ignore
{{#include ../../../examples/chapter8/03_gomemlimit_container/after/memlimit/memlimit_test.go:apply}}
```

沒有任何斷言碰 GC 次數、heap 或 RSS，那些數字每台機器、每次都不同。

## 取捨

- **活著的資料超過上限，`GOMEMLIMIT` 救不了。** 它是軟上限：垃圾可以收，活著的資料收不掉。活著的部分逼近上限時，GC 為了壓住總量越跑越頻繁，應用程式分到的 CPU 越來越少，這就是 death spiral。runtime 有保護，會把 GC 占用的 CPU 限制在大約一半以內，所以程式不會被 GC 完全餓死，代價是總量超過上限，容器還是可能被殺。GC 實際有多頻繁，本章〈讀 GC trace〉會談；該改的是活著的資料（本章〈記憶體配置與 heap 成長〉會談）或加大上限。
- **`GOGC=off` 加 `GOMEMLIMIT`。** 這樣 GC 只在總量碰到上限時才啟動，不再依比例觸發，少跑 GC、多用記憶體。服務獨占容器、活著的資料遠小於上限而且穩定時合理。危險在活著的資料長到靠近上限：沒有比例的觸發，GC 就一直在上限邊緣打轉，前面的 death spiral 立刻出現；上限也不能設到比容器還大。
- **也可以只在部署設定裡設環境變數。** 同一支 before，容器加 `-e GOMEMLIMIT=230MiB`，也跑完了（見 `repro.sh` 的 `env_override.txt`），不用改程式。代價是數字寫死在部署檔，上限調整了要記得同步。`Apply` 讓已經存在的設定優先，兩種做法可以並存。`automemlimit` 這類第三方套件做的是同一件事，只是不用自己寫。
- **這份計算有前提。** 只讀 cgroup v2 的 `memory.max`；cgroup v1 的檔名不同，這裡不處理。`/sys/fs/cgroup` 要是容器自己的 cgroup 根才對，容器的預設是如此，直接跑在主機上的行程不是。
- **before 在什麼情況下是對的？** 大多數服務用預設值就好。活著的資料加上 `GOGC` 帶來的餘裕遠小於容器上限，或根本沒有上限，`GOGC` 的比例已經夠用，不需要多這一層；`GOGC` 本身什麼時候該調，本章〈什麼時候該調 `GOGC`，什麼時候別動〉談過。

## 重點整理

- `GOGC` 只看自己活著的資料，不知道容器的上限；活著的資料夠大時，預設值會讓 heap 在 GC 前就撞到上限，被 OOM 殺掉（exit code 137）。
- 由 cgroup 的 `memory.max` 算出略低於上限的 `GOMEMLIMIT`，留出 runtime 管不到的記憶體；`max` 表示沒有上限，不設。
- 計算邏輯讀 `fs.FS`，用 `fstest.MapFS` 測；套用到全域設定的測試要還原、不能平行，也不斷言 GC 或記憶體的數字。
- `GOMEMLIMIT` 是軟上限，救不了活著的資料；`GOGC=off` 加它只適合活著的資料遠小於上限的服務。
- 環境變數與程式碼設定效果相同，選一個放在最容易維護的地方。
