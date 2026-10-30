package voucher_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"

	voucher "github.com/holmeslin/learning-go-to-senior/examples/chapter9/01_structured_logging/after"
)

// ANCHOR: test

func TestJSONFields(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	r := voucher.New(slog.New(slog.NewJSONHandler(&buf, nil)), voucher.Voucher{Code: "A1", Cents: 100})
	_ = r.Redeem(t.Context(), "u4", "ZZ")
	_ = r.Redeem(t.Context(), "u42", "A1")
	_ = r.Redeem(t.Context(), "u42", "A1")

	var u4, u42 []map[string]any
	dec := json.NewDecoder(&buf)
	for dec.More() {
		var rec map[string]any
		if err := dec.Decode(&rec); err != nil {
			t.Fatal(err)
		}
		switch rec["user_id"] {
		case "u4":
			u4 = append(u4, rec)
		case "u42":
			u42 = append(u42, rec)
		}
	}

	// 依欄位篩選：u4 只有一筆，不會混進 u42 的紀錄。
	if len(u4) != 1 || len(u42) != 2 {
		t.Fatalf("got %d records for u4 and %d for u42, want 1 and 2", len(u4), len(u42))
	}
	if u4[0]["level"] != "WARN" || u4[0]["msg"] != "redeem failed" ||
		u4[0]["code"] != "ZZ" || u4[0]["error"] != "unknown voucher" {
		t.Errorf("unexpected failure record: %v", u4[0])
	}
	// 自訂型別決定自己的欄位；JSON 的數字解析出來是 float64。
	v, ok := u42[0]["voucher"].(map[string]any)
	if !ok || v["code"] != "A1" || v["cents"] != float64(100) {
		t.Errorf("voucher = %v, want {code: A1, cents: 100}", u42[0]["voucher"])
	}
}

// ANCHOR_END: test

func TestLibraryLeavesDefaultLoggerAlone(t *testing.T) {
	var global, injected bytes.Buffer
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&global, nil)))
	t.Cleanup(func() { slog.SetDefault(old) })

	r := voucher.New(slog.New(slog.NewTextHandler(&injected, nil)), voucher.Voucher{Code: "A1", Cents: 100})
	_ = r.Redeem(t.Context(), "u1", "A1")

	if global.Len() != 0 {
		t.Errorf("default logger received %q, want nothing", global.String())
	}
	if injected.Len() == 0 {
		t.Error("injected logger received nothing")
	}
}

func TestNilLoggerDiscards(t *testing.T) {
	t.Parallel()
	r := voucher.New(nil, voucher.Voucher{Code: "A1", Cents: 100})
	if err := r.Redeem(t.Context(), "u1", "A1"); err != nil {
		t.Errorf("first redeem: %v", err)
	}
	if err := r.Redeem(t.Context(), "u1", "A1"); err == nil {
		t.Error("second redeem succeeded, want error")
	}
}
