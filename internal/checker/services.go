package checker

import "go/types"

// ServiceEmbeddingReason requires a value path to the selected runtime bind method.
func ServiceEmbeddingReason(t types.Type) string {
	const rejected = "service must embed supervise.Service or supervise.TerminalService " +
		"by value through each embedded field"
	if pointer, ok := types.Unalias(t).(*types.Pointer); ok {
		t = pointer.Elem()
	}
	methods := types.NewMethodSet(types.NewPointer(t))
	for i := 0; i < methods.Len(); i++ {
		selection := methods.At(i)
		method := selection.Obj()
		if method.Name() != "bind" || method.Pkg() == nil || method.Pkg().Path() != RuntimePath {
			continue
		}
		indices := selection.Index()
		current := t
		for _, index := range indices[:len(indices)-1] {
			current = types.Unalias(current.Underlying().(*types.Struct).Field(index).Type())
			if _, pointer := current.(*types.Pointer); pointer {
				return rejected
			}
		}
		if len(indices) > 1 && IsServiceBase(current) {
			return ""
		}
		return rejected
	}
	return rejected
}
