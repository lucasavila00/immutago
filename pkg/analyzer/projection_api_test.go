package analyzer

import "testing"

// TestProjectionEntryPoints checks native strings, frozen values, and generic projection adapters.
func TestProjectionEntryPoints(t *testing.T) {
	for _, source := range []string{
		`type Label string;func f(s Label){_=im.ProjectString(s).Len()}`,
		`func f(){_=((im.ProjectString[string]))("text").EqualString("text")}`,
		`type Text=im.FrozenString;func f(value Text){_=value.Projection().Len()}`,
		`type Values=im.FrozenSlice[int];func f(value Values){_=value.Projection().Len()}`,
		`type Values=im.FrozenMap[string,int];func f(value Values){_=value.Projection().Len()}`,
		`func f(value im.Projection[string]){_=im.ProjectedString(value).Len()}`,
		`func f(value im.Projection[im.FrozenString]){_=im.ProjectedFrozenString(value).Len()}`,
		`func f(value im.Projection[im.FrozenSlice[int]]){_=im.ProjectedSlice(value).Len()}`,
		`func f(value im.Projection[im.FrozenMap[string,int]]){_=im.ProjectedMap(value).Len()}`,
		`type Reply struct{im.Local;Text im.StringProjection}
func f(text string)Reply{return Reply{Text:im.ProjectString(text)}}`,
		`type Ordinary struct{};func(Ordinary)Projection()[]int{return nil}
func f(){_=im.ProjectString("text");ordinary:=Ordinary{};_=ordinary.Projection()}`,
	} {
		if got := checkSource(t, source); got != "" {
			t.Fatal(got)
		}
	}
}

// TestProjectionEntryPointBoundaries keeps each new entry point inside the same Local rules.
func TestProjectionEntryPointBoundaries(t *testing.T) {
	for _, source := range []string{
		`var saved=im.ProjectString("text")`,
		`var saved=im.FreezeString("text").Projection()`,
		`type Saved struct{im.Frozen;Text im.StringProjection}`,
		`func f(text im.FrozenString){projection:=text.Projection();_=func()int{return projection.Len()}}`,
		`func f(text im.FrozenString){_=any(text.Projection())}`,
		`func f(text im.FrozenString){method:=text.Projection;_=method}`,
		`func f(text im.FrozenString){_=im.FrozenString.Projection(text)}`,
		`type Wrapped struct{im.Frozen;im.FrozenSlice[int]};func f(value Wrapped){_=value.Projection()}`,
		`func f(){project:=im.ProjectString[string];_=project("text")}`,
		`func f(){go im.ProjectString("text")}`,
		`func f(text im.FrozenString){defer text.Projection()}`,
	} {
		if got := checkSource(t, source); got == "" {
			t.Fatalf("unsupported source passed: %s", source)
		}
	}
}
