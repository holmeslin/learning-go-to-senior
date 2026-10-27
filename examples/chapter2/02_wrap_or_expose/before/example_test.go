package profile_test

import (
	"errors"
	"fmt"
	"io/fs"
	"testing/fstest"

	profile "github.com/holmeslin/learning-go-to-senior/examples/chapter2/02_wrap_or_expose/before"
)

// ANCHOR: caller

func ExampleLoad_callerDependsOnFS() {
	_, err := profile.Load(fstest.MapFS{}, "alice")
	fmt.Println(err)
	fmt.Println(errors.Is(err, fs.ErrNotExist))

	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		fmt.Println(pathErr.Op, pathErr.Path)
	}
	// Output:
	// load profile "alice": open alice.json: file does not exist
	// true
	// open alice.json
}

// ANCHOR_END: caller
