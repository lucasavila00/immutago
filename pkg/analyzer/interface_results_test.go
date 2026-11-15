package analyzer

import (
	"strings"
	"testing"
)

const interfaceResultTypes = `
type Result interface { im.FrozenInterface; Lookup([]byte)(im.StringProjection,bool,[]byte,error) }
type Data struct { im.Frozen; Text string; Values im.FrozenSlice[byte] }
func(d *Data)Lookup(buf []byte)(im.StringProjection,bool,[]byte,error){
return im.ProjectString(d.Text),true,buf[:0],nil}
`

// TestInterfaceResultForms checks tuple bindings and ordinary caller buffers.
func TestInterfaceResultForms(t *testing.T) {
	cases := []struct{ name, source string }{
		{"short declaration", `func f(r Result){text,ok,buf,err:=r.Lookup(nil);_=text;_=ok;_=buf;_=err}`},
		{"assignment", `func f(r Result){var text im.StringProjection;var ok bool;var buf []byte;var err error;
text,ok,buf,err=r.Lookup(buf);_=text;_=ok;_=buf;_=err}`},
		{"declaration", `func f(r Result){var text,ok,buf,err=r.Lookup(nil);_=text;_=ok;_=buf;_=err}`},
		{"blanks", `func f(r Result){_,_,_,_=r.Lookup(nil)}`},
		{"reused short binding", `func f(r Result){var buf []byte;_,ok,buf,_:=r.Lookup(buf);_=ok;_=buf}`},
		{"projection", `func f(r Result){local:=NewResultProjection(im.ProjectionOf(r));_,_,_,_=local.Lookup(nil)}`},
		{"concrete method", `func f(d *Data){_,_,_,_=d.Lookup(nil)}`},
		{"helper", `func read(r Result)(im.StringProjection,bool,[]byte,error){return r.Lookup(nil)};
func f(r Result){_,_,_,_=read(r)}`},
		{"forward", `func take(im.StringProjection,bool,[]byte,error){};func f(r Result){take(r.Lookup(nil))}`},
		{"named slice and error alias", `type Buffer []byte;type Bytes=Buffer;type Failure=error;
type Reader interface{im.FrozenInterface;Read(Bytes)(int,bool,Bytes,Failure)};
func(d *Data)Read(buf Bytes)(int,bool,Bytes,Failure){return 1,true,buf,nil};
func f(){var r Reader=&Data{};_,_,buf,_:=((r).Read)(nil);_=buf}`},
		{"Copied slice", `type Count struct{im.Copied;N int};type Reader interface{im.FrozenInterface;Read([]Count)[]Count};
func(d *Data)Read(buf []Count)[]Count{return buf};func f(){var r Reader=&Data{};_=r.Read(nil)}`},
		{"clone", `type Reader interface{im.FrozenInterface;Clone()[]byte};
func(d *Data)Clone()[]byte{return d.Values.Clone()};func f(){var r Reader=&Data{};_=r.Clone()}`},
		{"several read results", `type Reader interface{im.FrozenInterface;Read()(int,bool)};
func(d *Data)Read()(int,bool){return len(d.Text),true};func f(){var r Reader=&Data{};_,_=r.Read()}`},
		{"error only", `type Reader interface{im.FrozenInterface;Read()error};
func(d *Data)Read()error{return nil};func f(){var r Reader=&Data{};_=r.Read()}`},
		{"Frozen results", `type Reader interface{im.FrozenInterface;Self()(*Data,Reader)};
func(d *Data)Self()(*Data,Reader){return d,d};func f(){var r Reader=&Data{};
p,s:=r.Self();_=p;_=s;local:=NewReaderProjection(im.ProjectionOf(r));p,s=local.Self()}`},
		{"ordinary results", `type Input struct{Value any;Callback func()};
type Reader interface{im.FrozenInterface;Read(Input)(string,any,func(),map[int]int,[][]byte)};
func(d *Data)Read(input Input)(string,any,func(),map[int]int,[][]byte){
return d.Text,input.Value,input.Callback,nil,nil};

func f(){var r Reader=&Data{};a,b,c,d,e:=r.Read(Input{});_=a;_=b;_=c;_=d;_=e}`},
		{"defined error result", `type Failure interface{Error()string};
type Reader interface{im.FrozenInterface;Read()Failure}`},
		{"record tuple helper", `func read()(Data,bool){return Data{},true};func f(){value,ok:=read();_=value;_=ok}`},
		{"user Get", `func(d *Data)Get()(Data,bool){return *d,true};func f(d *Data){value,ok:=d.Get();_=value;_=ok}`},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := checkSource(t, interfaceResultTypes+test.source); got != "" {
				t.Fatalf("allowed result form rejected: %s", got)
			}
		})
	}
}

// TestInterfaceResultBoundaries checks result proofs, binding destinations, and method bodies.
func TestInterfaceResultBoundaries(t *testing.T) {
	cases := []struct{ name, source, want string }{
		{"Local pointer", `type Local struct{im.Local;N int};type Bad interface{im.FrozenInterface;Read()*Local}`,
			"returning Local pointers"},
		{"Frozen slice", `type Bad interface{im.FrozenInterface;Read()[]Data}`, "native slices"},
		{"Local slice", `type Bad interface{im.FrozenInterface;Read()[]im.StringProjection}`, "native slices"},
		{"boxing receiver", `type Reader interface{im.FrozenInterface;Read()any};func(d *Data)Read()any{return d}`,
			"boxing immutable"},
		{"capture receiver", `type Reader interface{im.FrozenInterface;Read()func()int};
func(d *Data)Read()func()int{return func()int{return len(d.Text)}}`, "closures cannot capture"},
		{"tuple boxing", `func read()(Data,bool){return Data{},true};
func f(){var value any;var ok bool;value,ok=read();_=value;_=ok}`,
			"boxing immutable"},
		{"tuple Local package", `var output im.StringProjection;func f(r Result){output,_,_,_=r.Lookup(nil)}`,
			"multiple assignments"},
		{"tuple Freeze", `func read()([]byte,bool){return []byte{1},true};func f(){buf,_:=read();_=im.FreezeSlice(buf)}`,
			"fresh"},
		{"package declaration", `var _,_,_,_=Result(nil).Lookup(nil)`, "multiple assignments"},
		{"package assignment", `var output []byte;func f(r Result){_,_,output,_=r.Lookup(nil)}`, "multiple assignments"},
		{"element assignment", `func f(r Result){var outputs [1][]byte;_,_,outputs[0],_=r.Lookup(nil)}`,
			"multiple assignments"},
		{"field assignment", `func f(r Result){var output struct{Buf []byte};_,_,output.Buf,_=r.Lookup(nil)}`,
			"multiple assignments"},
		{"multiple expressions", `func f(r Result){a,b:=r.Lookup,1;_=a;_=b}`, "bound immutable"},
		{"function variable", `func f(call func()(im.StringProjection,bool,[]byte,error)){_,_,_,_=call()}`, "dynamic calls"},
		{"unsealed interface", `type Reader interface{Read()(im.StringProjection,bool)};
func f(r Reader){_,_=r.Read()}`, "dynamic calls"},
		{"promoted receiver", `type Outer struct{im.Frozen;Result};func f(r Outer){_,_,_,_=r.Lookup(nil)}`,
			"promoted immutable"},
		{"implicit receiver", `func(d Data)Pair()(int,bool){return 1,true};func f(d *Data){_,_=d.Pair()}`,
			"declared type exactly"},
		{"retained builder", `func(d *Data)Clone()[]byte{buf:=[]byte{1};_=im.FreezeSlice(buf);return buf}`,
			"unsupported owned container"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := checkSource(t, interfaceResultTypes+test.source); !strings.Contains(got, test.want) {
				t.Fatalf("diagnostics = %s; want %s", got, test.want)
			}
		})
	}
}
