package analyzer

import (
	"bytes"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/lucasavila00/immutago/internal/checker"
	"github.com/lucasavila00/immutago/internal/projectiongen"
)

// checkGeneration verifies the exact generated source before it can read projection storage.
func (c *subsetChecks) checkGeneration() {
	position, valid := c.generationSource()
	if !valid {
		return
	}
	var generated *ast.File
	for _, file := range c.pass.Files {
		if filepath.Base(c.physicalFile(file.Pos())) == projectiongen.Filename {
			generated = file
		}
	}
	if !position.IsValid() {
		if generated != nil {
			c.pass.Reportf(generated.Pos(), "generated projections have no source Frozen records; remove %s",
				projectiongen.Filename)
		}
		return
	}
	if !c.generationDirective(position) {
		return
	}
	expected, err := projectiongen.GenerateTypes(c.pass.Pkg, c.pass.Fset)
	if err != nil {
		c.pass.Reportf(position, "cannot generate typed projections: %s", err)
		return
	}
	if generated == nil {
		c.pass.Reportf(position, "missing %s; run go generate in this package", projectiongen.Filename)
		return
	}
	readFile := c.pass.ReadFile
	if readFile == nil {
		readFile = os.ReadFile
	}
	actual, err := readFile(c.physicalFile(generated.Pos()))
	if err != nil || !bytes.Equal(bytes.ReplaceAll(actual, []byte("\r\n"), []byte("\n")), expected) {
		c.pass.Reportf(generated.Pos(), "%s is stale; run go generate in this package", projectiongen.Filename)
		return
	}
	c.generated = c.pass.Fset.File(generated.Pos())
}

// generatedSource recognizes only the physical file whose complete bytes passed validation.
func (c *subsetChecks) generatedSource(node ast.Node) bool {
	return c.generated != nil && c.pass.Fset.File(node.Pos()) == c.generated
}

// physicalFile ignores line directives when selecting the generator-owned source file.
func (c *subsetChecks) physicalFile(position token.Pos) string {
	return c.pass.Fset.PositionFor(position, false).Filename
}

// generationSource requires package-level named Frozen records in ordinary source files.
func (c *subsetChecks) generationSource() (token.Pos, bool) {
	var first token.Pos
	valid := true
	for _, file := range c.pass.Files {
		if filepath.Base(c.physicalFile(file.Pos())) == projectiongen.Filename {
			continue
		}
		ast.Inspect(file, func(node ast.Node) bool {
			position, allowed := c.generationDeclaration(node)
			if !first.IsValid() {
				first = position
			}
			valid = valid && allowed
			return true
		})
	}
	return first, valid
}

// generationDeclaration matches a named source record and its declaration scope.
func (c *subsetChecks) generationDeclaration(node ast.Node) (token.Pos, bool) {
	switch declaration := node.(type) {
	case *ast.TypeSpec:
		return c.generationNamed(declaration)
	case *ast.InterfaceType:
		if !checker.FrozenInterface(c.pass.TypesInfo.TypeOf(declaration)) {
			return token.NoPos, true
		}
		owner, named := c.parents[declaration].(*ast.TypeSpec)
		if !named || owner.Assign.IsValid() {
			c.pass.Reportf(declaration.Pos(),
				"Frozen interfaces require a named package-level type; anonymous interfaces are unsupported")
			return declaration.Pos(), false
		}
	case *ast.StructType:
		if !checker.Frozen(c.pass.TypesInfo.TypeOf(declaration)) {
			return token.NoPos, true
		}
		owner, named := c.parents[declaration].(*ast.TypeSpec)
		if !named || owner.Assign.IsValid() {
			c.pass.Reportf(declaration.Pos(),
				"Frozen records require a named package-level type; anonymous records are unsupported")
			return declaration.Pos(), false
		}
	}
	return token.NoPos, true
}

// generationNamed requires a non-test package declaration for each generated source type.
func (c *subsetChecks) generationNamed(declaration *ast.TypeSpec) (token.Pos, bool) {
	object := c.pass.TypesInfo.Defs[declaration.Name]
	if declaration.Assign.IsValid() || object == nil ||
		(!checker.Frozen(object.Type()) && !checker.FrozenInterface(object.Type())) {
		return token.NoPos, true
	}
	if object.Parent() != c.pass.Pkg.Scope() || strings.HasSuffix(c.physicalFile(declaration.Pos()), "_test.go") {
		kind := "Frozen records"
		if checker.FrozenInterface(object.Type()) {
			kind = "Frozen interfaces"
		}
		c.pass.Reportf(declaration.Pos(), "%s require package-level declarations in ordinary .go files", kind)
		return declaration.Pos(), false
	}
	return declaration.Pos(), true
}

// generationDirective accepts one exact command without accepting other invocation shapes.
func (c *subsetChecks) generationDirective(position token.Pos) bool {
	count := 0
	for _, file := range c.pass.Files {
		name := c.physicalFile(file.Pos())
		if filepath.Base(name) == projectiongen.Filename || strings.HasSuffix(name, "_test.go") {
			continue
		}
		count += c.generationCommands(file)
	}
	if count != 1 {
		kind := "Frozen records"
		for _, object := range c.pass.TypesInfo.Defs {
			if object != nil && object.Pos() == position && checker.FrozenInterface(object.Type()) {
				kind = "Frozen interfaces"
				break
			}
		}
		c.pass.Reportf(position, "%s require exactly one package directive: %s", kind, projectiongen.Directive)
		return false
	}
	return true
}

// generationCommands counts the exact required command and leaves other tools unchanged.
func (c *subsetChecks) generationCommands(file *ast.File) int {
	count := 0
	for _, group := range file.Comments {
		for _, comment := range group.List {
			if comment.Text == projectiongen.Directive && c.pass.Fset.PositionFor(comment.Pos(), false).Column == 1 {
				count++
			}
		}
	}
	return count
}
