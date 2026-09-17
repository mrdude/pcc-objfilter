package objfilter

import (
	"fmt"
	"testing"
)

func TestExpressionSimplification(t *testing.T) {
	type testCase struct {
		Query string // the input query

		ExpectedPrettyString     string // the expected pretty version of the query
		ExpectedSimplifiedString string // the expected simplified version of the query
	}

	cases := []testCase{
		{
			Query:                    `(@msg = "HELLO WHIRLED!" OR @msg = "Hello Whirled!") and (not @msg = "Hi")`,
			ExpectedPrettyString:     `(@msg = "HELLO WHIRLED!" or @msg = "Hello Whirled!") and (not @msg = "Hi")`,
			ExpectedSimplifiedString: `(@msg = "HELLO WHIRLED!" or @msg = "Hello Whirled!") and (not @msg = "Hi")`,
		},
		{
			// parenthesis de-nesting
			Query:                    `((((NOT (@msg = "Hi")))))`,
			ExpectedPrettyString:     `((((not (@msg = "Hi")))))`,
			ExpectedSimplifiedString: `not (@msg = "Hi")`,
		},
		{
			Query:                    `(not (NOT (not (NOT (not @msg = "Hi")))))`,
			ExpectedPrettyString:     `(not (not (not (not (not @msg = "Hi")))))`,
			ExpectedSimplifiedString: `not @msg = "Hi"`,
		},
		{
			// binary boolean operator merging
			Query:                    `(@msg = "Hi" OR @msg = "Bye") OR (@msg = "Who" OR @unit = 'TestUnit')`,
			ExpectedPrettyString:     `(@msg = "Hi" or @msg = "Bye") or (@msg = "Who" or @unit = "TestUnit")`,
			ExpectedSimplifiedString: `@msg = "Hi" or @msg = "Bye" or @msg = "Who" or @unit = "TestUnit"`,
		},
		{
			// boolean literals: true
			Query:                    `true`,
			ExpectedPrettyString:     `true`,
			ExpectedSimplifiedString: `true`,
		},
		{
			// boolean literals: false
			Query:                    `false`,
			ExpectedPrettyString:     `false`,
			ExpectedSimplifiedString: `false`,
		},
		{
			// "true or" expressions
			Query:                    `(true OR @unit = 'TestUnit')`,
			ExpectedPrettyString:     `(true or @unit = "TestUnit")`,
			ExpectedSimplifiedString: `true`,
		},
		{
			// "false and" expressions
			Query:                    `(false AND @unit = 'TestUnit')`,
			ExpectedPrettyString:     `(false and @unit = "TestUnit")`,
			ExpectedSimplifiedString: `false`,
		},
		{
			// superfluous boolean operators: AND
			Query:                    `(true AND @unit = 'TestUnit' AND true)`,
			ExpectedPrettyString:     `(true and @unit = "TestUnit" and true)`,
			ExpectedSimplifiedString: `@unit = "TestUnit"`,
		},
		{
			// superfluous boolean operators: OR
			Query:                    `(false OR @unit = 'TestUnit' OR false)`,
			ExpectedPrettyString:     `(false or @unit = "TestUnit" or false)`,
			ExpectedSimplifiedString: `@unit = "TestUnit"`,
		},
	}

	for i := range cases {
		tc := &cases[i]
		name := fmt.Sprintf("Case%d", i)
		t.Run(name, func(t *testing.T) {
			pq, err := ParseQuery(tc.Query)
			if err != nil {
				t.Fatalf("Failed to parse query: %s", err)
			}

			actualPretty := pq.root.PrettyString()
			pq.Simplify() // simplify the query
			actualSimple := pq.root.PrettyString()

			t.Logf("Prettifed query: %s", actualPretty)
			if actualPretty != tc.ExpectedPrettyString {
				t.Errorf("\tPrettyString doesn't match expected value; expected: %s", tc.ExpectedPrettyString)
			}

			t.Logf("Simplified query: %s", actualSimple)
			if actualSimple != tc.ExpectedSimplifiedString {
				t.Errorf("\tSimpleString doesn't match expected value; expected: %s", tc.ExpectedSimplifiedString)
			}

			if actualPretty != tc.ExpectedPrettyString || actualSimple != tc.ExpectedSimplifiedString {
				t.Fatal()
			}
		})
	}
}
