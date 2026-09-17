package objfilter

import (
	"testing"
)

func TestTokenStream(t *testing.T) {
	tokCh := startTokenizer("@message AND message = \"test\" OR NOT (test='test' AND facility = 16.3)")
	ts := newTokenStream(tokCh, 1)
	defer ts.Close()

	for {
		tok := ts.Next(1)
		if len(tok) == 0 {
			break
		}
		t.Logf("%s\n", tok[0].String())
	}
}

func TestTokenizer(t *testing.T) {
	type testCase struct {
		FilterString   string
		ExpectedTokens []token
	}

	cases := []testCase{
		{
			// boolean literal: true
			FilterString: "true",
			ExpectedTokens: []token{
				{
					Type: boolLiteralToken,
					Text: "true",
				},
				{
					Type: eofToken,
					Text: "",
				},
			},
		},
		{
			// boolean literal: false
			FilterString: "false",
			ExpectedTokens: []token{
				{
					Type: boolLiteralToken,
					Text: "false",
				},
				{
					Type: eofToken,
					Text: "",
				},
			},
		},
		{
			FilterString: "message = \"test\" OR NOT (test_message='test' AND facility = 16.3)",
			ExpectedTokens: []token{
				{
					Type: identifierToken,
					Text: "message",
				},
				{
					Type: operatorToken,
					Text: "=",
				},
				{
					Type: stringLiteralToken,
					Text: "\"test\"",
				},
				{
					Type: booleanOpToken,
					Text: "OR",
				},
				{
					Type: booleanOpToken,
					Text: "NOT",
				},
				{
					Type: parensToken,
					Text: "(",
				},
				{
					Type: identifierToken,
					Text: "test_message",
				},
				{
					Type: operatorToken,
					Text: "=",
				},
				{
					Type: stringLiteralToken,
					Text: "'test'",
				},
				{
					Type: booleanOpToken,
					Text: "AND",
				},
				{
					Type: identifierToken,
					Text: "facility",
				},
				{
					Type: operatorToken,
					Text: "=",
				},
				{
					Type: intLiteralToken,
					Text: "16.3",
				},

				{
					Type: parensToken,
					Text: ")",
				},
				{
					Type: eofToken,
					Text: "",
				},
			},
		},
	}

	for i := range cases {
		tc := &cases[i]
		t.Run(tc.FilterString, func(t *testing.T) {
			// convert the filter string into a list of tokens
			tokCh := startTokenizer(tc.FilterString)

			var tokens []token
			for tok := range tokCh {
				tokens = append(tokens, tok)
			}

			// validate tokens
			for i, tok := range tokens {
				t.Logf("[%d] %s", i, tok.String())

				if tok.Type != tc.ExpectedTokens[i].Type || tok.Text != tc.ExpectedTokens[i].Text {
					t.Errorf("\tactualToken[%d] = %s doesn't match expectedToken[%d] = %s",
						i, tok.String(), i, tc.ExpectedTokens[i].String())
				}
			}
		})
	}
}

func TestParser(t *testing.T) {
	q := "message = \"test\" OR NOT (test='test' AND facility = 16.3)"

	tokCh := startTokenizer(q)

	ts := newTokenStream(tokCh, 5)
	defer ts.Close()

	p := &parser{}
	err := p.parse(ts)
	if err != nil {
		t.Fatal(err)
	}

	for i, item := range p.stack {
		t.Logf("%d) %s",
			i, item.String())
	}
}
