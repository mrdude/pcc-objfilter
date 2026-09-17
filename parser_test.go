package objfilter

import "testing"

func TestParseQuery(t *testing.T) {
	pq, err := ParseQuery(`true AND field = 42 AND ((msg ~= "message") AND (NOT field2 == 42))
`)
	if err != nil {
		t.Fatal(err)
	}

	// TODO check root node and children expressions
	t.Logf("%s", pq.PrettyString())
}
