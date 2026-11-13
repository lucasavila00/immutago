package analyzer

import (
	"go/ast"
	"go/types"

	"github.com/lucasavila00/immutago/internal/checker"
)

// compositeLiteral checks each implicit assignment against its resolved storage type.
func (c *subsetChecks) compositeLiteral(literal *ast.CompositeLit) {
	storage := c.pass.TypesInfo.TypeOf(literal).Underlying()
	for index, element := range literal.Elts {
		value := element
		pair, keyed := element.(*ast.KeyValueExpr)
		if keyed {
			value = pair.Value
		}
		switch storage := storage.(type) {
		case *types.Array:
			c.transport(value, storage.Elem())
		case *types.Slice:
			c.transport(value, storage.Elem())
		case *types.Map:
			c.transport(pair.Key, storage.Key())
			c.transport(value, storage.Elem())
		case *types.Struct:
			if keyed {
				field := c.pass.TypesInfo.ObjectOf(pair.Key.(*ast.Ident))
				c.transport(value, field.Type())
			} else {
				c.transport(value, storage.Field(index).Type())
			}
		default:
			operands := []ast.Expr{value}
			if keyed {
				operands = append(operands, pair.Key)
			}
			for _, operand := range operands {
				typ := c.pass.TypesInfo.TypeOf(operand)
				if checker.Contains(typ) || protectedCallable(typ) {
					c.report(operand, "protected composite literals require a concrete storage type")
				}
			}
		}
	}
}
