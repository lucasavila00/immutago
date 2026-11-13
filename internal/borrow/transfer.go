package borrow

import (
 "go/constant"
 "go/token"
 "go/types"

 "golang.org/x/tools/go/ssa"
)

// step interprets one compiler instruction without confusing administrative addresses with source loans.
func (e *engine) step(point int,s state) {
 instruction:=e.instructions[point]
 switch op:=instruction.(type) {
 case *ssa.Alloc:
  fresh:=op.Comment=="complit"||op.Comment=="new"||op.Comment=="slicelit"||op.Comment=="makeslice"
  id:=e.root(op,storageType(op.Type()),false,-1,fresh)
  s.values[op]=[]reference{{place:Place{Root:id},loan:-1}}
 case *ssa.MakeSlice:
  id:=e.root(op,op.Type(),false,-1,true);s.values[op]=[]reference{{place:Place{Root:id},loan:-1}}
 case *ssa.FieldAddr:
  s.values[op]=e.project(point,op.Pos(),e.refs(s,op.X),'f',int64(op.Field))
 case *ssa.IndexAddr:
  kind,index:=indexProjection(op.Index)
  s.values[op]=e.project(point,op.Pos(),e.refs(s,op.X),kind,index)
 case *ssa.UnOp:
  e.load(point,s,op)
 case *ssa.Store:
  e.store(point,s,op)
 case *ssa.Slice:
  refs:=e.refs(s,op.X);out:=make([]reference,0,len(refs))
  for _,r:=range refs {
   e.use(point,op.Pos(),r,false)
   if !e.roots[r.place.Root].fresh || r.shared { r=e.borrow(r,point) }
   out=append(out,r)
  };s.values[op]=out
 case *ssa.Phi:
  var refs []reference;for _,edge:=range op.Edges { refs,_=mergeReferences(refs,e.refs(s,edge)) };s.values[op]=refs
 case *ssa.ChangeType: s.values[op]=e.copied(point,op.Pos(),e.refs(s,op.X))
 case *ssa.Convert: s.values[op]=e.copied(point,op.Pos(),e.refs(s,op.X))
 case *ssa.ChangeInterface: s.values[op]=e.copied(point,op.Pos(),e.refs(s,op.X))
 case *ssa.MakeInterface: s.values[op]=e.copied(point,op.Pos(),e.refs(s,op.X))
 case *ssa.TypeAssert: s.values[op]=e.copied(point,op.Pos(),e.refs(s,op.X))
 case *ssa.Extract:
  var refs []reference;for _,r:=range e.refs(s,op.Tuple) { if r.result==op.Index { r.result=0;refs=append(refs,r) } };s.values[op]=refs
 case *ssa.Call: s.values[op]=e.call(point,s,op.Common(),false,false)
 case *ssa.Defer: e.call(point,s,op.Common(),true,false)
 case *ssa.Go: e.call(point,s,op.Common(),false,true)
 case *ssa.MakeClosure: e.closure(point,s,op)
 case *ssa.Return: e.returned(point,s,op)
 case *ssa.DebugRef,*ssa.Jump: // Debug references are source metadata, not uses.
 default:
  e.other(point,s,instruction)
 }
}

// project retains alias provenance while selecting a disjoint field or possibly overlapping index.
func (e *engine) project(point int,pos token.Pos,refs []reference,kind byte,index int64) []reference {
 result:=make([]reference,0,len(refs))
 for _,r:=range refs { e.use(point,pos,r,false);r.place=r.place.project(kind,index);result=append(result,r) }
 return result
}

// indexProjection distinguishes statically disjoint array elements from unknown runtime indices.
func indexProjection(value ssa.Value) (byte,int64) {
 if c,ok:=value.(*ssa.Const);ok&&c.Value!=nil&&c.Value.Kind()==constant.Int {
  if index,ok:=constant.Int64Val(c.Value);ok{return 'i',index}
 }
 return '?',0
}

// copied records pointer-value uses without changing the origin of already shared references.
func (e *engine) copied(point int,pos token.Pos,refs []reference) []reference {
 for _,r:=range refs { e.use(point,pos,r,false) };return refs
}

// load resolves pointer variables through memory and records ordinary pointee reads.
func (e *engine) load(point int,s state,op *ssa.UnOp) {
 refs:=e.refs(s,op.X)
 for _,r:=range refs { e.use(point,op.Pos(),r,false) }
 if op.Op!=token.MUL || !referenceType(op.Type()) { return }
 var result []reference
 for _,r:=range refs {
  found:=false
  for _,stored:=range s.cells { if Overlap(r.place,stored.place,false) {
   result,_=mergeReferences(result,stored.refs);found=true
  } }
  if !found && e.roots[r.place.Root].global {
   id:=e.root(op,storageType(op.Type()),true,-1,false)
   result=append(result,reference{place:Place{Root:id},loan:-1,shared:true})
  }
 }
 s.values[op]=result
}

// store performs a strong update only for one definite local cell; may-alias writes use unions.
func (e *engine) store(point int,s state,op *ssa.Store) {
 addresses:=e.refs(s,op.Addr)
 source:=e.refs(s,op.Val)
 for _,address:=range addresses { e.use(point,op.Pos(),address,true) }
 source=e.copied(point,op.Pos(),source)
 for _,address:=range addresses {
  var values []reference
  for _,r:=range source {
   origin:=&e.roots[r.place.Root]
   if !r.shared && (origin.binding>=0&&origin.binding!=point || !origin.fresh) { r=e.borrow(r,point) }
   if !r.shared && origin.binding<0 { origin.binding=point }
   values=append(values,r)
   if len(address.place.Path)>0 && e.roots[r.place.Root].protected {
    e.report(op.Pos(),"unsupported borrowed reference stored in an aggregate")
   }
   if e.roots[address.place.Root].global { e.escape(point,r,false) }
  }
  key:=address.place.key()
  strong:=len(addresses)==1
  for _,projection:=range address.place.Path { strong=strong&&projection.Kind!='?' }
  if strong { s.cells[key]=cell{place:address.place,refs:values} } else {
   old:=s.cells[key];joined,_:=mergeReferences(old.refs,values);s.cells[key]=cell{place:address.place,refs:joined}
  }
 }
}

// other records remaining reads and rejects unsupported escapes of protected references.
func (e *engine) other(point int,s state,instruction ssa.Instruction) {
 var refs []reference
 for _,operand:=range instruction.Operands(nil) { if operand!=nil&&*operand!=nil {
  refs,_=mergeReferences(refs,e.refs(s,*operand))
 } }
 for _,r:=range refs { e.use(point,instruction.Pos(),r,false) }
 switch instruction.(type) {
 case *ssa.Send,*ssa.MapUpdate:
  for _,r:=range refs { if e.roots[r.place.Root].protected {
   e.report(instruction.Pos(),"unsupported reference escape through mutable container or channel")
  } }
 }
}

// globallyProtected distinguishes immutable global values from rebindable pointer slots.
func globallyProtected(t types.Type) bool {
 switch t.Underlying().(type) { case *types.Pointer,*types.Slice,*types.Map,*types.Interface,*types.Signature:return false }
 return true
}
