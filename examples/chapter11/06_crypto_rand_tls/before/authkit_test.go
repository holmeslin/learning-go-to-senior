package authkit_test

import (
	"testing"
	"time"

	authkit "github.com/holmeslin/learning-go-to-senior/examples/chapter11/06_crypto_rand_tls/before"
)

// ANCHOR: predict

func TestCodeIsPredictable(t *testing.T) {
	start := time.Unix(1_700_000_000, 0) // 服務啟動的時間（假的）
	svc := authkit.NewCodeGenerator(start)
	mine := svc.Next()   // 攻擊者替自己的帳號要一組重設碼
	victim := svc.Next() // 下一組發給受害者，攻擊者看不到

	// 攻擊者只知道服務是這一天之內啟動的：逐秒試，找出產生 mine 的種子。
	var guess string
	for s := start.Unix() - 86400; s <= start.Unix(); s++ {
		g := authkit.NewCodeGenerator(time.Unix(s, 0))
		if g.Next() == mine {
			guess = g.Next()
			break
		}
	}
	if guess != victim {
		t.Fatalf("guess = %q, victim = %q", guess, victim)
	}
}

// ANCHOR_END: predict
