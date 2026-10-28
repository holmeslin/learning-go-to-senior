// Package hits 用一個擁有計數的 goroutine 保護各路徑的點閱數，
// 其他 goroutine 透過請求與回應 channel 存取。
package hits

// ANCHOR: counter

// Counter 記錄各路徑的點閱數；計數只有 loop 這個 goroutine 碰得到。
// 用完要呼叫 Close，否則 loop 永遠不會結束。
type Counter struct {
	incs    chan string
	gets    chan getReq
	done    chan struct{}
	stopped chan struct{}
}

type getReq struct {
	path  string
	reply chan int
}

// New 建立 Counter 並啟動擁有計數的 goroutine。
func New() *Counter {
	c := &Counter{
		incs:    make(chan string),
		gets:    make(chan getReq),
		done:    make(chan struct{}),
		stopped: make(chan struct{}),
	}
	go c.loop()
	return c
}

func (c *Counter) loop() {
	defer close(c.stopped)
	counts := map[string]int{}
	for {
		select {
		case path := <-c.incs:
			counts[path]++
		case req := <-c.gets:
			req.reply <- counts[req.path]
		case <-c.done:
			return
		}
	}
}

// Inc 把 path 的點閱數加一。Close 之後呼叫會永遠卡住。
func (c *Counter) Inc(path string) { c.incs <- path }

// Count 回傳 path 目前的點閱數。Close 之後呼叫會永遠卡住。
func (c *Counter) Count(path string) int {
	reply := make(chan int)
	c.gets <- getReq{path: path, reply: reply}
	return <-reply
}

// Close 停止 loop 並等它結束。只能呼叫一次，重複呼叫會 panic。
func (c *Counter) Close() {
	close(c.done)
	<-c.stopped
}

// ANCHOR_END: counter
