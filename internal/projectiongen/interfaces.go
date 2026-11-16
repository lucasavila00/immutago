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
	source := "interface " + g.typeName(typ)
	if err := g.reserve(name, source); err != nil {
		return err
	}
	if err := g.reserve(constructorName(name), source); err != nil {
		return err
	}
	g.interfaces = append(g.interfaces, record{typ: typ, name: name})
	return nil
}

// interfaceReader selects the local adapter from the interface's declaring package.
func (g *generator) interfaceReader(typ types.Type) (string, string, error) {
	named := types.Unalias(typ).(*types.Named)
	projection := named.Obj().Name() + "Projection"
	if named.Obj().Pkg() == g.pkg {
		return projection, constructorName(projection), nil
	}
	if !named.Obj().Exported() {
		return "", "", fmt.Errorf("Frozen interface %s requires an exported declaration for a foreign projection",
			g.typeName(named))
	}
	qualifier := strings.TrimSuffix(g.typeName(named), named.Obj().Name())
	return qualifier + projection, qualifier + constructorName(projection), nil
}

// writeInterface forwards declared methods through a local interface projection.
func (g *generator) writeInterface(out *bytes.Buffer, item record) error {
	source := g.typeName(item.typ)
	fmt.Fprintf(out, "// %s provides local method access to %s.\n"+
		"type %s struct { im.Local; immutagoProjection im.Projection[%s] }\n",
		item.name, source, item.name, source)
	fmt.Fprintf(out, "// %s wraps a checked local interface projection.\n"+
		"func %s(value im.Projection[%s]) %s { return %s{immutagoProjection:value} }\n",
		constructorName(item.name), constructorName(item.name), source, item.name, item.name)
	fmt.Fprintf(out, "// IsNil reports whether the interface is nil; a stored nil pointer is not nil.\n"+
		"func (v %s) IsNil() bool { return v.immutagoProjection.Unsafe() == nil }\n", item.name)
	if g.bootstrap {
		return nil
	}
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

// writeInterfaceMethod uses fresh parameter names and forwards all declared results.
func (g *generator) writeInterfaceMethod(out *bytes.Buffer, receiver string, method *types.Func) {
	signature := method.Type().(*types.Signature)
	var parameters, arguments []string
	for index := 0; index < signature.Params().Len(); index++ {
		parameter := signature.Params().At(index)
		name := fmt.Sprintf("arg%d", index)
		parameterType := g.typeName(parameter.Type())
		argument := name
		if signature.Variadic() && index == signature.Params().Len()-1 {
			parameterType = "..." + g.typeName(parameter.Type().(*types.Slice).Elem())
			argument += "..."
		}
		parameters = append(parameters, name+" "+parameterType)
		arguments = append(arguments, argument)
	}
	result, prefix := "", ""
	var results []string
	for item := range signature.Results().Variables() {
		results = append(results, g.typeName(item.Type()))
	}
	if len(results) == 1 {
		result = " " + results[0]
	} else if len(results) > 1 {
		result = " (" + strings.Join(results, ", ") + ")"
	}
	if len(results) > 0 {
		prefix = "return "
	}
	fmt.Fprintf(out, "// %s calls the checked interface method with its ordinary Go contract.\n"+
		"func (v %s) %s(%s)%s { %sv.immutagoProjection.Unsafe().%s(%s) }\n",
		method.Name(), receiver, method.Name(), strings.Join(parameters, ", "), result, prefix,
		method.Name(), strings.Join(arguments, ", "))
}
