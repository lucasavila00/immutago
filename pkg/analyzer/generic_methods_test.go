package analyzer

import (
	"strings"
	"testing"
)

const genericValueMethod = `func(v Value)Map[T any](read func(int)T)T{return read(v.X)};`
const genericPointerMethod = `func(p *Value)MapPointer[T any](read func(int)T)T{return read(p.X)};`

// TestGenericFrozenMethods uses the same generic rules for functions and both receiver forms.
func TestGenericFrozenMethods(t *testing.T) {
	cases := map[string]string{
		"value inferred":   genericValueMethod + `func f(v Value)int{return v.Map(number)}`,
		"value explicit":   genericValueMethod + `func f(v Value)int{return v.Map[int](number)}`,
		"pointer inferred": genericPointerMethod + `func f(p *Value)int{return p.MapPointer(number)}`,
		"pointer explicit": genericPointerMethod + `func f(p *Value)int{return p.MapPointer[int](number)}`,
		"free function": `func mapValue[T any](v Value,read func(int)T)T{return read(v.X)};
func f(v Value)int{return mapValue(v,number)}`,
		"Frozen result": genericValueMethod + `func value(n int)Value{return Value{X:n}};
func f(v Value)Value{return v.Map[Value](value)}`,
		"Frozen constraint": `func(v Value)Keep[T Value](value T)T{return value};
func f(v Value)Value{return v.Keep(v)}`,
		"method value":      genericValueMethod + `func f(v Value)int{read:=v.Map[int];return read(number)}`,
		"method expression": genericValueMethod + `func f(v Value)int{return Value.Map[int](v,number)}`,
		"alias and parentheses": genericValueMethod + `type Alias=Value;
func f(v Alias)int{return ((v).Map[int])(number)}`,
		"alias receiver": `type Alias=Value;func(v Alias)Map[T any](read func(int)T)T{return read(v.X)};
func f(v Alias)int{return v.Map[int](number)}`,
		"protected field": genericValueMethod + `type Root struct{im.Frozen;Child Value};
func f(root *Root)int{return root.Child.Map[int](number)}`,
		"builder element": genericValueMethod + `func f()int{values:=[]Value{{X:1}};
out:=values[0].Map[int](number);_=im.FreezeSlice(values);return out}`,
		"Copied field": `type Count struct{im.Copied;N int};type Root struct{im.Frozen;Count Count};
func(c Count)Map[T any](read func(int)T)T{return read(c.N)};
func f(root *Root)int{return root.Count.Map[int](number)}`,
	}
	for name, source := range cases {
		t.Run(name, func(t *testing.T) {
			if got := checkSource(t, receiverTestTypes+`func number(n int)int{return n};`+source); got != "" {
				t.Fatal(got)
			}
		})
	}
}

// TestGenericMethodBoundaries keeps Local transport, receiver, and body checks.
func TestGenericMethodBoundaries(t *testing.T) {
	cases := map[string]struct{ source, want string }{
		"Local value receiver": {`func(v Local)Map[T any](read func(int)T)T{return read(v.X)};
func f(v Local)int{return v.Map[int](number)}`, "user generic declarations"},
		"Local pointer receiver": {`func(p *Local)Map[T any](read func(int)T)T{return read(p.X)};
func f(p *Local)int{return p.Map[int](number)}`, "user generic declarations"},
		"Local parameter": {`func(v Value)ReadLocal[T any](local Local)int{return local.X}`,
			"user generic declarations"},
		"Local result": {`func(v Value)LocalResult[T any](value T)Local{return Local{X:v.X}}`,
			"user generic declarations"},
		"Local constraint": {`func(v Value)Keep[T Local](value T)T{return value}`, "user generic declarations"},
		"Local type argument": {genericValueMethod + `func local(n int)Local{return Local{X:n}};
func f(v Value){_=v.Map[Local](local)}`, "user generic instantiations"},
		"Local callback": {`func(v Value)ReadCallback[T any](read func(Local)T)T{return read(Local{X:v.X})}`,
			"user generic declarations"},
		"value field write":   {`func(v Value)Change[T any](value T){v.X++}`, "writes to immutable"},
		"pointer field write": {`func(p *Value)Change[T any](value T){p.X++}`, "writes to immutable"},
		"receiver address":    {`func(v Value)Address[T any](value T)*Value{return &v}`, "fresh allocation"},
		"implicit address": {genericPointerMethod + `func f(v Value)int{return v.MapPointer[int](number)}`,
			"declared type exactly"},
		"generic receiver type": {`type Box[T any]struct{im.Frozen;X int};func(v Box[T])Read()int{return v.X}`,
			"user generic declarations cannot define Frozen or Local types"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := checkSource(t, receiverTestTypes+`func number(n int)int{return n};`+tc.source)
			if !strings.Contains(got, tc.want) {
				t.Fatalf("diagnostics = %s; want %s", got, tc.want)
			}
		})
	}
}
