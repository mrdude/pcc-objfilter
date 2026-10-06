package objfilter

import "fmt"

type ExampleLogEntry struct {
	Level   string `json:"level"`
	Message string `json:"msg"`
}

func ExampleQuery() {
	obj := NewJsonObject(ExampleLogEntry{
		Level:   "info",
		Message: "Hello Whirled!",
	})

	q, err := ParseQuery(`level = "info"`)
	if err != nil {
		panic(fmt.Errorf("failed to parse query: %w", err))
	}

	fmt.Printf("%t", q.Evaluate(obj))
	// Output: true
}
