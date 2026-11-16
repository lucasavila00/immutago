package analyzer

import (
	"go/ast"
	"go/types"

	"github.com/lucasavila00/immutago/internal/checker"
)

// checkHandles proves Ref targets.
func (c *subsetChecks) checkHandles() {
	for _, file := range c.pass.Files {
		ast.Inspect(file, c.visitHandle)
	}
}

// visitHandle checks declared and inferred reference types.
func (c *subsetChecks) visitHandle(node ast.Node) bool {
	if node, ok := node.(*ast.Ident); ok {
		if instance, ok := c.pass.TypesInfo.Instances[node]; ok {
			c.checkReference(node, instance.Type)
		}
	}
	if expression, ok := node.(ast.Expr); ok {
		if value := c.pass.TypesInfo.Types[expression]; value.IsValue() {
			c.checkReference(expression, value.Type)
		}
	}
	return true
}

// checkReference proves the concrete service target without restricting value transport.
func (c *subsetChecks) checkReference(node ast.Node, typ types.Type) {
	if checker.IsRuntimeType(typ, "Ref") {
		if failure := c.proofs.Failure(typ); failure.Reason != "" {
			c.reportFailure(node, failure)
		}
	}
}
