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
		{"raw record tuple", `type Bad interface{im.FrozenInterface;Read()(int,Data)}`, "Read result"},
		{"raw pointer", `type Bad interface{im.FrozenInterface;Read()*Data}`, "Read result"},
		{"raw interface tuple", `type Bad interface{im.FrozenInterface;Read()(bool,Result)}`, "Read result"},
		{"nested slice", `type Bad interface{im.FrozenInterface;Read()[][]byte}`, "Read result"},
		{"string slice", `type Bad interface{im.FrozenInterface;Read()[]string}`, "Read result"},
		{"Frozen slice", `type Bad interface{im.FrozenInterface;Read()[]Data}`, "Read result"},
		{"Local slice", `type Bad interface{im.FrozenInterface;Read()[]im.StringProjection}`, "Read result"},
		{"pointer slice", `type Bad interface{im.FrozenInterface;Read()[]*int}`, "Read result"},
		{"native map", `type Bad interface{im.FrozenInterface;Read()map[int]int}`, "Read result"},
		{"dynamic result", `type Bad interface{im.FrozenInterface;Read()any}`, "Read result"},
		{"callback result", `type Bad interface{im.FrozenInterface;Read()func()}`, "Read result"},
		{"defined error", `type Failure interface{Error()string};type Bad interface{im.FrozenInterface;Read()Failure}`, "Read result"},
		{"cell", `type Bad interface{im.FrozenInterface;Read()im.Cell[int]}`, "Read result"},
		{"package declaration", `var _,_,_,_=Result(nil).Lookup(nil)`, "multiple assignments"},
		{"package assignment", `var output []byte;func f(r Result){_,_,output,_=r.Lookup(nil)}`, "multiple assignments"},
		{"element assignment", `func f(r Result){var outputs [1][]byte;_,_,outputs[0],_=r.Lookup(nil)}`, "multiple assignments"},
		{"field assignment", `func f(r Result){var output struct{Buf []byte};_,_,output.Buf,_=r.Lookup(nil)}`, "multiple assignments"},
		{"multiple expressions", `func f(r Result){a,b:=r.Lookup,1;_=a;_=b}`, "bound immutable"},
		{"function variable", `func f(call func()(im.StringProjection,bool,[]byte,error)){_,_,_,_=call()}`, "dynamic calls"},
		{"unsealed interface", `type Reader interface{Read()(im.StringProjection,bool)};
func f(r Reader){_,_=r.Read()}`, "dynamic calls"},
		{"promoted receiver", `type Outer struct{im.Frozen;Result};func f(r Outer){_,_,_,_=r.Lookup(nil)}`, "promoted immutable"},
		{"implicit receiver", `func(d Data)Pair()(int,bool){return 1,true};func f(d *Data){_,_=d.Pair()}`, "declared type exactly"},
		{"retained builder", `func(d *Data)Clone()[]byte{buf:=[]byte{1};_=im.FreezeSlice(buf);return buf}`, "unsupported owned container"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := checkSource(t, interfaceResultTypes+test.source); !strings.Contains(got, test.want) {
				t.Fatalf("diagnostics = %s; want %s", got, test.want)
			}
		})
	}
}
