package drafts

import (
	"fmt"
	"os"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter5/04_where_test_doubles_live/before/kv"
)

// ANCHOR: example

func Example_staleFake() {
	fmt.Println("fake:", Autosave(newFakeStore(), "Q3/plan", "outline"))

	dir, err := os.MkdirTemp("", "drafts")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer os.RemoveAll(dir)
	fmt.Println("real:", Autosave(kv.NewDirStore(dir), "Q3/plan", "outline"))
	// Output:
	// fake: <nil>
	// real: autosave "Q3/plan": kv: invalid key
}

// ANCHOR_END: example
