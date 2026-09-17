package objfilter

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type expressionType string

const (
	rootExpression        expressionType = "root-expression"
	boolLiteralExpression expressionType = "bool-literal-expression"
	simpleExpression      expressionType = "simple-expression"
	regexMatchExpression  expressionType = "regex-match-expression"
	binaryBoolExpression  expressionType = "binary-boolean-expression"
	notExpression         expressionType = "not-expression"
	parenExpression       expressionType = "paren-expression"
)

func (t expressionType) Type() expressionType { return t }

type exprOrToken struct {
	token *token
	expr  expression
}

func (not *exprOrToken) IsToken() bool { return not.token != nil }
func (not *exprOrToken) IsNode() bool  { return not.expr != nil }
func (not *exprOrToken) MatchToken(tt tokenType) bool {
	return not.IsToken() && not.token.Type == tt
}
func (not *exprOrToken) MatchNode(nt expressionType, other ...expressionType) bool {
	if !not.IsNode() {
		return false
	}

	if not.expr.Type() == nt {
		return true
	}

	for _, t := range other {
		if not.expr.Type() == t {
			return true
		}
	}

	return false
}
func (not *exprOrToken) String() string {
	if not.token != nil {
		return fmt.Sprintf("Token: %s", not.token.String())
	} else if not.expr != nil {
		return fmt.Sprintf("expression: %+v", not.expr)
	} else {
		return "Unknown"
	}
}

type parser struct {
	stack []exprOrToken
}

func (pq *parser) parse(ts *tokenStream) error {
	eof := false
	for {
		// shift
		if !eof {
			tok := ts.Next(1)
			if len(tok) == 0 || tok[0].Type == eofToken {
				eof = true
			} else {
				pq.stack = append(pq.stack,
					exprOrToken{token: &tok[0]})
			}
		}

		// reduce
		reductions := 0
		for {
			n, err := pq.reduce()
			if err != nil {
				return err
			}
			if n == nil {
				break
			}
			pq.stack = append(pq.stack,
				exprOrToken{expr: n})
			reductions++
		}
		if reductions == 0 && eof {
			break
		}
	}

	// if the stack hasn't been reduced to a single expression, return an error
	if len(pq.stack) != 1 || pq.stack[0].expr == nil {
		return errors.New("failed to reduce")
	}

	return nil
}

func (pq *parser) reduce() (expr expression, err error) {
	// boolean literal
	if len(pq.stack) >= 1 {
		top := pq.peekStack(1)

		if top[0].MatchToken(boolLiteralToken) {
			pq.popStack(1)

			val := strings.ToLower(top[0].token.Text) == "true"
			expr = newBoolLiteralExpression(val)
			return
		}
	}

	// simple expression: <identifier> <operator> <identifier>
	if len(pq.stack) >= 3 {
		top := pq.peekStack(3)

		if top[0].MatchToken(identifierToken) && top[1].MatchToken(operatorToken) && (top[2].MatchToken(stringLiteralToken) || top[2].MatchToken(intLiteralToken)) {
			pq.popStack(3)

			op := comparisonOp(top[1].token.Text).Canonicalize()

			if op == regexMatchOp {
				expr, err = newRegexMatchExpression(
					top[0].token.Text,
					top[1].token.Text,
					valueFromToken(*top[2].token),
					[]token{*top[0].token, *top[1].token, *top[2].token},
				)
			} else {
				expr, err = newSimpleExpression(
					top[0].token.Text,
					top[1].token.Text,
					valueFromToken(*top[2].token),
					[]token{*top[0].token, *top[1].token, *top[2].token},
				)
			}
			return
		}
	}

	//// simple expression: <identifier>
	//if len(pq.stack) >= 1 {
	//	top := pq.peekStack(1)
	//
	//	if top[0].MatchToken(identifierToken) {
	//		pq.popStack(1)
	//		expr = &parseNode{
	//			Type:   simpleExpression,
	//			Tokens: []token{*top[0].token},
	//		}
	//		return
	//	}
	//}

	// boolean expression: <simple-expression> AND/OR <simple-expression>
	if len(pq.stack) >= 3 {
		top := pq.peekStack(3)

		if top[0].MatchNode(boolLiteralExpression, simpleExpression, regexMatchExpression, parenExpression, notExpression) && (top[1].MatchToken(booleanOpToken) && strings.ToLower(top[1].token.Text) != "not") && top[2].MatchNode(boolLiteralExpression, simpleExpression, regexMatchExpression, parenExpression, notExpression) {
			pq.popStack(3)
			expr, err = newBinaryBoolExpression(
				[]expression{top[0].expr, top[2].expr},
				top[1].token.Text,
				[]token{*top[1].token},
			)
			return
		}
	}

	// boolean expression extension: <boolean-expression> AND/OR <simple-expression>
	if len(pq.stack) >= 3 {
		top := pq.peekStack(3)

		if top[0].MatchNode(binaryBoolExpression) && (top[1].MatchToken(booleanOpToken) && top[0].expr.(*binaryBoolExpr).Operator.Canonicalize() == binaryBoolOp(top[1].token.Text).Canonicalize()) && top[2].MatchNode(boolLiteralExpression, simpleExpression, regexMatchExpression, parenExpression, notExpression) {
			pq.popStack(3)

			boolExpr := top[0].expr.(*binaryBoolExpr)
			newOperand := top[2].expr

			boolExpr.Operands = append(boolExpr.Operands, newOperand)

			return boolExpr, nil
		}
	}

	// not expression: NOT <expression>
	if len(pq.stack) >= 2 {
		top := pq.peekStack(2)

		if (top[0].MatchToken(booleanOpToken) && strings.ToLower(top[0].token.Text) == "not") && top[1].MatchNode(simpleExpression, parenExpression, binaryBoolExpression) {
			pq.popStack(2)
			expr, err = newNotExpression(top[1].expr, []token{*top[0].token})
			return
		}
	}

	// parenthetical expression: ( <expression> )
	if len(pq.stack) >= 3 {
		top := pq.peekStack(3)

		if (top[0].MatchToken(parensToken) && top[0].token.Text == "(") && top[1].MatchNode(boolLiteralExpression, simpleExpression, regexMatchExpression, binaryBoolExpression, parenExpression, notExpression) && (top[2].MatchToken(parensToken) && top[2].token.Text == ")") {
			pq.popStack(3)
			expr, err = newParenExpression(top[1].expr, []token{*top[0].token, *top[2].token})
			return
		}
	}

	return
}

func valueFromToken(tok token) Value {
	switch tok.Type {
	case stringLiteralToken:
		// strip quotes
		unquotedValue := tok.Text
		if strings.HasPrefix(unquotedValue, "\"") && strings.HasSuffix(unquotedValue, "\"") {
			unquotedValue = strings.TrimPrefix(unquotedValue, "\"")
			unquotedValue = strings.TrimSuffix(unquotedValue, "\"")
		}

		if strings.HasPrefix(unquotedValue, "'") && strings.HasSuffix(unquotedValue, "'") {
			unquotedValue = strings.TrimPrefix(unquotedValue, "'")
			unquotedValue = strings.TrimSuffix(unquotedValue, "'")
		}

		return Value{V: unquotedValue, Ok: true}
	case intLiteralToken:
		i, err := strconv.ParseInt(tok.Text, 10, 64)
		if err != nil {
			return Value{Ok: false}
		}

		return Value{V: i, Ok: true}
	default:
		return Value{Ok: false}
	}
}

// stack functions
func (pq *parser) peekStack(n int) []exprOrToken {
	if n > len(pq.stack) {
		n = len(pq.stack)
	}
	return pq.stack[len(pq.stack)-n : len(pq.stack)]
}

func (pq *parser) popStack(n int) []exprOrToken {
	top := pq.peekStack(n)
	pq.stack = pq.stack[0 : len(pq.stack)-len(top)]
	return top
}

// parseError
type parseError struct {
	tok token
	err error
}

func (err *parseError) Error() string {
	return fmt.Sprintf("Error @ [%s]: %s", err.tok.String(), err.err)
}

func (err *parseError) Unwrap() error {
	return err.err
}
