package viewgen

import (
	"bytes"
	"fmt"
	"go/types"
	"strings"
)

// arrayReader retains only the immutable root pointer and existing indices.
func (g *generator) arrayReader(
	typ *types.Array, root types.Type, path, stem string, indices []string,
) (string, string, error) {
	name := stem + "View"
	if err := g.reserve(name); err != nil {
		return "", "", err
	}
	g.arrays = append(g.arrays, array{typ: typ, name: name, root: root, path: path, indices: indices})
	fields := []string{"immutagoView:v.immutagoView"}
	for _, index := range indices {
		fields = append(fields, index+":"+index)
	}
	return name, name + "{" + strings.Join(fields, ",") + "}", nil
}

// writeArray emits indexed reads from the original frozen record.
func (g *generator) writeArray(out *bytes.Buffer, item array) error {
	index := fmt.Sprintf("index%d", len(item.indices))
	indices := append(append([]string(nil), item.indices...), index)
	result, expression, err := g.readPath(item.typ.Elem(), item.root, item.path+"["+index+"]",
		strings.TrimSuffix(item.name, "View")+"Element", indices)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "// %s reads a fixed array locally.\ntype %s struct { im.Local; immutagoView im.View[%s]\n",
		item.name, item.name, g.typeName(item.root))
	for _, saved := range item.indices {
		fmt.Fprintf(out, "%s int\n", saved)
	}
	out.WriteString("}\n")
	fmt.Fprintf(out, "// Len returns the array length.\nfunc (v %s) Len() int { return %d }\n", item.name, item.typ.Len())
	fmt.Fprintf(out, "// At reads the element at index.\nfunc (v %s) At(%s int) %s {\n", item.name, index, result)
	for _, saved := range item.indices {
		fmt.Fprintf(out, "%s := v.%s\n", saved, saved)
	}
	fmt.Fprintf(out, "return %s\n}\n", expression)
	return nil
}
