package projectiongen

import (
	"bytes"
	"fmt"
	"go/types"
	"strings"
	"unicode"

	"github.com/lucasavila00/immutago/internal/checker"
)

// readPath selects one field or element from a pointer root without copying its parent.
func (g *generator) readPath(typ, root types.Type, path, stem string, indices []string) (string, string, error) {
	if fixed, ok := typ.Underlying().(*types.Array); ok {
		return g.arrayReader(fixed, root, path, stem, indices)
	}
	expression := "v." + g.projectionField + ".Unsafe()" + strings.TrimPrefix(path, "value")
	selected := typ
	if checker.Frozen(typ) {
		selected = types.NewPointer(typ)
		expression = "&" + expression
	}
	if frozenPointer(selected) {
		name, constructor, err := g.pointerReader(selected)
		return name, constructor + "(" + g.imName + ".ProjectionOf(" + expression + "))", err
	}
	if checker.FrozenInterface(typ) {
		name, constructor, err := g.interfaceReader(typ)
		return name, constructor + "(" + g.imName + ".ProjectionOf(" + expression + "))", err
	}
	if g.proofs.ReadResultReason(typ) == "" {
		return g.typeName(typ), expression, nil
	}
	if result, expression, ok := g.collectionReader(typ, expression); ok {
		return result, expression, nil
	}
	// The Frozen proof leaves only native strings after the other supported forms.
	return g.imName + ".StringProjection", g.imName + ".ProjectString(" + expression + ")", nil
}

// frozenPointer identifies pointers to marked immutable records.
func frozenPointer(typ types.Type) bool {
	pointer, ok := types.Unalias(typ).(*types.Pointer)
	return ok && checker.Frozen(pointer.Elem())
}

// pointerReader uses the projection generated in the target record's package.
func (g *generator) pointerReader(typ types.Type) (string, string, error) {
	pointer := types.Unalias(typ).(*types.Pointer)
	named, ok := types.Unalias(pointer.Elem()).(*types.Named)
	if !ok {
		return "", "", fmt.Errorf("projection target must be a named Frozen record")
	}
	projection := named.Obj().Name() + "Projection"
	if named.Obj().Pkg() == g.pkg {
		return projection, constructorName(projection), nil
	}
	if !named.Obj().Exported() {
		return "", "", fmt.Errorf("Frozen record %s requires an exported declaration for a foreign projection",
			g.typeName(named))
	}
	full := g.typeName(named)
	qualifier := strings.TrimSuffix(full, named.Obj().Name())
	return qualifier + projection, qualifier + constructorName(projection), nil
}

// collectionReader maps fixed collection types to their local projection APIs.
func (g *generator) collectionReader(typ types.Type, input string) (string, string, bool) {
	if checker.IsLibraryType(typ, "FrozenString") {
		return g.imName + ".StringProjection", input + ".Projection()", true
	}
	named, ok := types.Unalias(typ).(*types.Named)
	if !ok {
		return "", "", false
	}
	if checker.IsLibraryType(typ, "FrozenSlice") {
		return g.imName + ".SliceProjection[" + g.typeName(named.TypeArgs().At(0)) + "]", input + ".Projection()", true
	}
	if checker.IsLibraryType(typ, "FrozenMap") {
		arguments := g.typeName(named.TypeArgs().At(0)) + ", " + g.typeName(named.TypeArgs().At(1))
		return g.imName + ".MapProjection[" + arguments + "]", input + ".Projection()", true
	}
	return "", "", false
}

// writeRecord emits pointer-based accessors and a typed cell for a source record.
func (g *generator) writeRecord(out *bytes.Buffer, item record) error {
	source := g.typeName(item.typ)
	fmt.Fprintf(out, "// %s provides local read access to %s.\n"+
		"type %s struct {\n%s\n%s %s.Projection[%s]\n}\n",
		item.name, source, item.name, g.localMarker, g.projectionField, g.imName, source)
	fmt.Fprintf(out, "// %s wraps a checked local projection without copying backing storage.\n",
		constructorName(item.name))
	fmt.Fprintf(out, "func %s(value %s.Projection[%s]) %s { return %s{%s:value} }\n",
		constructorName(item.name), g.imName, source, item.name, item.name, g.projectionField)
	fmt.Fprintf(out, "// IsNil reports whether the selected pointer is nil.\n"+
		"func (v %s) IsNil() bool { return %s.IsNil(v.%s) }\n", item.name, g.imName, g.projectionField)
	fmt.Fprintf(out, "// Unsafe returns the selected pointer without a record copy or allocation.\n"+
		"// The caller controls retention and must not change the target.\n"+
		"func (v %s) Unsafe() %s { return v.%s.Unsafe() }\n", item.name, source, g.projectionField)
	target := types.Unalias(item.typ).(*types.Pointer).Elem()
	structure := target.Underlying().(*types.Struct)
	for field := range structure.Fields() {
		if field.Name() == "_" || checker.IsLibraryType(field.Type(), "Frozen") {
			continue
		}
		if field.Name() == "IsNil" || field.Name() == "Unsafe" {
			return fmt.Errorf("%s.%s conflicts with a generated projection member", source, field.Name())
		}
		if err := g.writeField(out, item, field); err != nil {
			return err
		}
	}
	g.writeCell(out, item)
	return nil
}

// writeField emits one typed getter without copying the containing record.
func (g *generator) writeField(out *bytes.Buffer, item record, field *types.Var) error {
	stem := strings.TrimSuffix(item.name, "Projection") + field.Name()
	result, expression, err := g.readPath(field.Type(), item.typ, "value."+field.Name(), stem, nil)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "// %s reads the %s field.\nfunc (v %s) %s() %s {\n",
		field.Name(), field.Name(), item.name, field.Name(), result)
	if _, fixed := field.Type().Underlying().(*types.Array); fixed {
		fmt.Fprintf(out, "if %s.IsNil(v.%s) { panic(\"nil frozen record\") }\n", g.imName, g.projectionField)
	}
	fmt.Fprintf(out, "return %s\n}\n", expression)
	return nil
}

// writeCell emits an ordinary holder whose Load reads the published root pointer.
func (g *generator) writeCell(out *bytes.Buffer, item record) {
	stem := strings.TrimSuffix(item.name, "Projection")
	source := g.typeName(types.Unalias(item.typ).(*types.Pointer).Elem())
	fmt.Fprintf(out, "// %sCell publishes frozen %s values. Do not copy it.\ntype %sCell struct { value %s.Cell[%s] }\n",
		stem, source, stem, g.imName, source)
	fmt.Fprintf(out, "// Store publishes value.\n"+
		"func (c *%sCell) Store(value %s) { c.value.Store(value) }\n", stem, source)
	fmt.Fprintf(out, "// Load reads the current value through a local projection.\n"+
		"func (c *%sCell) Load() %s { return %s(c.value.LoadPointer()) }\n",
		stem, item.name, constructorName(item.name))
}

// constructorName preserves the source type's exported or unexported visibility.
func constructorName(projection string) string {
	first := []rune(projection)
	if !unicode.IsUpper(first[0]) {
		first[0] = unicode.ToUpper(first[0])
		return "new" + string(first)
	}
	return "New" + projection
}
