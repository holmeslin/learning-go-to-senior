package split_test

import (
	"testing"

	split "github.com/holmeslin/learning-go-to-senior/examples/chapter5/11_coverage_meaning_limits/before"
)

// ANCHOR: tests

func TestSplit(t *testing.T) {
	_, _ = split.Split(100, 1, 1, 1)
}

func TestSplitNoWeights(t *testing.T) {
	_, _ = split.Split(100)
}

func TestSplitBadWeight(t *testing.T) {
	_, _ = split.Split(100, 1, -1)
}

// ANCHOR_END: tests
