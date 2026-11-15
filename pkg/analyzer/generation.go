package analyzer

import (
	"bytes"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/lucasavila00/immutago/internal/checker"
	"github.com/lucasavila00/immutago/internal/viewgen"
)

// checkGeneration requires the single generated API and still checks its ordinary Go body.
func (c *subsetChecks) checkGeneration() {
	position, valid := c.generationSource()
	if !valid {
		return
	}
	var generated *ast.File
	for _, file := range c.pass.Files {
		if filepath.Base(c.physicalFile(file.Pos())) == viewgen.Filename {
			generated = file
		}
	}
	if !position.IsValid() {
		if generated != nil {
			c.pass.Reportf(generated.Pos(), "generated views have no source marked records; remove %s", viewgen.Filename)
		}
		return
	}
	if !c.generationDirective(position) {
		return
	}
	expected, err := viewgen.GenerateTypes(c.pass.Pkg, c.pass.Fset)
	if err != nil {
		c.pass.Reportf(position, "cannot generate typed views: %s", err)
		return
	}
	if generated == nil {
		c.pass.Reportf(position, "missing %s; run go generate in this package", viewgen.Filename)
		return
	}
	readFile := c.pass.ReadFile
	if readFile == nil {
		readFile = os.ReadFile
	}
	actual, err := readFile(c.physicalFile(generated.Pos()))
	if err != nil || !bytes.Equal(bytes.ReplaceAll(actual, []byte("\r\n"), []byte("\n")), expected) {
		c.pass.Reportf(generated.Pos(), "%s is stale; run go generate in this package", viewgen.Filename)
	}
}

// physicalFile ignores line directives when selecting the generator-owned source file.
func (c *subsetChecks) physicalFile(position token.Pos) string {
	return c.pass.Fset.PositionFor(position, false).Filename
}

// generationSource requires package-level named records in ordinary source files.
func (c *subsetChecks) generationSource() (token.Pos, bool) {
	var first token.Pos
	valid := true
	for _, file := range c.pass.Files {
		if filepath.Base(c.physicalFile(file.Pos())) == viewgen.Filename {
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
		object := c.pass.TypesInfo.Defs[declaration.Name]
		if declaration.Assign.IsValid() || object == nil || !checker.Marked(object.Type()) {
			return token.NoPos, true
		}
		if object.Parent() != c.pass.Pkg.Scope() || strings.HasSuffix(c.physicalFile(declaration.Pos()), "_test.go") {
			c.pass.Reportf(declaration.Pos(), "marked records require package-level declarations in ordinary .go files")
			return declaration.Pos(), false
		}
		return declaration.Pos(), true
	case *ast.StructType:
		if !checker.Marked(c.pass.TypesInfo.TypeOf(declaration)) {
			return token.NoPos, true
		}
		owner, named := c.parents[declaration].(*ast.TypeSpec)
		if !named || owner.Assign.IsValid() {
			c.pass.Reportf(declaration.Pos(),
				"marked records require a named package-level type; anonymous records are unsupported")
			return declaration.Pos(), false
		}
	}
	return token.NoPos, true
}

// generationDirective accepts one exact command without accepting other invocation shapes.
func (c *subsetChecks) generationDirective(position token.Pos) bool {
	count := 0
	valid := true
	for _, file := range c.pass.Files {
		if filepath.Base(c.physicalFile(file.Pos())) == viewgen.Filename {
			continue
		}
		number, allowed := c.generationCommands(file)
		count += number
		valid = valid && allowed
	}
	if count != 1 || !valid {
		c.pass.Reportf(position, "marked records require exactly one package directive: %s", viewgen.Directive)
		return false
	}
	return true
}

// generationCommands counts exact directives and rejects alternate immutago invocations.
func (c *subsetChecks) generationCommands(file *ast.File) (int, bool) {
	count := 0
	valid := true
	for _, group := range file.Comments {
		for _, comment := range group.List {
			if !strings.HasPrefix(comment.Text, "//go:generate") || !strings.Contains(comment.Text, "immutago") {
				continue
			}
			if comment.Text == viewgen.Directive && c.pass.Fset.PositionFor(comment.Pos(), false).Column == 1 {
				count++
			} else {
				valid = false
			}
		}
	}
	return count, valid
}
