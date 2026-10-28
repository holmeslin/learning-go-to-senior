package split_test

import (
	"errors"
	"slices"
	"testing"

	split "github.com/holmeslin/learning-go-to-senior/examples/chapter5/11_coverage_meaning_limits/after"
)

// ANCHOR: shares

func TestSplitShares(t *testing.T) {
	tests := []struct {
		name    string
		total   int
		weights []int
		want    []int
	}{
		{"平分", 100, []int{1, 1}, []int{50, 50}},
		{"依比例", 1000, []int{3, 1}, []int{750, 250}},
		{"餘數歸第一份", 100, []int{1, 1, 1}, []int{34, 33, 33}},
		{"金額為 0", 0, []int{1, 2}, []int{0, 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := split.Split(tt.total, tt.weights...)
			if err != nil {
				t.Fatalf("Split(%d, %v) error = %v, want nil", tt.total, tt.weights, err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("Split(%d, %v) = %v, want %v", tt.total, tt.weights, got, tt.want)
			}
		})
	}
}

// ANCHOR_END: shares

// ANCHOR: errors

func TestSplitErrors(t *testing.T) {
	tests := []struct {
		name    string
		total   int
		weights []int
		want    error
	}{
		{"沒有權重", 100, nil, split.ErrNoWeights},
		{"權重為負", 100, []int{1, -1}, split.ErrBadWeight},
		{"權重為 0", 100, []int{1, 0}, split.ErrBadWeight},
		{"金額為負", -100, []int{1, 1}, split.ErrNegativeTotal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := split.Split(tt.total, tt.weights...)
			if !errors.Is(err, tt.want) {
				t.Errorf("Split(%d, %v) error = %v, want %v", tt.total, tt.weights, err, tt.want)
			}
			if got != nil {
				t.Errorf("Split(%d, %v) = %v, want nil shares on error", tt.total, tt.weights, got)
			}
		})
	}
}

// ANCHOR_END: errors
