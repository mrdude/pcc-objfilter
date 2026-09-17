package objfilter

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

type tokenizer struct {
	filterString string // the string being parsed
	start, pos   int
	width        int // the width of the current rune
	tokenCh      chan token
}

func (t *tokenizer) emitToken(typ tokenType) {
	tok := token{
		Type: typ,
		From: t.start,
		To:   t.pos,
		Text: "",
	}
	if typ != eofToken && typ != errorToken {
		tok.Text = t.filterString[t.start:t.pos]
	}

	t.tokenCh <- tok

	t.start = t.pos
}

func (t *tokenizer) emitError(msg string) {
	tok := token{
		Type: errorToken,
		From: t.start,
		To:   t.pos,
		Text: msg,
	}

	t.tokenCh <- tok
}

// advance the tokenizer
func (t *tokenizer) next() (ch rune, ok bool) {
	if t.pos >= len(t.filterString) {
		ok = false
		return
	}
	ch, t.width = utf8.DecodeRuneInString(t.filterString[t.pos:])
	t.pos += t.width
	ok = true
	return ch, true
}

type tokenizerFn func(*tokenizer) tokenizerFn

type tokenType string

const (
	identifierToken    tokenType = "identifier"
	operatorToken      tokenType = "operator"
	booleanOpToken     tokenType = "boolean-op"
	parensToken        tokenType = "parens"
	stringLiteralToken tokenType = "string-literal"
	intLiteralToken    tokenType = "int-literal"
	boolLiteralToken   tokenType = "bool-literal"
	eofToken           tokenType = "eof"
	errorToken         tokenType = "error"
)

type token struct {
	Type     tokenType
	From, To int // token indices

	Text string // the token text
}

func (tok *token) String() string {
	return fmt.Sprintf("%s[%d:%d]: %s", tok.Type, tok.From, tok.To, tok.Text)
}

func startTokenizer(filterString string) <-chan token {
	l := &tokenizer{
		filterString: filterString,
		tokenCh:      make(chan token),
	}

	// main loop
	go func(filterString string, tokenCh chan<- token) {
		defer close(tokenCh)

		for state := tokenizerBase; state != nil; {
			state = state(l)
		}

		// emit eof
		l.emitToken(eofToken)
	}(filterString, l.tokenCh)

	return l.tokenCh
}

// tokenizer state functions
func tokenizerBase(l *tokenizer) tokenizerFn {
	// advance the tokenizer
	ch, ok := l.next()
	if !ok {
		return nil
	}

	switch {
	case ch == rune('"') || ch == rune('\''):
		// emit a token for previous text
		// TODO this is not always an identifier -- handle that case
		l.pos -= l.width
		if l.start != l.pos {
			l.emitToken(identifierToken)
		}

		// consume the quoted string
		l.pos += l.width

		if ch == rune('"') {
			return tokenizerDoubleQuotes
		} else {
			return tokenizerSingleQuotes
		}
	case isWhitespace(ch):
		// emit a token for previous text
		l.pos -= l.width
		if l.start != l.pos {
			l.emitToken(identifierToken)
		}

		// consume whitespace
		return tokenizerWhitespace
	case ch == '(' || ch == ')':
		// emit a token for previous text
		l.pos -= l.width
		if l.start != l.pos {
			l.emitToken(identifierToken)
		}

		// consume the paren
		l.pos += l.width
		l.emitToken(parensToken)

		return tokenizerBase
	case ch == '>' || ch == '<' || ch == '=' || ch == '!' || ch == '~':
		// emit a token for previous text
		l.pos -= l.width
		if l.start != l.pos {
			l.emitToken(identifierToken)
		}

		// tokenize operator
		return tokenizerOperator
	case (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '@' || ch == '_':
		return tokenizerIdentifier
	case ch >= '0' && ch <= '9':
		return tokenizerInteger
	default:
		l.emitError("Unknown token")
		return nil
	}
}

func tokenizerIdentifier(l *tokenizer) tokenizerFn {
	// advance the tokenizer
	ch, ok := l.next()

	if ok {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '@' || ch == '_' {
			return tokenizerIdentifier
		}
	}

	// emit a token for previous text
	if ok {
		l.pos -= l.width
	}

	if l.start != l.pos {
		tokText := strings.ToLower(l.filterString[l.start:l.pos])

		if tokText == "not" || tokText == "and" || tokText == "or" {
			// handle boolean operators
			l.emitToken(booleanOpToken)
		} else if tokText == "true" || tokText == "false" {
			// handle boolean literals
			l.emitToken(boolLiteralToken)
		} else {
			l.emitToken(identifierToken)
		}
	}

	if !ok {
		// EOF
		return nil
	}

	return tokenizerBase
}

func tokenizerWhitespace(l *tokenizer) tokenizerFn {
	// advance the tokenizer
	ch, ok := l.next()
	if !ok {
		return nil
	}

	if isWhitespace(ch) {
		// Discard the whitespace rune
		l.start = l.pos
		return tokenizerWhitespace
	} else {
		// discard everything up to the current rune
		l.start = l.pos

		// rollback
		l.start -= l.width
		l.pos -= l.width

		// continue parsing
		return tokenizerBase
	}
}

func tokenizerOperator(l *tokenizer) tokenizerFn {
	// advance the tokenizer
	ch, ok := l.next()
	if !ok {
		return nil
	}

	switch {
	case ch == '>' || ch == '<' || ch == '=' || ch == '!' || ch == '~':
		return tokenizerOperator
	default:
		// emit a token for the previous text
		l.pos -= l.width
		l.emitToken(operatorToken)

		// consume the remaining tokens
		return tokenizerBase
	}
}

func tokenizerDoubleQuotes(l *tokenizer) tokenizerFn {
	// advance the tokenizer
	ch, ok := l.next()
	if !ok {
		return nil
	}

	if ch == rune('"') {
		// emit a token
		l.emitToken(stringLiteralToken)

		// return to base
		return tokenizerBase
	}

	return tokenizerDoubleQuotes
}

func tokenizerSingleQuotes(l *tokenizer) tokenizerFn {
	// advance the tokenizer
	ch, ok := l.next()
	if !ok {
		return nil
	}

	if ch == rune('\'') {
		// emit a token
		l.emitToken(stringLiteralToken)

		// return to base
		return tokenizerBase
	}

	return tokenizerSingleQuotes
}

func tokenizerInteger(l *tokenizer) tokenizerFn {
	// advance the tokenizer
	ch, ok := l.next()
	if !ok {
		return nil
	}

	switch {
	case (ch >= '0' && ch <= '9') || ch == '.':
		return tokenizerInteger
	default:
		// emit a token for the previous text
		l.pos -= l.width
		l.emitToken(intLiteralToken)

		// consume the remaining tokens
		return tokenizerBase
	}
}

// helper functions
func isWhitespace(ch rune) bool {
	return ch == rune(' ') || ch == rune('\t') || ch == rune('\n')
}

/*

unit = "sshd.service" OR facility = 16

*/

/*
Token Types:

identifier (/[a-zA-Z]/)
operator (=, <=, >=, ==, <, >, !=, ~=, OR, AND, NOT)
parens ()
string-literal
integer-literal
*/
