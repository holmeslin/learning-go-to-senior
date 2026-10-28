package blobsum_test

import (
	"fmt"
	"testing"

	blobsum "github.com/holmeslin/learning-go-to-senior/examples/chapter8/01_scheduler_gmp/before"
)

func BenchmarkSum(b *testing.B) {
	buf := make([]byte, 16<<10)
	for _, n := range []int{1_000, 100_000} {
		blobs := make([][]byte, n)
		for i := range blobs {
			blobs[i] = buf
		}
		b.Run(fmt.Sprintf("blobs=%d", n), func(b *testing.B) {
			for b.Loop() {
				blobsum.Sum(blobs)
			}
		})
	}
}
