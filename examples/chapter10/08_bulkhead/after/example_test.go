package bulkhead_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	bulkhead "github.com/holmeslin/learning-go-to-senior/examples/chapter10/08_bulkhead/after"
)

// ANCHOR: example

func Example() {
	search := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "3 results")
	}))
	defer search.Close()
	pricing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "9.99")
	}))
	defer pricing.Close()

	g, err := bulkhead.New(map[string]bulkhead.Downstream{
		"search":  {BaseURL: search.URL, Client: bulkhead.NewClient(4), Limit: 4},
		"pricing": {BaseURL: pricing.URL, Client: bulkhead.NewClient(2), Limit: 2},
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, name := range []string{"search", "pricing"} {
		body, err := g.Get(context.Background(), name, "/")
		fmt.Println(name, body, err)
	}
	// Output:
	// search 3 results <nil>
	// pricing 9.99 <nil>
}

// ANCHOR_END: example
