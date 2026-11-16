package diagnosticcontext

import "github.com/lucasavila00/immutago/pkg/im"

type TextScore struct {
	im.Copied
	Label string
}

type ProjectionScore struct {
	im.Copied
	Label im.StringProjection
}

type FrozenScore struct {
	im.Copied
	Label im.FrozenString
}

type Reply struct {
	im.Local
	Name im.StringProjection
}

type Summary struct {
	im.Copied
	Count int
	Reply Reply
}

type Nested struct {
	im.Copied
	Scores [2]ProjectionScore
}

type Counts [2]int

type NamedArray struct {
	im.Copied
	Values Counts
}

type Marker = im.Frozen

type MarkerData struct {
	im.Copied
	Markers [0]Marker
}

type InterfaceMarkerData struct {
	im.Local
	Markers [0]im.FrozenInterface
}

type InterfaceMarker = im.FrozenInterface

type InterfaceReply struct {
	im.Local
	Model InterfaceMarker
}

type InterfaceSummary struct {
	im.Copied
	Count  int
	Models [0]InterfaceMarker
}

type NestedInterfaceReply struct {
	im.Local
	Replies [2]InterfaceReply
}

type NestedInterfaceSummary struct {
	im.Copied
	Summaries [2]InterfaceSummary
}
