package analyzer

import (
	"github.com/lucasavila00/immutago/internal/checker"
	"go/ast"
	"go/types"
)

// matchLiteral reuses the complete type match when several operands share one literal.
// Statement and expression-use checks still run for every protected operand.
func (c *subsetChecks) matchLiteral(literal *ast.CompositeLit) checker.Failure {
	if rule, checked := c.literals[literal]; checked {
		return rule
	}
	rule := c.matchLiteralFields(literal)
	c.literals[literal] = rule
	return rule
}

// matchLiteralFields resolves every field, key, and element assignment in a concrete literal.
func (c *subsetChecks) matchLiteralFields(literal *ast.CompositeLit) checker.Failure {
	storage := c.pass.TypesInfo.TypeOf(literal).Underlying()
	if _, pointer := storage.(*types.Pointer); pointer {
		return shapeFailure("reference operation is outside the supported allowlist")
	}
	for index, element := range literal.Elts {
		if rule := c.matchLiteralElement(storage, index, element); rule.Reason != "" {
			return rule
		}
	}
	return checker.Failure{}
}

// matchLiteralElement resolves one concrete field, key, or element destination.
func (c *subsetChecks) matchLiteralElement(storage types.Type, index int, element ast.Expr) checker.Failure {

	value := element
	pair, keyed := element.(*ast.KeyValueExpr)
	if keyed {
		value = pair.Value
	}
	var destination types.Type
	switch storage := storage.(type) {
	case *types.Array:
		destination = storage.Elem()
	case *types.Slice:
		destination = storage.Elem()
	case *types.Map:
		if rule := c.matchStoredValue(c.pass.TypesInfo.TypeOf(pair.Key), storage.Key()); rule.Reason != "" {
			return rule
		}
		destination = storage.Elem()
	case *types.Struct:
		if keyed {
			destination = c.pass.TypesInfo.ObjectOf(pair.Key.(*ast.Ident)).Type()
		} else {
			destination = storage.Field(index).Type()
		}
	default:
		if c.queries.Contains(c.pass.TypesInfo.TypeOf(value)) || c.queries.Callable(c.pass.TypesInfo.TypeOf(value)) {
			return shapeFailure("protected composite literals require a concrete storage type")
		}
		if keyed && c.queries.Contains(c.pass.TypesInfo.TypeOf(pair.Key)) {
			return shapeFailure("protected composite literals require a concrete storage type")
		}
	}
	if rule := c.matchStoredValue(c.pass.TypesInfo.TypeOf(value), destination); rule.Reason != "" {
		return rule
	}
	return checker.Failure{}
}

// matchStoredValue permits immutable copies and Frozen pointers in aggregate storage.
func (c *subsetChecks) matchStoredValue(source, destination types.Type) checker.Failure {
	if rule := c.matchTransport(source, destination); rule.Reason != "" {
		return rule
	}
	if localReference(source) {
		return shapeFailure("Local pointers cannot enter aggregate storage")
	}
	return checker.Failure{}
}
