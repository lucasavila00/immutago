package borrow

import (
 "fmt"
 "go/token"
 "go/types"

 "github.com/lucasavila00/immutago/internal/checker"
 "golang.org/x/tools/go/ssa"
)

// call applies checked input effects and carries returned aliases back to the caller.
func (e *engine) call(point int,s state,call *ssa.CallCommon,deferred,asynchronous bool) []reference {
 if builtin,ok:=call.Value.(*ssa.Builtin);ok { return e.builtin(point,s,call,builtin.Name()) }
 callee:=call.StaticCallee()
 arguments:=make([][]reference,len(call.Args))
 for i,arg:=range call.Args { arguments[i]=e.refs(s,arg) }
 if callee==nil {
  for _,r:=range e.refs(s,call.Value) {
   e.use(point,call.Pos(),r,false)
   if r.closure!=nil {
    if callee!=nil&&callee!=r.closure { e.unknownCall(point,call.Pos(),arguments);return nil }
    callee=r.closure
   }
  }
 }
 if callee!=nil && len(callee.FreeVars)>0 {
  for range callee.FreeVars { arguments=append(arguments,nil) }
  for _,r:=range e.refs(s,call.Value) { if r.binding>=0 {
   index:=len(call.Args)+r.binding;r.closure=nil;r.binding=0
   arguments[index]=append(arguments[index],r)
  } }
 }
 if isLibrary(callee) {
  for _,args:=range arguments { for _,r:=range args { e.use(point,call.Pos(),r,false) } }
  return nil // Sealed operations return value copies or immutable iterators, never writable aliases.
 }
 summary,known:=e.program.summary(callee)
 if !known { e.unknownCall(point,call.Pos(),arguments);return nil }
 for i,args:=range arguments {
  effect:=Effect{};if i<len(summary.Inputs){effect=summary.Inputs[i]}
  for j,r:=range args {
   // Calls borrow caller-owned pointer arguments; the callee does not obtain ownership by copying them.
   r=e.borrow(r,point);arguments[i][j]=r
   e.use(point,call.Pos(),r,effect.Write)
   if effect.Capture||asynchronous { e.escape(point,r,false) }
   if deferred { e.escape(point,r,true) }
  }
 }
 return e.callResults(point,call,summary,arguments)
}

// isLibrary recognizes only the fixed collection implementation boundary.
func isLibrary(fn *ssa.Function) bool {
 return fn!=nil && fn.Pkg!=nil && fn.Pkg.Pkg.Path()==checker.LibraryPath
}

// unknownCall rejects unavailable effects only when protected references cross the call boundary.
func (e *engine) unknownCall(point int,pos token.Pos,arguments [][]reference) {
 for _,args:=range arguments { for _,r:=range args {
  e.use(point,pos,r,false)
  if e.roots[r.place.Root].protected { e.report(pos,"unsupported call effects for borrowed immutable storage") }
 } }
}

// callResults instantiates symbolic returned aliases and keeps repeated fresh results on one root.
func (e *engine) callResults(point int,call *ssa.CallCommon,summary Summary,args [][]reference) []reference {
 var result []reference
 freshSeen:=make(map[int]bool)
 for index,aliases:=range summary.Results { for _,alias:=range aliases {
  if alias.Input>=0 && alias.Input<len(args) {
   for _,r:=range args[alias.Input] {
    r.place.Path=append(append([]Projection(nil),r.place.Path...),alias.Path...)
    r.result=index;result=append(result,r)
   }
   continue
  }
  if !alias.Fresh { continue }
  key:=fmt.Sprintf("%d:%d",point,alias.FreshID)
  id,ok:=e.callRoots[key]
  if !ok {
   typ:=call.Signature().Results().At(index).Type()
   id=len(e.roots);e.callRoots[key]=id
   e.roots=append(e.roots,root{typ:storageType(typ),protected:alias.Protected||checker.Contains(storageType(typ)),
    input:-1,fresh:true,binding:-1})
  }
  r:=reference{place:Place{Root:id,Path:alias.Path},loan:-1,result:index}
  if alias.Shared||freshSeen[alias.FreshID] { r=e.borrow(r,point) }
  freshSeen[alias.FreshID]=true;result=append(result,r)
 } }
 return result
}

// builtin models standard operations that read or write native storage.
func (e *engine) builtin(point int,s state,call *ssa.CallCommon,name string) []reference {
 if name=="new" { return nil } // The SSA builder lowers new to Alloc.
 var result []reference
 for index,arg:=range call.Args { for _,r:=range e.refs(s,arg) {
  write:=index==0&&(name=="copy"||name=="clear"||name=="append")
  e.use(point,call.Pos(),r,write)
  if name=="append"&&index==0 { result=append(result,e.borrow(r,point)) }
 } }
 return result
}

// escape adds a caller-visible capture or a delayed-use requirement to a loan.
func (e *engine) escape(point int,r reference,deferred bool) {
 if !e.record { return }
 e.use(point,e.instructions[point].Pos(),r,false)
 root:=e.roots[r.place.Root]
 if root.input>=0&&!deferred { e.summary.Inputs[root.input].Capture=true }
 if !deferred { e.captured[r.place.Root]=true }
 if r.loan>=0 { if deferred {e.loans[r.loan].deferred=true} else {e.loans[r.loan].external=true} }
}

// returned records origin relationships instead of treating every returned pointer as fresh.
func (e *engine) returned(point int,s state,op *ssa.Return) {
 if !e.record { return }
 for i,value:=range op.Results { for _,r:=range e.refs(s,value) {
  e.use(point,op.Pos(),r,false)
  root:=e.roots[r.place.Root]
  result:=Result{Input:root.input,Path:r.place.Path,Protected:root.protected}
  if root.input<0&&!root.global { result.Fresh=true;result.FreshID=r.place.Root }
  if root.global { result.Shared=true }
  found:=false
  for _,old:=range e.summary.Results[i] { if sameResult(old,result) { found=true;break } }
  if !found { e.summary.Results[i]=append(e.summary.Results[i],result) }
 } }
}

// sameResult compares portable origin summaries without depending on slice identity.
func sameResult(a,b Result) bool {
 if a.Input!=b.Input||a.Fresh!=b.Fresh||a.FreshID!=b.FreshID||a.Shared!=b.Shared||a.Protected!=b.Protected {
  return false
 }
 return Place{Path:a.Path}.key()==(Place{Path:b.Path}).key()
}

// closure preserves each captured reference and turns address captures into shared loans.
func (e *engine) closure(point int,s state,op *ssa.MakeClosure) {
 fn:=op.Fn.(*ssa.Function)
 var refs []reference
 for i,binding:=range op.Bindings { for _,r:=range e.refs(s,binding) {
  r=e.borrow(r,point);e.use(point,op.Pos(),r,false);r.closure=fn;r.binding=i;refs=append(refs,r)
 } }
 if len(refs)==0 {
  id:=e.root(op,types.NewSignatureType(nil,nil,nil,nil,nil,false),false,-1,false)
  refs=[]reference{{place:Place{Root:id},loan:-1,closure:fn,binding:-1}}
 }
 s.values[op]=refs
}
