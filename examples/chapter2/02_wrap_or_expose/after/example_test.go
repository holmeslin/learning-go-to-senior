package profile_test

import (
	"errors"
	"fmt"
	"io/fs"
	"testing/fstest"

	profile "github.com/holmeslin/learning-go-to-senior/examples/chapter2/02_wrap_or_expose/after"
)

// ANCHOR: example

func ExampleLoad_notFound() {
	_, err := profile.Load(fstest.MapFS{}, "alice")
	fmt.Println(err)
	fmt.Println(errors.Is(err, profile.ErrNotFound))
	fmt.Println(errors.Is(err, fs.ErrNotExist))
	// Output:
	// load profile "alice": profile not found
	// true
	// false
}

// ANCHOR_END: example
