package projectiongen

import (
	"bytes"
	"fmt"
	"go/types"
	"strings"

	"github.com/lucasavila00/immutago/internal/checker"
)

// addInterface queues a local method reader and its constructor.
func (g *generator) addInterface(typ types.Type, name string) error {
	if err := g.reserve(name); err != nil {
		return err
	}
	if err := g.reserve(constructorName(name)); err != nil {
		return err
	}
	g.interfaces = append(g.interfaces, record{typ: typ, name: name})
	return nil
}

// interfaceReader selects the local adapter from the interface's declaring package.
func (g *generator) interfaceReader(typ types.Type) (string, string) {
	named := types.Unalias(typ).(*types.Named)
	projection := named.Obj().Name() + "Projection"
	if named.Obj().Pkg() == g.pkg {
		return projection, constructorName(projection)
	}
	qualifier := strings.TrimSuffix(g.typeName(named), named.Obj().Name())
	return qualifier + projection, qualifier + constructorName(projection)
}

// writeInterface forwards checked methods without returning the persistent interface.
func (g *generator) writeInterface(out *bytes.Buffer, item record) error {
	source := g.typeName(item.typ)
	fmt.Fprintf(out, "// %s provides local method access to %s.\n"+
		"type %s struct { im.Local; immutagoProjection im.Projection[%s] }\n",
		item.name, source, item.name, source)
	fmt.Fprintf(out, "// %s wraps a checked local interface projection.\n"+
		"func %s(value im.Projection[%s]) %s { return %s{immutagoProjection:value} }\n",
		constructorName(item.name), constructorName(item.name), source, item.name, item.name)
	fmt.Fprintf(out, "// IsNil reports whether the interface is nil; a stored nil pointer is not nil.\n"+
		"func (v %s) IsNil() bool { return v.immutagoProjection.Read() == nil }\n", item.name)
	for method := range item.typ.Underlying().(*types.Interface).Methods() {
		if checker.FrozenInterfaceSeal(method) {
			continue
		}
		if method.Name() == "Local" || method.Name() == "immutagoProjection" || method.Name() == "IsNil" {
			return fmt.Errorf("%s.%s conflicts with a generated projection member", source, method.Name())
		}
		g.writeInterfaceMethod(out, item.name, method)
	}
	return nil
}

// writeInterfaceMethod uses fresh parameter names and the proved read-result type.
func (g *generator) writeInterfaceMethod(out *bytes.Buffer, receiver string, method *types.Func) {
	signature := method.Type().(*types.Signature)
	var parameters, arguments []string
	for index := 0; index < signature.Params().Len(); index++ {
		parameter := signature.Params().At(index)
		name := fmt.Sprintf("arg%d", index)
		parameters = append(parameters, name+" "+g.typeName(parameter.Type()))
		arguments = append(arguments, name)
	}
	result, prefix := "", ""
	if signature.Results().Len() == 1 {
		result = " " + g.typeName(signature.Results().At(0).Type())
		prefix = "return "
	}
	fmt.Fprintf(out, "// %s calls the checked interface method with its ordinary Go contract.\n"+
		"func (v %s) %s(%s)%s { %sv.immutagoProjection.Read().%s(%s) }\n",
		method.Name(), receiver, method.Name(), strings.Join(parameters, ", "), result, prefix, method.Name(), strings.Join(arguments, ", "))
}
