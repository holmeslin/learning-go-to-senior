package publish_test

import (
	"fmt"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter3/01_define_interface_at_consumer/after/blob"
	"github.com/holmeslin/learning-go-to-senior/examples/chapter3/01_define_interface_at_consumer/after/publish"
)

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

// ANCHOR: bucket

// 編譯期確認 *blob.Bucket 滿足 publish.Blobs。blob 不認得 publish，檢查只能放在接線的這一側。
var _ publish.Blobs = (*blob.Bucket)(nil)

func ExamplePublish_bucket() {
	b := blob.NewBucket()
	if err := b.Put("drafts/42", []byte("  hello  \n")); err != nil {
		fmt.Println("unexpected:", err)
		return
	}

	if err := publish.Publish(b, "42"); err != nil {
		fmt.Println("unexpected:", err)
		return
	}

	out, err := b.Get("published/42")
	if err != nil {
		fmt.Println("unexpected:", err)
		return
	}
	fmt.Printf("%q\n", out)
	// Output: "hello"
}

// ANCHOR_END: bucket
