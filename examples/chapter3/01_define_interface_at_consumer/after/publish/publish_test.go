package publish_test

import (
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter3/01_define_interface_at_consumer/after/publish"
)

// ANCHOR: fake

// fake 是 Publish 測試用的替身。publish.Blobs 只有兩個方法，所以只需要寫兩個。
type fake struct {
	data    map[string][]byte
	calls   []string
	putFail error
}

func (f *fake) Get(key string) ([]byte, error) {
	f.calls = append(f.calls, "Get "+key)
	data, ok := f.data[key]
	if !ok {
		return nil, fmt.Errorf("get %q: not found", key)
	}
	return data, nil
}

func (f *fake) Put(key string, data []byte) error {
	f.calls = append(f.calls, "Put "+key)
	if f.putFail != nil {
		return f.putFail
	}
	f.data[key] = data
	return nil
}

// ANCHOR_END: fake

func TestPublishTrimsDraft(t *testing.T) {
	f := &fake{data: map[string][]byte{"drafts/42": []byte("  hello  \n")}}

	if err := publish.Publish(f, "42"); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if got := string(f.data["published/42"]); got != "hello" {
		t.Errorf("published/42 = %q, want %q", got, "hello")
	}
	if want := []string{"Get drafts/42", "Put published/42"}; !slices.Equal(f.calls, want) {
		t.Errorf("calls = %v, want %v", f.calls, want)
	}
}

func TestPublishMissingDraftWritesNothing(t *testing.T) {
	f := &fake{data: map[string][]byte{}}

	if err := publish.Publish(f, "42"); err == nil {
		t.Fatal("Publish succeeded without a draft")
	}
	if want := []string{"Get drafts/42"}; !slices.Equal(f.calls, want) {
		t.Errorf("calls = %v, want %v", f.calls, want)
	}
}

func TestPublishReportsWriteFailure(t *testing.T) {
	boom := errors.New("disk full")
	f := &fake{data: map[string][]byte{"drafts/42": []byte("hello")}, putFail: boom}

	err := publish.Publish(f, "42")
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want it to wrap %v", err, boom)
	}
}
