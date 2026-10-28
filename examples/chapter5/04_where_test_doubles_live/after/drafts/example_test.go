package drafts_test

import (
	"fmt"
	"os"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter5/04_where_test_doubles_live/after/drafts"
	"github.com/holmeslin/learning-go-to-senior/examples/chapter5/04_where_test_doubles_live/after/kv"
	"github.com/holmeslin/learning-go-to-senior/examples/chapter5/04_where_test_doubles_live/after/kv/kvtest"
)

// ANCHOR: example

func Example_fakeAndDir() {
	fmt.Println("fake:", drafts.Autosave(kvtest.NewFake(), "Q3/plan", "outline"))

	dir, err := os.MkdirTemp("", "drafts")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer os.RemoveAll(dir)
	fmt.Println("real:", drafts.Autosave(kv.NewDirStore(dir), "Q3/plan", "outline"))
	// Output:
	// fake: autosave "Q3/plan": kv: invalid key
	// real: autosave "Q3/plan": kv: invalid key
}

// ANCHOR_END: example
