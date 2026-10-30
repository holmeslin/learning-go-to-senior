package httpmetrics

import (
	"context"
	"database/sql"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// PoolStats 是能回報連線池統計的東西，*sql.DB 就是。
type PoolStats interface {
	Stats() sql.DBStats
}

// ANCHOR: pool

// ObservePool 把 src 的連線池統計匯出成非同步的指標。
// 數值在每次收集時才讀：目前的連線數用量測值（gauge），
// DBStats.WaitCount 本來就是累計值，用非同步的計數器直接回報它，不自己加減。
// 用完要呼叫回傳的 Registration 的 Unregister。
func ObservePool(mp metric.MeterProvider, src PoolStats) (metric.Registration, error) {
	meter := mp.Meter("httpmetrics")
	conns, err := meter.Int64ObservableGauge("db.client.connection.count",
		metric.WithUnit("{connection}"))
	if err != nil {
		return nil, err
	}
	limit, err := meter.Int64ObservableGauge("db.client.connection.max",
		metric.WithUnit("{connection}"))
	if err != nil {
		return nil, err
	}
	waits, err := meter.Int64ObservableCounter("db.client.connection.waits",
		metric.WithUnit("{wait}"))
	if err != nil {
		return nil, err
	}
	return meter.RegisterCallback(func(_ context.Context, o metric.Observer) error {
		s := src.Stats() // 一次收集只讀一次，幾個指標看到同一刻的數字
		state := attribute.Key("db.client.connection.state")
		o.ObserveInt64(conns, int64(s.Idle), metric.WithAttributes(state.String("idle")))
		o.ObserveInt64(conns, int64(s.InUse), metric.WithAttributes(state.String("used")))
		o.ObserveInt64(limit, int64(s.MaxOpenConnections))
		o.ObserveInt64(waits, s.WaitCount)
		return nil
	}, conns, limit, waits)
}

// ANCHOR_END: pool
