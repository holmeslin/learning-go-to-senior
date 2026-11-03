// Package dispatch 定期掃描到期的提醒並送出：先把逾期的記為失敗，再挑出到期的與該重試的，
// 每一則依「限流、寫紀錄、送出、更新狀態」的順序處理。
//
// 送出用並行上限（errgroup 的 SetLimit）加每秒的限流；暫時失敗用退避加 jitter 重試，
// 重試的時間點只會落在掃描的時間，期限是預定時間後 store.SendWindow。
//
// 每批掃描留下一個 span、每次送出（呼叫供應商）一個子 span，以及三個指標：reminder.deliveries（標籤只有 result）、
// reminder.scan.duration、reminder.overdue.unsent。Options 的 Meter 與 Tracer 是 nil 時什麼也不記。
package dispatch

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/internal/push"
	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/internal/store"
)

// 重試退避的預設值。第一次重試前等待時間的上限，之後每次加倍，直到 retryMax。
// 重試只會在掃描時發生，所以 retryBase 取和預設掃描間隔一樣。
const (
	defaultRetryBase = 5 * time.Minute
	defaultRetryMax  = 30 * time.Minute
)

// ANCHOR: sender

// Sender 送出一則通知。介面定義在使用端：dispatch 只需要「送出」這一個動作，
// push.Client 滿足它，測試用一個記錄呼叫的假實作。
type Sender interface {
	// Send 送出 n。失敗時的錯誤要包著 push.ErrTransient（會重試）或 push.ErrPermanent（放棄）；
	// 分不出類別的錯誤一律當成永久失敗，因為無法證明重送是安全的。
	Send(ctx context.Context, n push.Notification) error
}

// ANCHOR_END: sender

// Options 是 Dispatcher 的設定。
type Options struct {
	Interval      time.Duration // 掃描間隔，必須大於零
	Concurrency   int           // 同時進行中的送出上限，至少 1
	RatePerSecond float64       // 每秒最多送出幾則，必須大於零
	QueryTimeout  time.Duration // 每個資料庫查詢的逾時，必須大於零

	// RetryBase 是第一次重試前，等待時間的上限，之後每次加倍；零值用預設的 5 分鐘。
	RetryBase time.Duration
	// RetryMax 是等待時間上限的天花板，不得小於 RetryBase；零值用預設的 30 分鐘。
	RetryMax time.Duration

	// Jitter 回傳 [0, limit) 之間的等待時間。nil 時用 math/rand/v2 的全域亂數。
	// 這個欄位是注入亂數來源的接縫，讓測試能斷言重試的時間點；注入的函式要自己負責並行安全。
	Jitter func(limit time.Duration) time.Duration

	// Logger 記錄掃描的結果與錯誤；nil 時用 slog.Default()。日誌不記裝置 token。
	Logger *slog.Logger

	// Meter 建立指標用；nil 時不記錄指標。指標只有 result 這個標籤，沒有活動、報名者這類高基數的標籤。
	Meter metric.Meter
	// Tracer 建立 span 用：每批一個，每次送出一個子 span；nil 時不記錄追蹤。
	Tracer trace.Tracer
}

// Summary 是一次掃描的結果。
type Summary struct {
	Expired  int // 寄送中或待重試、超過期限而記為失敗
	Missed   int // 從未嘗試、超過期限而直接記為失敗，沒有送出
	Sent     int // 送出成功
	Retrying int // 暫時失敗，排在之後的掃描重試
	Failed   int // 永久失敗
	Skipped  int // 紀錄寫不進去（別人已經處理過），沒有送出
	Errored  int // 資料庫操作失敗，結果沒有記錄下來
}

// Dispatcher 掃描到期的提醒並送出。
type Dispatcher struct {
	store   *store.Store
	sender  Sender
	limiter *rate.Limiter
	opts    Options
	log     *slog.Logger

	tracer     trace.Tracer
	deliveries metric.Int64Counter     // 每個結果的提醒數，標籤只有 result
	duration   metric.Float64Histogram // 每一批掃描花的時間（秒）
}

// New 回傳 Dispatcher；opts 不合法時回傳錯誤。
func New(st *store.Store, sender Sender, opts Options) (*Dispatcher, error) {
	if opts.RetryBase == 0 {
		opts.RetryBase = defaultRetryBase
	}
	if opts.RetryMax == 0 {
		opts.RetryMax = defaultRetryMax
	}
	switch {
	case opts.Interval <= 0:
		return nil, errors.New("dispatch: Interval must be positive")
	case opts.Concurrency < 1:
		return nil, errors.New("dispatch: Concurrency must be at least 1")
	case opts.RatePerSecond <= 0:
		return nil, errors.New("dispatch: RatePerSecond must be positive")
	case opts.QueryTimeout <= 0:
		return nil, errors.New("dispatch: QueryTimeout must be positive")
	case opts.RetryBase < 0 || opts.RetryMax < opts.RetryBase:
		return nil, errors.New("dispatch: want 0 < RetryBase <= RetryMax")
	}
	log := opts.Logger
	if log == nil {
		log = slog.Default()
	}
	d := &Dispatcher{
		store: st,
		// burst 是 1：每秒 50 次是供應商的限制，請求要均勻地分散，不能一開始就湧出一整批。
		limiter: rate.NewLimiter(rate.Limit(opts.RatePerSecond), 1),
		opts:    opts,
		log:     log,
		tracer:  opts.Tracer,
	}
	if d.tracer == nil {
		d.tracer = tracenoop.NewTracerProvider().Tracer("")
	}
	d.sender = tracedSender{next: sender, tracer: d.tracer}
	meter := opts.Meter
	if meter == nil {
		meter = metricnoop.NewMeterProvider().Meter("")
	}
	if err := d.instrument(meter); err != nil {
		return nil, err
	}
	return d, nil
}

// ANCHOR: run

// Run 立刻掃描一次，之後每個 Interval 掃描一次，直到 ctx 結束。
// ctx 結束代表「不要再拿新的提醒」：進行中的送出會做完、結果寫進紀錄，Run 才回傳。
// 單次掃描失敗只記錄、不停止，下一個間隔再試。
func (d *Dispatcher) Run(ctx context.Context) {
	ticker := time.NewTicker(d.opts.Interval)
	defer ticker.Stop()
	for {
		sum, err := d.Scan(ctx)
		switch {
		case err != nil && ctx.Err() == nil:
			d.log.Error("scan failed", "error", err)
		case err == nil:
			d.log.Info("scan finished",
				"expired", sum.Expired, "missed", sum.Missed, "sent", sum.Sent, "retrying", sum.Retrying,
				"failed", sum.Failed, "skipped", sum.Skipped, "errored", sum.Errored)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// ANCHOR_END: run

// Scan 做一次掃描（見 scan），並留下這一批的觀測資料：一個 span、批次花的時間，以及各個結果的提醒數。
// 回傳時，這次開始送出的提醒都已經有結果。ctx 結束時不再開始新的提醒，進行中的做完。
// 只有「失敗記錄」與「挑出提醒」這兩個查詢失敗才回傳錯誤；單一提醒的失敗記在 Summary、指標與日誌裡。
func (d *Dispatcher) Scan(ctx context.Context) (Summary, error) {
	ctx, span := d.tracer.Start(ctx, "reminder.scan")
	defer span.End()
	start := time.Now()

	sum, err := d.scan(ctx)

	d.duration.Record(ctx, time.Since(start).Seconds())
	for _, r := range sum.results() {
		d.deliveries.Add(ctx, int64(r.n), metric.WithAttributes(attribute.String("result", r.name)))
		span.SetAttributes(attribute.Int(r.name, r.n))
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "scan failed")
	}
	return sum, err
}

// result 是一種結果與這一批裡有幾則；名稱同時是指標的 result 標籤與 span 的屬性名稱。
type result struct {
	name string
	n    int
}

// results 把 Summary 展開成固定的七種結果。標籤的值只有這七種，所以時間序列的數量不隨提醒的數量成長。
func (s Summary) results() []result {
	return []result{
		{"expired", s.Expired}, {"missed", s.Missed}, {"sent", s.Sent}, {"retrying", s.Retrying},
		{"failed", s.Failed}, {"skipped", s.Skipped}, {"errored", s.Errored},
	}
}

// ANCHOR: tracedsender

// tracedSender 在每次送出外面包一個 span，成為這一批 span 的子 span：它只量供應商那一段花的時間。
// 屬性有活動編號（追蹤不是指標，高基數沒關係）與結果，不帶裝置 token，也不記錄錯誤訊息。
// 包在 Sender 外面而不是寫進 deliver：限流與寫紀錄的程式不必知道追蹤的存在，
// 而且等限流、被取消而沒有送出的提醒不會留下 span。
type tracedSender struct {
	next   Sender
	tracer trace.Tracer
}

func (s tracedSender) Send(ctx context.Context, n push.Notification) error {
	ctx, span := s.tracer.Start(ctx, "reminder.send", trace.WithAttributes(attribute.Int64("event.id", n.EventID)))
	defer span.End()
	err := s.next.Send(ctx, n)
	result := "sent"
	switch {
	case err == nil:
	case errors.Is(err, push.ErrTransient):
		result = "transient"
	default:
		result = "permanent" // 分不出類別的，dispatch 也當成永久失敗
	}
	span.SetAttributes(attribute.String("result", result))
	if err != nil {
		span.SetStatus(codes.Error, result)
	}
	return err
}

// ANCHOR_END: tracedsender

// ANCHOR: instrument

// instrument 建立指標：送出結果的計數、批次時間，以及逾期未寄的數量。
// 逾期未寄的數量在收集時才查資料庫：服務卡住、沒有在掃描的時候，它仍然是準的。
func (d *Dispatcher) instrument(m metric.Meter) error {
	var errs [3]error
	d.deliveries, errs[0] = m.Int64Counter("reminder.deliveries",
		metric.WithDescription("每個結果的提醒數"), metric.WithUnit("{reminder}"))
	d.duration, errs[1] = m.Float64Histogram("reminder.scan.duration",
		metric.WithDescription("每一批掃描花的時間"), metric.WithUnit("s"),
		// 預設的區間是為毫秒設計的；一批的時間從不到一秒到超過掃描間隔（一批送不完）。
		metric.WithExplicitBucketBoundaries(0.1, 1, 5, 15, 30, 60, 120, 300, 600, 1200, 3600))
	_, errs[2] = m.Int64ObservableGauge("reminder.overdue.unsent",
		metric.WithDescription("已過預定時間、還沒寄出的提醒數"), metric.WithUnit("{reminder}"),
		metric.WithInt64Callback(d.observeOverdue))
	if err := errors.Join(errs[:]...); err != nil {
		return fmt.Errorf("create instruments: %w", err)
	}
	return nil
}

// observeOverdue 是 reminder.overdue.unsent 的 callback。查詢失敗只記錄：
// 資料庫一時有問題，不該讓整份指標快照跟著失敗。
func (d *Dispatcher) observeOverdue(ctx context.Context, o metric.Int64Observer) error {
	ctx, cancel := context.WithTimeout(ctx, d.opts.QueryTimeout)
	defer cancel()
	n, err := d.store.CountOverdue(ctx, time.Now())
	if err != nil {
		d.log.Warn("count overdue reminders failed", "error", err)
		return nil
	}
	o.Observe(int64(n))
	return nil
}

// ANCHOR_END: instrument

// scan 做一次掃描：先把超過期限的記為失敗，再挑出到期的與該重試的，在並行上限與限流之內送出。
func (d *Dispatcher) scan(ctx context.Context) (Summary, error) {
	var sum Summary
	now := time.Now()

	qctx, cancel := context.WithTimeout(ctx, d.opts.QueryTimeout)
	var err error
	sum.Expired, sum.Missed, err = d.store.FailOverdue(qctx, now)
	cancel()
	if err != nil {
		return sum, err
	}

	qctx, cancel = context.WithTimeout(ctx, d.opts.QueryTimeout)
	due, err := d.store.Due(qctx, now)
	cancel()
	if err != nil {
		return sum, err
	}

	var mu sync.Mutex
	var g errgroup.Group
	g.SetLimit(d.opts.Concurrency)
	for _, r := range due {
		if ctx.Err() != nil {
			break
		}
		g.Go(func() error {
			o := d.deliver(ctx, r)
			mu.Lock()
			defer mu.Unlock()
			switch o {
			case sent:
				sum.Sent++
			case retrying:
				sum.Retrying++
			case failed:
				sum.Failed++
			case skipped:
				sum.Skipped++
			case errored:
				sum.Errored++
			}
			return nil
		})
	}
	_ = g.Wait() // 每個函式都回傳 nil；errgroup 在這裡只負責並行上限
	return sum, nil
}

// outcome 是一則提醒這次處理的結果。
type outcome int

const (
	aborted  outcome = iota // 等限流的時候 ctx 結束，還沒有寫紀錄
	sent                    // 送出成功
	retrying                // 暫時失敗，排了重試
	failed                  // 永久失敗
	skipped                 // 紀錄寫不進去，別人處理過了
	errored                 // 資料庫操作失敗
)

// ANCHOR: deliver

// deliver 處理一則提醒：限流、寫紀錄、送出、更新狀態。
func (d *Dispatcher) deliver(ctx context.Context, r store.Reminder) outcome {
	// 限流在寫紀錄之前：等的時候 ctx 結束，丟掉的只是還沒寫紀錄的提醒，下次掃描會再挑到。
	// 寫完紀錄到送出之間，沒有任何會被取消的等待。
	if err := d.limiter.Wait(ctx); err != nil {
		return aborted
	}
	// 從這裡起這一則要做完：取消只決定「要不要開始下一則」，不切斷進行中的送出，
	// 也不讓結果寫不進去而留下停在寄送中的紀錄。逾時由每個請求與每個查詢自己的期限負責。
	ctx = context.WithoutCancel(ctx)
	attempt := r.Attempts + 1

	ok, err := d.begin(ctx, r)
	switch {
	case err != nil:
		d.log.Error("write delivery record failed", "event", r.EventID, "registrant", r.RegistrantID, "error", err)
		return errored
	case !ok:
		return skipped
	}

	err = d.sender.Send(ctx, push.Notification{
		Key:         fmt.Sprintf("reminder-%d-%d", r.EventID, r.RegistrantID),
		DeviceToken: r.DeviceToken,
		EventID:     r.EventID,
		Title:       "活動提醒",
		Body:        "你報名的活動將在一天後開始。",
	})

	qctx, cancel := context.WithTimeout(ctx, d.opts.QueryTimeout)
	defer cancel()
	now := time.Now()
	var o outcome
	switch {
	case err == nil:
		o, err = sent, d.store.MarkSent(qctx, r, now)
	case errors.Is(err, push.ErrTransient):
		o, err = retrying, d.store.MarkRetry(qctx, r, now.Add(d.backoff(attempt)), err.Error(), now)
	default:
		o, err = failed, d.store.MarkFailed(qctx, r, err.Error(), now)
	}
	if err != nil {
		d.log.Error("record delivery result failed", "event", r.EventID, "registrant", r.RegistrantID, "error", err)
		return errored
	}
	return o
}

// begin 在送出之前寫紀錄；回傳 false 代表不要送。
func (d *Dispatcher) begin(ctx context.Context, r store.Reminder) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, d.opts.QueryTimeout)
	defer cancel()
	return d.store.Begin(ctx, r, time.Now())
}

// ANCHOR_END: deliver

// backoff 是第 attempt 次（從 1 起算）嘗試失敗之後要等多久：
// 上限從 RetryBase 起每次加倍、最多到 RetryMax，在 [0, 上限) 之間隨機取一個（full jitter）。
// 實際重試的時間是下一次掃描，等待時間不到一個間隔時就是下一個間隔。
func (d *Dispatcher) backoff(attempt int) time.Duration {
	limit := d.opts.RetryBase
	for i := 1; i < attempt && limit < d.opts.RetryMax; i++ {
		limit *= 2
	}
	limit = min(limit, d.opts.RetryMax)
	if d.opts.Jitter != nil {
		return d.opts.Jitter(limit)
	}
	return rand.N(limit)
}
