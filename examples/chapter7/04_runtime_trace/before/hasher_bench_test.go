package passwd_test

import (
	"strconv"
	"sync"
	"testing"

	passwd "github.com/holmeslin/learning-go-to-senior/examples/chapter7/04_runtime_trace/before"
)

// iter 是 PBKDF2 的迭代次數。
const iter = 10000

// BenchmarkHashBurst 量「同時進來 burst 個登入請求，全部處理完要多久」。
func BenchmarkHashBurst(b *testing.B) {
	h := passwd.New(iter)
	defer h.Close()
	salt := []byte("bench-salt")

	for _, burst := range []int{1, 32} {
		b.Run("burst="+strconv.Itoa(burst), func(b *testing.B) {
			for b.Loop() {
				var wg sync.WaitGroup
				for range burst {
					wg.Go(func() {
						if _, err := h.Hash("hunter2", salt); err != nil {
							b.Error(err)
						}
					})
				}
				wg.Wait()
			}
		})
	}
}
