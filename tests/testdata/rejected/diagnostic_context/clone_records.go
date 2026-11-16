package diagnosticcontext

import "github.com/lucasavila00/immutago/pkg/im"

func cloneRecords(records im.MapProjection[int, Record], pairs im.SliceProjection[[2]Settings],
	mapPairs im.MapProjection[int, [2]Settings], keys im.MapProjection[[0]Record, int]) {
	_ = records.Clone()
	_ = pairs.Clone()
	_ = mapPairs.Clone()
	_ = keys.Clone()
}

func cloneHandles(pointers im.SliceProjection[*Settings], interfaces im.SliceProjection[Scorer],
	mapPointers im.MapProjection[int, *Settings], mapInterfaces im.MapProjection[int, Scorer]) {
	_ = pointers.Clone()
	_ = interfaces.Clone()
	_ = mapPointers.Clone()
	_ = mapInterfaces.Clone()
}

func cloneBindings(records im.SliceProjection[Record]) {
	short := records.Clone()
	var declared = (records.Clone)()
	var assigned []Settings
	assigned = records.Clone()
	_ = len(short)
	_ = len(declared)
	_ = len(assigned)
}

type Score struct {
	im.Copied
	Value int
}

func cloneAfterFreeze(records im.SliceProjection[Score]) {
	values := records.Clone()
	_ = im.FreezeSlice(values)
	_ = len(values)
}

func cloneCapture(records im.SliceProjection[Score]) {
	_ = func() int { return len(records.Clone()) }
}

type ProjectionHolder struct {
	im.Local
	im.SliceProjection[int]
}

func clonePromoted(values ProjectionHolder) {
	_ = values.Clone()
}

func cloneExpression(values im.SliceProjection[int]) {
	_ = im.SliceProjection[int].Clone(values)
}
