package objfilter

import (
	"errors"
	"fmt"
)

// evaluate returns true if expr matches the supplied entry
func evaluate(expr expression, obj Object) (result bool) {
	switch n := expr.(type) {
	case *rootExpr:
		return evaluate(n.SubExpr, obj)
	case *simpleExpr:
		val1 := obj.GetValue(n.IdentifierOperand)
		val2 := n.ValueOperand

		return evalOperator(val1, val2, n.Operator)
	case *regexMatchExpr:
		val1 := obj.GetValue(n.IdentifierOperand)
		if !val1.Ok {
			return false
		}

		// if the value isn't a string, return false
		if _, ok := val1.V.(string); !ok {
			return false
		}

		// apply regex
		return n.Re.MatchString(val1.V.(string))
	case *binaryBoolExpr:
		switch n.Operator.Canonicalize() {
		case andOp:
			// All operands must evaluate to true, or the entire expression is false
			for i := range n.Operands {
				result = evaluate(n.Operands[i], obj)
				if !result {
					return false
				}
			}

			return true
		case orOp:
			// If any operand evaluates to true, the entire expression is true
			for i := range n.Operands {
				result = evaluate(n.Operands[i], obj)
				if result {
					return true
				}
			}

			return false
		default:
			// we validate expressions before evaluating them, so this should never happen
			panic(errors.New("unknown boolean operator"))
		}
	case *notExpr:
		result = evaluate(n.SubExpr, obj)
		return !result
	case *parenExpr:
		return evaluate(n.SubExpr, obj)
	case *boolLiteralExpr:
		return n.Value
	default:
		panic(fmt.Errorf("this should not happen: %+v", n))
	}
}

type Value struct {
	V  any
	Ok bool
}

func evalOperator(op1, op2 Value, operator comparisonOp) bool {
	switch operator.Canonicalize() {
	case equalsOp:
		fallthrough
	case altEqualsOp:
		// TODO do type coercion
		return op1.V == op2.V
	case notEqualsOp:
		// TODO do type coercion
		return op1.V != op2.V
	case regexMatchOp:
		panic(errors.New("this shouldn't be executed -- there is a separate expression for regexes"))
	default:
		// we validate expressions before evaluating them, so this should never happen
		panic(fmt.Errorf("unknown operator: %s", operator))
	}
}
