package store_test

import (
	"errors"
	"testing"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter2/07_error_contract/after/store"
)

// ANCHOR: test

// TestFindErrorContract 鎖住 Find 文件註解承諾的錯誤：呼叫端依賴的是
// errors.Is 與 errors.AsType 的結果，不是錯誤訊息。
func TestFindErrorContract(t *testing.T) {
	tests := []struct {
		name      string
		id        int
		wantName  string
		wantFound bool // errors.Is(err, store.ErrNotFound)
		wantBadID bool // errors.AsType[*store.InvalidIDError] 成功
	}{
		{name: "found", id: 1, wantName: "amy"},
		{name: "missing", id: 42, wantFound: true},
		{name: "invalid", id: -1, wantBadID: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := store.Find(tt.id)
			if got != tt.wantName {
				t.Errorf("Find(%d) = %q, want %q", tt.id, got, tt.wantName)
			}
			if gotFound := errors.Is(err, store.ErrNotFound); gotFound != tt.wantFound {
				t.Errorf("errors.Is(err, ErrNotFound) = %v, want %v (err = %v)", gotFound, tt.wantFound, err)
			}
			idErr, gotBadID := errors.AsType[*store.InvalidIDError](err)
			if gotBadID != tt.wantBadID {
				t.Fatalf("errors.AsType[*InvalidIDError] = %v, want %v (err = %v)", gotBadID, tt.wantBadID, err)
			}
			if gotBadID && idErr.ID != tt.id {
				t.Errorf("InvalidIDError.ID = %d, want %d", idErr.ID, tt.id)
			}
			if tt.wantName != "" && err != nil {
				t.Errorf("Find(%d) returned unexpected error: %v", tt.id, err)
			}
		})
	}
}

// ANCHOR_END: test
