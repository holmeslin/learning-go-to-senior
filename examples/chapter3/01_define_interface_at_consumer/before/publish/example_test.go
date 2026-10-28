package publish_test

import (
	"fmt"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter3/01_define_interface_at_consumer/before/publish"
)

// ANCHOR: fake

// fake 是 Publish 測試用的替身。blob.Store 有七個方法，所以七個都得寫。
type fake struct {
	data  map[string][]byte
	calls []string
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
	f.data[key] = data
	return nil
}

func (f *fake) Delete(key string) error     { panic("not used") }
func (f *fake) Exists(key string) bool      { panic("not used") }
func (f *fake) List(prefix string) []string { panic("not used") }
func (f *fake) Copy(src, dst string) error  { panic("not used") }
func (f *fake) Close() error                { panic("not used") }

// ANCHOR_END: fake

// ANCHOR: example

func ExamplePublish() {
	f := &fake{data: map[string][]byte{"drafts/42": []byte("  hello  \n")}}

	if err := publish.Publish(f, "42"); err != nil {
		fmt.Println("unexpected:", err)
		return
	}
	fmt.Println(f.calls)
	fmt.Printf("%q\n", f.data["published/42"])
	// Output:
	// [Get drafts/42 Put published/42]
	// "hello"
}

// ANCHOR_END: example
