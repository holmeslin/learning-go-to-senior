# 匯出名稱與 API 穩定性

## 本集目標

改一個已經發布的套件前，先判斷這個改動會不會讓別人的程式編譯失敗；能用「只新增、不修改」演進，就不要動既有的匯出名稱。

## 情境

你維護一個內部共用的 `notify` 套件，v1 已經有三個服務在用。現在要支援 context 取消、批次送出，還要給訊息加優先順序，準備發 v1.1。匯出的基本規則，前作〈[匯出](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/03_exported.html)〉與〈[文件註解與 `go doc`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter8/07_doc_comments.html)〉已經講過，這集談的是：匯出之後，名稱就是對呼叫端的承諾。

## 常見寫法

直接把 API 改成「理想的樣子」。v1 長這樣：

```text
type Message struct{ To, Body string }
func NewMessage(to, body string) Message
type Sender interface{ Send(m Message) error }
func Send(s Sender, m Message) error
```

v1.1 順手做了四件事：`NewMessage` 改名叫 `New`；`Send` 加上 `ctx` 參數；`Sender` 加上 `SendBatch` 方法；`Message` 多一個 `Priority` 欄位。

```go,ignore
{{#include ../../../examples/chapter1/04_exported_api_stability/before/notify/notify.go:api}}
```

完整程式碼在 `examples/chapter1/04_exported_api_stability/before/`。

## 問題在哪

套件本身編譯、`go vet` 都沒有問題，壞的是呼叫端。下面是 v1 的呼叫端，一行都沒改：

```text
package main

import "example.com/app/notify"

type logSender struct{}

func (logSender) Send(m notify.Message) error { return nil }

func main() {
	var s notify.Sender = logSender{}
	m := notify.NewMessage("amy@example.com", "你好")
	old := notify.Message{"bob@example.com", "早安"}
	_ = notify.Send(s, m)
	_ = old
}
```

對 v1 它能編譯；對上面的 v1.1，`go build` 的結果是（`notify` 就是 `before/notify`）：

編譯錯誤：

```text
./main.go:10:24: cannot use logSender{} (value of struct type logSender) as notify.Sender value in variable declaration: logSender does not implement notify.Sender (missing method SendBatch)
./main.go:11:14: undefined: notify.NewMessage
./main.go:12:51: too few values in struct literal of type notify.Message
./main.go:13:21: not enough arguments in call to notify.Send
	have (notify.Sender, unknown type)
	want (context.Context, notify.Sender, notify.Message)
```

四個改動，四種錯誤，每一種都是呼叫端要自己動手修：

- 改名：舊名稱消失，`undefined`。
- 改簽名：參數個數或型別不同，呼叫處全部要改。
- 介面加方法：呼叫端自己寫的實作，立刻不再滿足介面。連「只是把介面拿來用」的人都不會受影響，「實作它」的人才會壞，所以很容易漏看。
- struct 加欄位：用欄位名稱建立的程式不受影響，依欄位順序寫的程式會壞。

錯誤值也是公開 API 的一部分，改了訊息或型別，呼叫端的 `errors.Is` 同樣會失效（第 2 章〈錯誤設計〉會談）。

## 較好的做法

只新增、不修改。同樣三個需求，這樣做：

- context：保留 `Send`，另外新增 `SendContext`；`Send` 內部轉呼它。
- 批次：`Sender` 維持只有 `Send`，批次能力放進新的 `BatchSender` 介面，用型別斷言判斷。
- 優先順序：新增 `Priority` 欄位，零值代表一般優先。用欄位名稱建立的舊程式不受影響；依欄位順序寫的程式一樣會壞（見下面的取捨）。

```go,ignore
{{#include ../../../examples/chapter1/04_exported_api_stability/after/notify/notify.go:sendcontext}}
```

```go,ignore
{{#include ../../../examples/chapter1/04_exported_api_stability/after/notify/notify.go:sendall}}
```

接著用外部測試套件把 v1 的公開 API 鎖住。只要有一行編譯失敗，就代表你正在破壞 v1 的呼叫端：

```go,ignore
{{#include ../../../examples/chapter1/04_exported_api_stability/after/notify/api_test.go:lock}}
```

`v1Sender` 就是前面 `logSender` 的角色：如果有人在 `Sender` 加方法，這個檔案在 `go test` 時就編譯不過，不用等到呼叫端回報。這個鎖擋不住依順序寫欄位的 `notify.Message{"a", "b"}`：`Priority` 已經加進去了，這種寫法放進測試自己就編譯不過，所以它沒辦法同時當成「v1 還能編譯」的證明。`Example` 則同時示範 v1 的舊寫法與 v1.1 的新功能，`go test` 會比對輸出：

```go,ignore
{{#include ../../../examples/chapter1/04_exported_api_stability/after/notify/example_test.go:example}}
```

完整程式碼在 `examples/chapter1/04_exported_api_stability/after/`。這個範例只示範新增；`New` 這種「確實想換名字」的需求，要保留舊名稱並導向新的，本章〈棄用（deprecation）與相容性〉會談。

## 取捨

只新增不修改，會讓 API 越長越大：`Send` 與 `SendContext` 並存，以後誰也不敢刪。可選介面加型別斷言也比直接加方法繞，呼叫端要讀文件才知道有 `BatchSender`。

這些代價在沒有外部使用者時不值得付。套件只有你一個人在用、還沒發布，或所有呼叫端都在同一個 repo、改完能一起編譯，那就直接重構，改成理想的樣子，比留下一堆歷史包袱划算。要守相容的，是你改了之後「看不到、改不動」的呼叫端。

另外，struct 加欄位對「依順序寫欄位」的程式不相容，Go 的相容性承諾也不涵蓋這種寫法；與其替它們設防，不如在文件註解寫明請用欄位名稱建立（`Message` 就這樣寫了），`go vet` 也會對別的套件的 struct 這樣寫提出警告。

## 重點整理

- 匯出的名稱、簽名、介面的方法集合、struct 的欄位，都是對呼叫端的承諾。
- 改名、改簽名、介面加方法、struct 加欄位（對依順序寫的程式）會讓呼叫端編譯失敗，而且壞的是呼叫端，不是套件自己。
- 演進時只新增：新函式、新的可選介面、新欄位，舊的保持原樣。
- 用外部測試套件的編譯期斷言鎖住 v1 的函式、介面與用欄位名稱建立的寫法，這些被破壞時 `go test` 就會發現；依順序寫欄位的程式鎖不住，要靠文件註解與 `go vet`。
- 還沒有外部使用者時，大膽重構比維持相容更划算。
