package borrow

import (
	"go/ast"
	"go/token"
	"go/types"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ssa"
)

// program coordinates function effects before checking loans with complete local summaries.
type program struct {
	pass            *analysis.Pass
	summaries       map[*ssa.Function]Summary
	objectSummaries map[*types.Func]Summary
	functions       []*ssa.Function
	reported        map[token.Pos]map[string]bool
}

// Check builds place-preserving SSA, infers effects to a fixed point, and checks each body.
// NaiveForm retains addressable variable slots; GlobalDebug retains source expressions.
func Check(pass *analysis.Pass) error {
	if _, err := compilerSourceDirectory(); err != nil {
		return err
	}
	if IsStandardLibrary(pass) {
		return nil
	}
	p := buildProgram(pass)
	slices.SortFunc(p.functions, func(left, right *ssa.Function) int {
		if left.Pos() < right.Pos() {
			return -1
		}
		if left.Pos() > right.Pos() {
			return 1
		}
		return 0
	})
	p.infer()
	for _, fn := range p.functions {
		newEngine(p, fn, true).run()
	}
	p.export()
	return nil
}

// buildProgram builds SSA and registers package functions, methods, and closures.
func buildProgram(pass *analysis.Pass) *program {
	ssaProgram := ssa.NewProgram(pass.Fset, ssa.NaiveForm|ssa.GlobalDebug)
	for _, imported := range pass.Pkg.Imports() {
		ssaProgram.CreatePackage(imported, nil, nil, true)
	}
	pkg := ssaProgram.CreatePackage(pass.Pkg, pass.Files, pass.TypesInfo, false)
	pkg.Build()
	p := &program{pass: pass, summaries: make(map[*ssa.Function]Summary),
		objectSummaries: make(map[*types.Func]Summary), reported: make(map[token.Pos]map[string]bool)}
	for _, member := range pkg.Members {
		if fn, ok := member.(*ssa.Function); ok {
			p.register(fn)
		}
	}
	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok {
				if object, ok := pass.TypesInfo.Defs[fn.Name].(*types.Func); ok {
					p.register(ssaProgram.FuncValue(object))
				}
			}
		}
	}
	return p
}

// register visits local and generated bodies, including otherwise unused methods and closures.
func (p *program) register(fn *ssa.Function) {
	if fn == nil || fn.Blocks == nil {
		return
	}
	if _, ok := p.summaries[fn]; ok {
		return
	}
	p.summaries[fn] = Summary{Inputs: make([]Effect, len(fn.Params)+len(fn.FreeVars)),
		Results: make([][]Result, fn.Signature.Results().Len())}
	p.functions = append(p.functions, fn)
	for _, anonymous := range fn.AnonFuncs {
		p.register(anonymous)
	}
	for _, block := range fn.Blocks {
		for _, instruction := range block.Instrs {
			for _, operand := range instruction.Operands(nil) {
				if operand != nil {
					if target, ok := (*operand).(*ssa.Function); ok {
						p.register(target)
					}
				}
			}
		}
	}
}

// infer joins effects until recursive calls and newly materialized wrappers stop changing.
func (p *program) infer() {
	for changed := true; changed; {
		changed = false
		count := len(p.functions)
		for _, fn := range p.functions {
			summary := newEngine(p, fn, false).run()
			joined, added := mergeSummary(p.summaries[fn], summary)
			p.summaries[fn] = joined
			changed = changed || added
		}
		changed = changed || count != len(p.functions)
	}
}

// summary resolves local bodies or imported facts through a generic function's origin.
func (p *program) summary(fn *ssa.Function) (Summary, bool) {
	if fn == nil {
		return Summary{}, false
	}
	if summary, ok := p.summaries[fn]; ok {
		return summary, true
	}
	if origin := fn.Origin(); origin != nil {
		if summary, ok := p.summaries[origin]; ok {
			return summary, true
		}
	}
	object, ok := fn.Object().(*types.Func)
	if ok {
		object = object.Origin()
		if summary, known := p.objectSummaries[object]; known {
			return summary, true
		}
		var fact Summary
		if p.pass.ImportObjectFact != nil && p.pass.ImportObjectFact(object, &fact) {
			p.objectSummaries[object] = fact
			return fact, true
		}
	}
	if fn.Blocks != nil {
		p.register(fn)
		return p.summaries[fn], true
	}
	return Summary{}, false
}

// export writes only this package's object facts after all recursive effects are resolved.
func (p *program) export() {
	if p.pass.ExportObjectFact == nil {
		return
	}
	facts := make(map[*types.Func]Summary)
	for fn, summary := range p.summaries {
		object, ok := fn.Object().(*types.Func)
		if !ok || object.Pkg() != p.pass.Pkg {
			continue
		}
		object = object.Origin()
		facts[object], _ = mergeSummary(facts[object], summary)
	}
	for object, summary := range facts {
		p.pass.ExportObjectFact(object, &summary)
	}
}

// report prevents duplicate findings from repeated dataflow paths at one source operation.
func (p *program) report(position token.Pos, message string) {
	if p.reported[position] == nil {
		p.reported[position] = make(map[string]bool)
	}
	if !p.reported[position][message] {
		p.reported[position][message] = true
		p.pass.Reportf(position, "%s", message)
	}
}

// mergeSummary monotonically unions parameter effects and returned alias alternatives.
func mergeSummary(left, right Summary) (Summary, bool) {
	result := Summary{Inputs: make([]Effect, max(len(left.Inputs), len(right.Inputs))),
		Results: make([][]Result, max(len(left.Results), len(right.Results)))}
	copy(result.Inputs, left.Inputs)
	changed := len(left.Inputs) != len(result.Inputs) || len(left.Results) != len(result.Results)
	for index, effect := range right.Inputs {
		old := result.Inputs[index]
		next := Effect{
			Read: old.Read || effect.Read, Write: old.Write || effect.Write,
			Capture: old.Capture || effect.Capture, Unknown: old.Unknown || effect.Unknown,
		}
		changed = changed || next != old
		result.Inputs[index] = next
	}
	for index, aliases := range left.Results {
		result.Results[index] = slices.Clone(aliases)
	}
	for index, aliases := range right.Results {
		for _, alias := range aliases {
			if losesDereference(result.Results[index], alias) && alias.Input >= 0 && alias.Input < len(result.Inputs) {
				changed = changed || !result.Inputs[alias.Input].Unknown
				result.Inputs[alias.Input].Unknown = true
			}
			var added bool
			result.Results[index], added = mergeResult(result.Results[index], alias)
			changed = changed || added
		}
	}
	return result, changed
}

// losesDereference detects widening that would confuse a pointer slot with its target.
// Such summaries require unknown input effects rather than claiming the slot contains the pointee.
func losesDereference(results []Result, next Result) bool {
	for _, current := range results {
		left, right := current, next
		left.Path, right.Path = nil, nil
		if !sameResult(left, right) {
			continue
		}
		length := 0
		for length < len(current.Path) && length < len(next.Path) && current.Path[length] == next.Path[length] {
			length++
		}
		for _, path := range [][]Projection{current.Path[length:], next.Path[length:]} {
			for _, projection := range path {
				if projection.Kind == 'd' {
					return true
				}
			}
		}
	}
	return false
}

// mergeResult widens aliases with the same source to their common storage prefix.
// This finite may-alias abstraction prevents recursive projected returns from growing paths forever.
func mergeResult(results []Result, next Result) ([]Result, bool) {
	for index, current := range results {
		left, right := current, next
		left.Path, right.Path = nil, nil
		if !sameResult(left, right) {
			continue
		}
		length := 0
		for length < len(current.Path) && length < len(next.Path) && current.Path[length] == next.Path[length] {
			length++
		}
		if length == len(current.Path) {
			return results, false
		}
		results[index].Path = slices.Clone(current.Path[:length])
		return results, true
	}
	next.Path = slices.Clone(next.Path)
	return append(results, next), true
}

// IsStandardLibrary identifies compiler-supplied source by its SDK directory.
// Unavailable standard-library effects remain unknown; this does not trust pointer calls.
func IsStandardLibrary(pass *analysis.Pass) bool {
	sdk, err := compilerSourceDirectory()
	if err != nil {
		return false
	}
	for _, file := range pass.Files {
		path := pass.Fset.PositionFor(file.Pos(), false).Filename
		relative, err := filepath.Rel(sdk, path)
		if err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) &&
			!filepath.IsAbs(relative) {
			return true
		}
	}
	return false
}

// compilerSourceDirectory uses the active Go toolchain rather than the analyzer's build-machine path.
var compilerSourceDirectory = sync.OnceValues(func() (string, error) {
	output, err := exec.Command("go", "env", "GOROOT").Output()
	if err != nil {
		return "", err
	}
	return filepath.Join(strings.TrimSpace(string(output)), "src"), nil
})
