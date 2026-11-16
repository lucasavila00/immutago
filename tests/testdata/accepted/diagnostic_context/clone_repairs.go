package diagnosticcontext

import "github.com/lucasavila00/immutago/pkg/im"

func limits(records im.SliceProjection[Settings]) []int {
	out := make([]int, 0, records.Len())
	for index := range records.All() {
		out = append(out, NewSettingsProjection(records.AtPointer(index)).Limit())
	}
	return out
}

func mapLimits(records im.MapProjection[int, Settings]) map[int]int {
	out := make(map[int]int, records.Len())
	for key, value := range records.All() {
		out[im.Value(key)] = NewSettingsProjection(im.CopyRecord(value)).Limit()
	}
	return out
}

func pairLimits(pairs im.SliceProjection[[2]Settings]) [][2]int {
	out := make([][2]int, pairs.Len())
	for index, pair := range pairs.All() {
		values := pair.Unsafe()
		out[index] = [2]int{values[0].Limit, values[1].Limit}
	}
	return out
}

func pointerLimits(records im.SliceProjection[*Settings]) []int {
	out := make([]int, records.Len())
	for index, value := range records.All() {
		out[index] = NewSettingsProjection(value).Limit()
	}
	return out
}

func mapScores(records im.MapProjection[int, Scorer]) map[int]int {
	out := make(map[int]int, records.Len())
	for key, value := range records.All() {
		out[im.Value(key)] = NewScorerProjection(value).Score()
	}
	return out
}

func rebuild(records im.SliceProjection[Settings]) im.FrozenSlice[Settings] {
	fresh := make([]Settings, 0, records.Len())
	for _, record := range records.All() {
		fresh = append(fresh, record.Unsafe())
	}
	frozen := im.FreezeSlice(fresh)
	return frozen
}

func rebuildMap(records im.MapProjection[int, Settings]) im.FrozenMap[int, Settings] {
	fresh := make(map[int]Settings, records.Len())
	for key, record := range records.All() {
		fresh[im.Value(key)] = record.Unsafe()
	}
	frozen := im.FreezeMap(fresh)
	return frozen
}

func cloneCopied(records im.SliceProjection[Score], pairs im.SliceProjection[[2]Score],
	lookup im.MapProjection[Score, [2]Score]) {
	values := records.Clone()
	alias := values
	if len(alias) > 0 {
		alias[0].Value++
	}
	_ = pairs.Clone()
	_ = lookup.Clone()
}

func cloneThenFreeze(records im.SliceProjection[Score]) im.FrozenSlice[Score] {
	values := records.Clone()
	frozen := im.FreezeSlice(values)
	return frozen
}

type ProjectionHolder struct {
	im.Local
	im.SliceProjection[int]
}

func cloneExplicit(values ProjectionHolder) []int {
	return values.SliceProjection.Clone()
}

func cloneDirect(values im.SliceProjection[int]) []int {
	return values.Clone()
}
