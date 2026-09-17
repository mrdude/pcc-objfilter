package objfilter

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

var _ expression = &rootExpr{}
var _ expression = &boolLiteralExpr{}
var _ expression = &simpleExpr{}
var _ expression = &regexMatchExpr{}
var _ expression = &binaryBoolExpr{}
var _ expression = &notExpr{}
var _ expression = &parenExpr{}

// expression is a logquery expression
type expression interface {
	Type() expressionType

	// PrettyString returns a pretty-printed version
	// of the original logquery
	PrettyString() string

	// Simplify attempts to simplify this expression.
	//
	// returns changesMade=true if changes were made to this expresssion's internal values
	// or children, or changesMade=false if not.
	//
	// if replace is non-nil, this node (the one that Simplify was called on) will be replaced
	// by the given expression.
	Simplify() (replace expression, changesMade bool)
}

type rootExpr struct {
	expressionType
	SubExpr expression
}

func newRootExpression(subExpr expression) *rootExpr {
	// validation
	if subExpr == nil {
		panic(errors.New("nil subExpr"))
	}

	return &rootExpr{
		expressionType: rootExpression,
		SubExpr:        subExpr,
	}
}

func (e *rootExpr) PrettyString() string {
	return e.SubExpr.PrettyString()
}

func (e *rootExpr) Simplify() (replace expression, changesMade bool) {
	// recursively simplify our children
	replace, changesMade = e.SubExpr.Simplify()
	if replace != nil {
		e.SubExpr = replace
		return nil, true
	} else if changesMade {
		return nil, true
	} else /* !changesMade */ {
		// Our children did not apply simplification.
		// Continue to apply simplification on this node.
	}

	// Case 1: top-level parens removal
	// if this root expression is just "(expr)", remove the parens
	if e.SubExpr.Type() == parenExpression {
		e.SubExpr = e.SubExpr.(*parenExpr).SubExpr
		changesMade = true
		return
	}

	// TODO implement other cases

	return
}

type boolLiteralExpr struct {
	expressionType
	Value bool
}

func newBoolLiteralExpression(val bool) *boolLiteralExpr {
	return &boolLiteralExpr{
		expressionType: boolLiteralExpression,
		Value:          val,
	}
}

func (e *boolLiteralExpr) PrettyString() string {
	if e.Value {
		return "true"
	} else {
		return "false"
	}
}

func (e *boolLiteralExpr) Simplify() (replace expression, changesMade bool) {
	return
}

type comparisonOp string

const (
	equalsOp     comparisonOp = "="
	altEqualsOp  comparisonOp = "=="
	notEqualsOp  comparisonOp = "!="
	regexMatchOp comparisonOp = "~="
)

func (op comparisonOp) IsValid() bool {
	switch op.Canonicalize() {
	case equalsOp:
		fallthrough
	case altEqualsOp:
		fallthrough
	case notEqualsOp:
		fallthrough
	case regexMatchOp:
		return true
	default:
		return false
	}
}

func (op comparisonOp) Canonicalize() comparisonOp {
	// comparisonOp's only have symbols, so there is nothing to canonicalize
	return op
}

type simpleExpr struct {
	expressionType
	IdentifierOperand string // the identifier
	Operator          comparisonOp
	ValueOperand      Value

	tokens []token
}

func newSimpleExpression(identifier string, operator string, valueOperand Value, tokens []token) (*simpleExpr, error) {
	// validation
	op := comparisonOp(operator)
	if !op.IsValid() {
		return nil, &parseError{tok: tokens[1], err: fmt.Errorf("invalid operator: %s", operator)}
	}

	if op == regexMatchOp {
		panic(errors.New("newRegexMatchExpression should be used for regexes"))
	}

	return &simpleExpr{
		expressionType:    simpleExpression,
		IdentifierOperand: identifier,
		Operator:          op,
		ValueOperand:      valueOperand,

		tokens: tokens,
	}, nil
}

func (e *simpleExpr) PrettyString() string {
	var valueOperand string
	switch val := e.ValueOperand.V; val.(type) {
	case string:
		valueOperand = quoteStringLiteral(val.(string))
	case int:
		valueOperand = strconv.FormatInt(int64(val.(int)), 10)
	case int8:
		valueOperand = strconv.FormatInt(int64(val.(int8)), 10)
	case int16:
		valueOperand = strconv.FormatInt(int64(val.(int16)), 10)
	case int32:
		valueOperand = strconv.FormatInt(int64(val.(int32)), 10)
	case int64:
		valueOperand = strconv.FormatInt(int64(val.(int64)), 10)
	default:
		panic(fmt.Errorf("unexpected value type: %s", e.ValueOperand.V))
	}

	return fmt.Sprintf("%s %s %s", e.IdentifierOperand, e.Operator.Canonicalize(), valueOperand)
}

func (e *simpleExpr) Simplify() (replace expression, changesMade bool) {
	// TODO implement me
	return
}

func quoteStringLiteral(s string) string {
	containsSingleQuote := strings.Contains(s, "'")
	containsDoubleQuote := strings.Contains(s, "\"")

	switch {
	case !containsSingleQuote && !containsDoubleQuote:
		// use double quotes
		return "\"" + s + "\""
	case containsSingleQuote && containsDoubleQuote:
		// use double quotes, apply double-quote escaping
		return "\"" + strings.ReplaceAll(s, "\"", "\\\"") + "\""
	case containsSingleQuote && !containsDoubleQuote:
		// use double quotes
		return "\"" + s + "\""
	case !containsSingleQuote && containsDoubleQuote:
		// use single quotes
		return "'" + s + "'"
	default:
		panic(errors.New("this should never happen"))
	}
}

type regexMatchExpr struct {
	expressionType
	IdentifierOperand string         // the identifier
	Re                *regexp.Regexp // the parsed regex

	tokens []token
}

func newRegexMatchExpression(identifier string, operator string, valueOperand Value, tokens []token) (*regexMatchExpr, error) {
	// validation
	op := comparisonOp(operator)
	if !op.IsValid() {
		return nil, &parseError{tok: tokens[1], err: fmt.Errorf("invalid operator: %s", operator)}
	}

	if op != regexMatchOp {
		panic(errors.New("newSimpleExpression should be used for non-regexes"))
	}

	if !valueOperand.Ok {
		panic(errors.New("there should always be a value"))
	}

	regexStr, ok := valueOperand.V.(string)
	if !ok {
		return nil, &parseError{tok: tokens[2], err: fmt.Errorf("regex expressions must be strings")}
	}

	re, err := regexp.Compile(regexStr)
	if err != nil {
		return nil, &parseError{tok: tokens[2], err: fmt.Errorf("invalid regex: %s", err)}
	}

	return &regexMatchExpr{
		expressionType:    regexMatchExpression,
		IdentifierOperand: identifier,
		Re:                re,

		tokens: tokens,
	}, nil
}

func (e *regexMatchExpr) PrettyString() string {
	return fmt.Sprintf("%s %s %s", e.IdentifierOperand, string(regexMatchOp), quoteStringLiteral(e.Re.String())) // TODO use quoteRegex instead
}

func (e *regexMatchExpr) Simplify() (replace expression, changesMade bool) {
	// TODO implement me

	// Case 1: simple regex
	// if e.Re is a "simple" regex (meaning it is just matching a flat set of characters),
	// then replace the expression with a CONTAINS expression
	//
	// for example, "@msg ~= 'hi'" will get replaced with "@msg CONTAINS 'hi'"
	// TODO add contains operator
	return
}

type binaryBoolOp string

const (
	andOp binaryBoolOp = "and"
	orOp  binaryBoolOp = "or"
)

func (op binaryBoolOp) IsValid() bool {
	switch op.Canonicalize() {
	case andOp:
		fallthrough
	case orOp:
		return true
	default:
		return false
	}
}

func (op binaryBoolOp) Canonicalize() binaryBoolOp {
	switch op {
	case andOp:
		fallthrough
	case orOp:
		return op
	default:
		return binaryBoolOp(strings.ToLower(string(op)))
	}
}

type binaryBoolExpr struct {
	expressionType
	Operands []expression
	Operator binaryBoolOp

	tokens []token
}

func newBinaryBoolExpression(operands []expression, operator string, tokens []token) (*binaryBoolExpr, error) {
	// validation
	op := binaryBoolOp(operator)
	if !op.IsValid() {
		return nil, &parseError{tok: tokens[0], err: fmt.Errorf("invalid binary bool operator: %s", operator)}
	}

	return &binaryBoolExpr{
		expressionType: binaryBoolExpression,
		Operands:       operands,
		Operator:       op,
		tokens:         tokens,
	}, nil
}

func (e *binaryBoolExpr) PrettyString() string {
	var (
		elems []string
		sep   = fmt.Sprintf(" %s ", string(e.Operator.Canonicalize()))
	)

	for _, elem := range e.Operands {
		elems = append(elems, elem.PrettyString())
	}

	return strings.Join(elems, sep)
}

func (e *binaryBoolExpr) Simplify() (replace expression, changesMade bool) {
	// recursively simplify our children
	for i := range e.Operands {
		replace, changesMade = e.Operands[i].Simplify()
		if replace != nil {
			e.Operands[i] = replace
			return nil, true
		} else if changesMade {
			return nil, true
		} else /* !changesMade */ {
			// Our children did not apply simplification.
			// Continue to apply simplification on this node.
		}
	}

	// Case 1: "true or" and "false and" expressions
	//
	// if this expression is of the form "(true OR ...)",
	// simplify it to "true"
	//
	// if this expression is of the form "(false AND ...)",
	// simplify it to "false"
	isTrueLiteral := func(op expression) bool {
		return op.Type() == boolLiteralExpression && op.(*boolLiteralExpr).Value == true
	}

	isFalseLiteral := func(op expression) bool {
		return op.Type() == boolLiteralExpression && op.(*boolLiteralExpr).Value == false
	}

	if e.Operator.Canonicalize() == orOp && e.anyOperandMatches(isTrueLiteral) {
		replace = newBoolLiteralExpression(true)
		return
	}

	if e.Operator.Canonicalize() == andOp && e.anyOperandMatches(isFalseLiteral) {
		replace = newBoolLiteralExpression(false)
		return
	}

	// Case 2: superfluous binary literals
	//
	// if this expression is of the form "(true AND ...)", the "true"
	// is redundant and can be removed.
	//
	// if this expression is of the form "(false OR ...)", the "false"
	// is redundant and can be removed.
	switch e.Operator.Canonicalize() {
	case orOp:
		for i := 0; i < len(e.Operands); {
			if isFalseLiteral(e.Operands[i]) {
				// remove this literal
				e.Operands = slices.Delete(e.Operands, i, i+1)
				changesMade = true
			} else {
				i++
			}
		}
	case andOp:
		for i := 0; i < len(e.Operands); {
			if isTrueLiteral(e.Operands[i]) {
				// remove this literal
				e.Operands = slices.Delete(e.Operands, i, i+1)
				changesMade = true
			} else {
				i++
			}
		}
	}
	if changesMade {
		return
	}

	// Case 3: mergeable binary expressions
	// if this expression is of the form "(expr OP expr) OP (expr OP expr)",
	// simplify to "(expr OP expr OP expr OP expr)"
	allSubExprAreBinaryBool := func() bool {
		// are all operands of the form "(expr OP expr)"?
		for _, op := range e.Operands {
			if op.Type() != parenExpression {
				return false
			}

			if op.(*parenExpr).SubExpr.Type() != binaryBoolExpression {
				return false
			}

			boolExpr := op.(*parenExpr).SubExpr.(*binaryBoolExpr)
			if boolExpr.Operator.Canonicalize() != e.Operator.Canonicalize() {
				return false
			}
		}
		return true
	}()
	if allSubExprAreBinaryBool {
		var newOperands []expression
		for _, op := range e.Operands {
			subExpr := op.(*parenExpr).SubExpr.(*binaryBoolExpr)
			newOperands = append(newOperands, subExpr.Operands...)
		}
		e.Operands = newOperands
		changesMade = true
		return
	}

	// TODO implement more cases

	return
}

func (e *binaryBoolExpr) anyOperandMatches(pred func(expression) bool) bool {
	return slices.ContainsFunc(e.Operands, pred)
}

type notExpr struct {
	expressionType
	SubExpr expression

	tokens []token
}

func newNotExpression(subExpr expression, tokens []token) (*notExpr, error) {
	// validation
	if subExpr == nil {
		panic(errors.New("nil subExpr"))
	}

	return &notExpr{
		expressionType: notExpression,
		SubExpr:        subExpr,
		tokens:         tokens,
	}, nil
}

func (e *notExpr) PrettyString() string {
	return "not " + e.SubExpr.PrettyString()
}

func (e *notExpr) Simplify() (replace expression, changesMade bool) {
	// recursively simplify our children
	replace, changesMade = e.SubExpr.Simplify()
	if replace != nil {
		e.SubExpr = replace
		return nil, true
	} else if changesMade {
		return nil, true
	} else /* !changesMade */ {
		// Our children did not apply simplification.
		// Continue to apply simplification on this node.
	}

	// Case 1: double-negatives
	// if this expression is of the form "NOT NOT expr" or "NOT (NOT expr)",
	// simplify it to "expr"
	if e.SubExpr.Type() == notExpression || (e.SubExpr.Type() == parenExpression && e.SubExpr.(*parenExpr).SubExpr.Type() == notExpression) {
		replace = e.SubExpr.(*parenExpr).SubExpr
		return
	}

	// TODO implement more cases

	return
}

type parenExpr struct {
	expressionType
	SubExpr expression

	tokens []token
}

func newParenExpression(subExpr expression, tokens []token) (*parenExpr, error) {
	// validation
	if subExpr == nil {
		panic(errors.New("nil subExpr"))
	}

	return &parenExpr{
		expressionType: parenExpression,
		SubExpr:        subExpr,
		tokens:         tokens,
	}, nil
}

func (e *parenExpr) PrettyString() string {
	return "(" + e.SubExpr.PrettyString() + ")"
}

func (e *parenExpr) Simplify() (replace expression, changesMade bool) {
	// recursively simplify our children
	replace, changesMade = e.SubExpr.Simplify()
	if replace != nil {
		e.SubExpr = replace
		return nil, true
	} else if changesMade {
		return nil, true
	} else /* !changesMade */ {
		// Our children did not apply simplification.
		// Continue to apply simplification on this node.
	}

	// Case 1: denest nested parens
	// if this parenExpr is of the form "((expr))",
	// simplify it to "(expr)"
	if e.SubExpr.Type() == parenExpression {
		e.SubExpr = e.SubExpr.(*parenExpr).SubExpr
		changesMade = true
		return
	}

	// TODO implement more cases

	return
}
