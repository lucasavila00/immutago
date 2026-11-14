package analyzer

import (
	"go/ast"
	"go/types"
)

// matchLiteral reuses the complete type match when several operands share one literal.
// Statement and expression-use checks still run for every protected operand.
func (c *subsetChecks) matchLiteral(literal *ast.CompositeLit) string {
	if rule, checked := c.literals[literal]; checked {
		return rule
	}
	rule := c.matchLiteralFields(literal)
	c.literals[literal] = rule
	return rule
}

// matchLiteralFields resolves every field, key, and element assignment in a concrete literal.
func (c *subsetChecks) matchLiteralFields(literal *ast.CompositeLit) string {
	storage := c.pass.TypesInfo.TypeOf(literal).Underlying()
	if _, pointer := storage.(*types.Pointer); pointer {
		return "reference operation is outside the supported allowlist"
	}
	for index, element := range literal.Elts {
		if rule := c.matchLiteralElement(storage, index, element); rule != "" {
			return rule
		}
	}
	return ""
}

// matchLiteralElement resolves one concrete field, key, or element destination.
func (c *subsetChecks) matchLiteralElement(storage types.Type, index int, element ast.Expr) string {

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
		if rule := c.matchStoredValue(c.pass.TypesInfo.TypeOf(pair.Key), storage.Key()); rule != "" {
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
			return "protected composite literals require a concrete storage type"
		}
		if keyed && c.queries.Contains(c.pass.TypesInfo.TypeOf(pair.Key)) {
			return "protected composite literals require a concrete storage type"
		}
	}
	if rule := c.matchStoredValue(c.pass.TypesInfo.TypeOf(value), destination); rule != "" {
		return rule
	}
	return ""
}

// matchStoredValue permits immutable copies and Shared pointers in aggregate storage.
func (c *subsetChecks) matchStoredValue(source, destination types.Type) string {
	if rule := c.matchTransport(source, destination); rule != "" {
		return rule
	}
	if localReference(source) {
		return "Scoped pointers cannot enter aggregate storage"
	}
	return ""
}
