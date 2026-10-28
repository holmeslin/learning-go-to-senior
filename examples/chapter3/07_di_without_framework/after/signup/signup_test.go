package signup_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter3/07_di_without_framework/after/signup"
)

// ANCHOR: graph

// fakeMailer 記下收到的信，不寄任何東西；設了 err 就讓 Send 失敗。
type fakeMailer struct {
	sent []string
	err  error
}

func (f *fakeMailer) Send(to, subject string) error {
	if f.err != nil {
		return f.err
	}
	f.sent = append(f.sent, to)
	return nil
}

// newService 組出和 main 同一張圖，只是把真的寄信換成替身。
func newService() (*signup.Service, *fakeMailer) {
	mailer := &fakeMailer{}
	return signup.NewService(signup.NewStore(), mailer), mailer
}

// ANCHOR_END: graph

// 下面兩個測試用同一個信箱，又同時執行：各自組了一張圖，所以不會互相影響。

func TestRegisterSendsWelcomeMail(t *testing.T) {
	t.Parallel()
	svc, mailer := newService()

	if err := svc.Register("ann@example.com"); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if want := []string{"ann@example.com"}; !slices.Equal(mailer.sent, want) {
		t.Errorf("sent = %v, want %v", mailer.sent, want)
	}
}

func TestRegisterTwiceSendsOneMail(t *testing.T) {
	t.Parallel()
	svc, mailer := newService()

	_ = svc.Register("ann@example.com")
	err := svc.Register("ann@example.com")

	if !errors.Is(err, signup.ErrExists) {
		t.Errorf("second Register() error = %v, want ErrExists", err)
	}
	if len(mailer.sent) != 1 {
		t.Errorf("sent %d mails, want 1", len(mailer.sent))
	}
}

func TestRegisterReportsSendFailure(t *testing.T) {
	t.Parallel()
	svc, mailer := newService()
	errDown := errors.New("smtp down")
	mailer.err = errDown

	if err := svc.Register("ann@example.com"); !errors.Is(err, errDown) {
		t.Errorf("Register() error = %v, want it to wrap %v", err, errDown)
	}
}
