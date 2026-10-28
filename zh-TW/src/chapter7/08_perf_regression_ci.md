# 把效能回歸擋在 CI 之外

## 本集目標

決定 CI 該怎麼擋效能回歸：配置次數用 `AllocsPerRun` 的測試每次確定地擋，benchmark 每次只跑一遍確保不會爛掉，時間的比較交給固定機器上的 `benchstat`。

## 情境

服務的中介層把請求路徑整理成指標標籤，`/users/48213/orders/90177/items/3` 變成 `/users/:id/orders/:id/items/:id`，每個請求都會走一次。團隊已經把 `Template` 改成只配置一次，希望之後沒有人不小心改慢。benchmark 的寫法前作教過：〈[benchmark 與 `b.Loop`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/12_benchmark.html)〉。這集談的是怎麼讓 CI 守住它。

## 常見寫法

CI 跑一次 benchmark，ns/op 超過門檻就失敗。門檻取本機某次結果（約 66 ns）的 1.5 倍，湊個整數 100：

```yaml
- run: go test -run='^$' -bench=Template ./... | tee bench.txt
- run: awk '/^BenchmarkTemplate/ { if ($3 > 100) exit 1 }' bench.txt
```

```go,ignore
{{#include ../../../examples/chapter7/08_perf_regression_ci/before/route_test.go:bench}}
```

完整程式碼在 `examples/chapter7/08_perf_regression_ci/before/`。

## 問題在哪

一次 benchmark 只是一個樣本。共用 runner 上同時還有別的 job 在搶 CPU，同一份程式的 ns/op 每次都不一樣，單一門檻不是太緊就是太鬆。下面在背景放跟 CPU 數一樣多的忙碌行程，模擬有鄰居的 runner，再跑同一個 benchmark 十次：

```bash
for _ in $(seq "$(getconf _NPROCESSORS_ONLN)"); do (while :; do :; done) & done
go test -run='^$' -bench=. -benchmem -count=10 ./examples/chapter7/08_perf_regression_ci/before
kill $(jobs -p)
```

執行結果（某一次）：

```text
goos: darwin
goarch: arm64
cpu: Apple M4 Max
BenchmarkTemplate-14    	12718030	        95.58 ns/op	      48 B/op	       1 allocs/op
BenchmarkTemplate-14    	10450203	       135.9 ns/op	      48 B/op	       1 allocs/op
BenchmarkTemplate-14    	 8419665	       144.9 ns/op	      48 B/op	       1 allocs/op
BenchmarkTemplate-14    	 8727076	       121.0 ns/op	      48 B/op	       1 allocs/op
BenchmarkTemplate-14    	 9303310	       135.0 ns/op	      48 B/op	       1 allocs/op
BenchmarkTemplate-14    	 9483279	       114.3 ns/op	      48 B/op	       1 allocs/op
BenchmarkTemplate-14    	15475882	        99.65 ns/op	      48 B/op	       1 allocs/op
BenchmarkTemplate-14    	12545958	       114.2 ns/op	      48 B/op	       1 allocs/op
BenchmarkTemplate-14    	 8277645	       124.9 ns/op	      48 B/op	       1 allocs/op
BenchmarkTemplate-14    	14447038	       110.5 ns/op	      48 B/op	       1 allocs/op
```

輸出省略了 `pkg:` 行、`PASS` 與 `ok` 行。程式沒有變，配置次數每次都是 1，只有時間在跳，而且多數次超過 100 ns 的門檻：CI 變紅，但沒有任何一行程式有問題。把沒有鄰居時的十次結果存成 `quiet.txt`、這十次存成 `loaded.txt`，用 `benchstat` 比較（用法本章〈`b.Loop` 與 `benchstat`：怎麼比較前後結果才可信〉談過）：

執行結果（某一次）：

```text
goos: darwin
goarch: arm64
cpu: Apple M4 Max
            │  quiet.txt   │              loaded.txt               │
            │    sec/op    │    sec/op      vs base                │
Template-14   66.38n ± 16%   117.65n ± 16%  +77.25% (p=0.000 n=10)
```

輸出省略了 `pkg:` 行，以及 `B/op` 與 `allocs/op` 兩張表，兩張表都是 `~`，沒有差異。差距大到 `benchstat` 判為顯著，而造成它的只是機器的忙碌程度。門檻要放到不誤報，就得高過這個雜訊，這時小幅的退步會從門檻底下溜過去；紅燈時好時壞，團隊就會開始重跑到過，最後把這一步關掉。這和第 5 章〈[不穩定（flaky）測試](../chapter5/10_flaky_tests.md)〉是同一個問題。

## 較好的做法

依成本分三層，每層只擋它量得準的東西。

**第一層，每次 CI：用 `AllocsPerRun` 擋配置次數。** 配置次數是確定的，與鄰居無關，`-race` 下也成立（`AllocsPerRun` 本章〈常見熱點：配置、拷貝、鎖競爭〉談過）：

```go,ignore
{{#include ../../../examples/chapter7/08_perf_regression_ci/after/route_test.go:allocs}}
```

改寫不能改變行為，所以同一個測試檔有 `TestTemplateMatchesReference`：拿下面這個改寫前的寫法當對照，對一批路徑比對結果：

```go,ignore
{{#include ../../../examples/chapter7/08_perf_regression_ci/after/route_test.go:reference}}
```

門檻擋不擋得住，要實際看。把 after 的 `Template` 換成上面的寫法（暫時換上，不進 repo；輸出省略了套件那一行的路徑前綴與耗時），門檻立刻失敗：

執行結果（某一次）：

```text
--- FAIL: TestTemplateAllocs (0.00s)
    route_test.go:50: Template("/") allocs = 2, want at most 1
    route_test.go:50: Template("/healthz") allocs = 2, want at most 1
    route_test.go:50: Template("/users/7") allocs = 2, want at most 1
    route_test.go:50: Template("/users/48213/orders/90177/items/3") allocs = 2, want at most 1
    route_test.go:50: Template("/users/48213/profile") allocs = 2, want at most 1
    route_test.go:50: Template("/v2/users/abc123/orders/") allocs = 2, want at most 1
    route_test.go:50: Template("//1//2") allocs = 2, want at most 1
FAIL
```

**第二層，每次 CI：`-benchtime=1x`。** `go test -run='^$' -bench=. -benchtime=1x ./...` 讓每個 benchmark 只跑一遍，不量速度，只確保它還能跑、不會 panic。本書的 `.github/workflows/deploy.yml` 就有這一步。

**第三層，固定機器：用 `benchstat` 比較時間。** 下面這份設定不在本書的 CI 執行，只是起點。它在一台專用機器上，先跑 PR 的 merge base，再跑 PR，各 `-count=10`，把 `benchstat` 的結果寫進 job 摘要，由 reviewer 判讀 `±` 與 p 值：

```yaml
name: bench-compare
on:
  pull_request:
    paths: ["**.go"]
concurrency:
  group: bench-machine # 專用機器一次只做一個比較
jobs:
  benchstat:
    runs-on: [self-hosted, bench]
    env:
      BASE_REF: ${{ github.base_ref }}
    steps:
      - uses: actions/checkout@v6
        with:
          fetch-depth: 0
      - name: 基準（merge base）
        run: |
          git worktree add ../base "$(git merge-base "origin/$BASE_REF" HEAD)"
          (cd ../base && go test -run='^$' -bench=. -benchmem -count=10 ./... > ../old.txt)
      - name: PR
        run: go test -run='^$' -bench=. -benchmem -count=10 ./... > ../new.txt
      - name: 比較
        run: go run golang.org/x/perf/cmd/benchstat@v0.0.0-20261009192801-be2c69fb417e ../old.txt ../new.txt | tee -a "$GITHUB_STEP_SUMMARY"
```

基準與 PR 在同一台機器、同一個 job 裡先後跑，比的是兩邊的差，不是和上週的數字比。

## 取捨

- **固定機器要花錢。** 專用機器要買、要維護，job 要排隊；一次比較是兩邊各十遍的 benchmark，PR 越多越慢。機器升級或換了系統，舊的結果就不能再拿來比。
- **門檻怎麼設。** `AllocsPerRun` 的門檻直接設成現在的值，不留餘裕：它是確定的，有意增加配置時，改測試並在 PR 說明理由。時間不設數字門檻，由 reviewer 看 `benchstat` 的顯著差異與幅度決定；真要自動擋，只擋「顯著而且幅度很大」的。
- **不是每個專案都需要時間的比較。** 多數服務沒有哪段程式的耗時值得每個 PR 都盯。只有量過、確定是熱點的少數函式，才值得寫 `AllocsPerRun`；只有函式庫或延遲預算很緊的路徑，才值得養一台機器。為了湊數而到處加門檻，和第 5 章〈[覆蓋率的意義與限制](../chapter5/11_coverage_meaning_limits.md)〉的硬門檻是同一個道理：數字會被迎合，最後失去意義。
- **`AllocsPerRun` 看不到時間。** 不配置但變慢的改動（演算法變差、多繞一圈）它擋不住，那一類靠第三層，或靠熱點上的 benchmark 在 code review 時人工比較。

before 在這些情況下反而是對的：機器是自己專用的、不跟別人共用，門檻故意設得很寬，只想擋住「慢了十倍」的災難性回歸。這時單次 ns/op 是便宜的哨兵，前提是雜訊遠小於門檻。

## 重點整理

- 單次 benchmark 的 ns/op 是一個樣本，共用 runner 上的雜訊就能讓同一份程式時好時壞；這樣的紅燈最後會被重跑或關掉。
- 配置次數用 `AllocsPerRun` 在測試裡斷言，每次 CI 都確定地擋，`-race` 下也成立；改寫要有測試保證結果和舊寫法一致。
- `-benchtime=1x` 只確保 benchmark 還能跑，不量速度。
- 時間的比較留給固定機器上的 `benchstat`：同機器、`-count=10`、以 merge base 為基準，由人判讀顯著與否。
- 沒有需要天天盯的熱點，就不需要時間的比較；先決定要守什麼，再決定要多貴的守法。
