package objfilter

import (
	"testing"
)

type LogEntry struct {
	RawMessage string `json:"msg"`
	Level      string `json:"level"`
}

func mustExpr(expr expression, err error) expression {
	if err != nil {
		panic(err)
	}
	return expr
}

func TestSimpleExpressionEval(t *testing.T) {
	pq := &Query{
		root: newRootExpression(
			mustExpr(newSimpleExpression(
				newFieldOperand("msg"),
				"==",
				newLiteralOperand(Value{V: "Hello Whirled!", Ok: true}),
				nil,
			)),
		),
	}
	t.Logf("Prettified query: %s", pq.root.PrettyString())

	entry := LogEntry{
		RawMessage: "Hello Whirled!",
	}

	v := pq.Evaluate(NewJsonObject(&entry))
	t.Logf("v = %+v\n", v)
}

func TestSimpleExpressionEval2(t *testing.T) {
	pq, err := ParseQuery(`
	(msg = "HELLO WHIRLED!" OR msg = "Hello Whirled!") and (not msg = "Hi")
`)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Prettified query: %s", pq.root.PrettyString())

	entry := LogEntry{
		RawMessage: "Hello Whirled!",
	}

	v := pq.Evaluate(NewJsonObject(&entry))
	if !v {
		t.Fatalf("Expected v = true, got v = %t", v)
	}
}

func TestSimpleExpressionEval3(t *testing.T) {
	pq, err := ParseQuery(`
	msg == level
`)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Prettified query: %s", pq.root.PrettyString())

	entry := LogEntry{
		RawMessage: "info",
		Level:      "info",
	}

	v := pq.Evaluate(NewJsonObject(&entry))
	if !v {
		t.Fatalf("Expected v = true, got v = %t", v)
	}
}

func TestSimpleExpressionEval4(t *testing.T) {
	pq, err := ParseQuery(`
	msg != level
`)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Prettified query: %s", pq.root.PrettyString())

	entry := LogEntry{
		RawMessage: "Hello Whirled!",
		Level:      "info",
	}

	v := pq.Evaluate(NewJsonObject(&entry))
	if !v {
		t.Fatalf("Expected v = true, got v = %t", v)
	}
}

func BenchmarkExpressionEval(b *testing.B) {
	pq, err := ParseQuery(`
	(msg = "HELLO WHIRLED!")
`)
	if err != nil {
		b.Fatal(err)
	}
	b.Logf("Prettified query: %s", pq.root.PrettyString())

	entries := []*LogEntry{
		{
			RawMessage: "Hello Whirled!",
		},
		{
			RawMessage: "Goodbye!",
		},
	}

	b.ResetTimer()
	for b.Loop() {
		for i := range entries {
			_ = pq.Evaluate(NewJsonObject(entries[i]))
		}
	}
}

func BenchmarkJsonConvert(b *testing.B) {
	var entry = LogEntry{
		RawMessage: "Hello Whirled!",
	}

	var m = make(map[string]any)

	b.ResetTimer()
	for b.Loop() {
		err := jsonConvertToMap(&entry, m)
		if err != nil {
			b.Fatal(err)
		}
	}
}
