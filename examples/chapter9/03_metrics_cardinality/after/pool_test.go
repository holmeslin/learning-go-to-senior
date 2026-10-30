package httpmetrics_test

import (
	"database/sql"
	"maps"
	"testing"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	httpmetrics "github.com/holmeslin/learning-go-to-senior/examples/chapter9/03_metrics_cardinality/after"
)

// ANCHOR: pool

// fakePool 讓測試決定連線池「現在」的統計。
type fakePool struct{ stats sql.DBStats }

func (p *fakePool) Stats() sql.DBStats { return p.stats }

func TestPoolStatsAreReadAtCollect(t *testing.T) {
	t.Parallel()
	pool := &fakePool{stats: sql.DBStats{MaxOpenConnections: 10, InUse: 3, Idle: 2, WaitCount: 5}}
	reader := sdkmetric.NewManualReader()
	reg, err := httpmetrics.ObservePool(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)), pool)
	if err != nil {
		t.Fatal(err)
	}
	defer reg.Unregister()

	count := func() map[string]int64 {
		return maps.Clone(values(collect(reader)["db.client.connection.count"]))
	}
	want := map[string]int64{
		"db.client.connection.state=idle": 2,
		"db.client.connection.state=used": 3,
	}
	if got := count(); !maps.Equal(got, want) {
		t.Errorf("first collect = %v, want %v", got, want)
	}

	pool.stats.InUse, pool.stats.Idle, pool.stats.WaitCount = 10, 0, 12 // 池子被用滿，開始排隊
	want = map[string]int64{
		"db.client.connection.state=idle": 0,
		"db.client.connection.state=used": 10,
	}
	if got := count(); !maps.Equal(got, want) {
		t.Errorf("second collect = %v, want %v", got, want)
	}
	if got := values(collect(reader)["db.client.connection.waits"]); got[""] != 12 {
		t.Errorf("waits = %v, want the pool's cumulative WaitCount 12", got)
	}
	if got := values(collect(reader)["db.client.connection.max"]); got[""] != 10 {
		t.Errorf("max = %v, want 10", got)
	}
}

// ANCHOR_END: pool

func TestPoolInstrumentUnitsAndKinds(t *testing.T) {
	t.Parallel()
	reader := sdkmetric.NewManualReader()
	reg, err := httpmetrics.ObservePool(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)), &fakePool{})
	if err != nil {
		t.Fatal(err)
	}
	defer reg.Unregister()

	got := collect(reader)
	if m := got["db.client.connection.count"]; m.Unit != "{connection}" {
		t.Errorf("count unit = %q", m.Unit)
	} else if _, ok := m.Data.(metricdata.Gauge[int64]); !ok {
		t.Errorf("count data = %T, want a gauge", m.Data)
	}
	if m := got["db.client.connection.waits"]; m.Unit != "{wait}" {
		t.Errorf("waits unit = %q", m.Unit)
	} else if s, ok := m.Data.(metricdata.Sum[int64]); !ok || !s.IsMonotonic {
		t.Errorf("waits data = %T, want a monotonic sum", m.Data)
	}
}
