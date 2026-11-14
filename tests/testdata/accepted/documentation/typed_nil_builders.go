package main

import "github.com/lucasavila00/immutago/pkg/im"

type nilInts []int
type nilScores map[string]int
type nilIntsAlias = nilInts
type nilScoresAlias = nilScores

type nilShared struct {
	im.Shared
	N int
}

type nilLocal struct {
	im.LocalRef
	N int
}

// typedNilBuilders constructs empty builders without copying an existing handle.
func typedNilBuilders() {
	values := []int(nil)
	values = append(values, 1)
	frozen := im.FreezeSlice(values)
	println(frozen.At(0))

	scores := map[string]int(nil)
	clear(scores)
	_ = im.FreezeMap(scores)

	var named = nilInts(nil)
	named = append(named, 2)
	_ = im.FreezeSlice(named)
	alias := (nilScoresAlias)((nil))
	_ = im.FreezeMap(alias)

	_ = im.FreezeSlice((nilIntsAlias)((nil)))
	_ = im.FreezeMap(nilScores(nil))
	_ = im.FreezeSlice([]int(nil))
	_ = im.FreezeMap(map[string]int(nil))
}

// typedNilProtectedBuilders still use the normal element and ownership checks.
func typedNilProtectedBuilders() {
	values := ([]nilShared)((nil))
	values = append(values, nilShared{N: 1})
	_ = im.FreezeSlice(values)
	pointers := []*nilShared(nil)
	pointers = append(pointers, &nilShared{N: 2})
	_ = im.FreezeSlice(pointers)
	locals := []nilLocal(nil)
	locals = append(locals, nilLocal{N: 3})
	_ = im.FreezeSlice(locals)
	entries := map[*nilShared]nilLocal(nil)
	_ = im.FreezeMap(entries)
	_ = im.FreezeSlice([]nilShared(nil))
	_ = im.FreezeSlice([]*nilShared(nil))
	_ = im.FreezeMap((map[*nilShared]nilLocal)((nil)))
	unfrozen := []nilShared(nil)
	unfrozen = append(unfrozen, nilShared{N: 4})
	println(unfrozen[0].N)
}
