package objfilter

import (
	"errors"
)

type Query struct {
	root *rootExpr
}

func ParseQuery(q string) (pq *Query, err error) {
	tokCh := startTokenizer(q)

	ts := newTokenStream(tokCh, 5)
	defer ts.Close()

	p := &parser{}
	err = p.parse(ts)
	if err == nil {
		pq = &Query{root: newRootExpression(p.stack[0].expr)}

		// TODO simplify ParsedQuery
	}

	return
}

// Evaluate returns true if the Query matches the object
func (pq *Query) Evaluate(obj Object) bool {
	obj = newCachingObject(obj)
	return evaluate(pq.root, obj)
}

// PrettyString returns a pretty-printed version
// of this Query
func (pq *Query) PrettyString() string {
	return pq.root.PrettyString()
}

// Simplify attempts to simplify this query
func (pq *Query) Simplify() {
	for i := 0; ; i++ {
		replace, changesMade := pq.root.Simplify()
		if replace != nil {
			panic(errors.New("cannot replace rootExpr"))
		} else if changesMade {
			// continue looping
		} else /* !changesMade */ {
			return
		}

		// avoid infinite loops
		if i > 1000 {
			panic(errors.New("simplify loop detected"))
		}
	}
}
