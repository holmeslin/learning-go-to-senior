// Package quote 同時向多個供應商詢價，用最先回來的那一筆。
package quote

// Source 向一個供應商詢價，回傳報價。
type Source func() string

// ANCHOR: fastest

// Fastest 同時詢問所有供應商，回傳最先回來的報價；sources 不可為空。
func Fastest(sources []Source) string {
	results := make(chan string)
	for _, src := range sources {
		go func() { results <- src() }()
	}
	return <-results
}

// ANCHOR_END: fastest
