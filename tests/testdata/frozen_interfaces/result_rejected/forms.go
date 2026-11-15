package result_rejected

import (
	"example.com/consumer/api"
	"example.com/consumer/implementation"
	"github.com/lucasavila00/immutago/pkg/im"
)

var buffer []byte
var reader api.Reader
var _, _, _, _ = reader.Lookup(nil, nil)

func rejected(r api.Reader) {
	_, _, buffer, _ = r.Lookup(nil, nil)
	var outputs [1][]byte
	_, _, outputs[0], _ = r.Lookup(nil, nil)
	var output struct{ Buf []byte }
	_, _, output.Buf, _ = r.Lookup(nil, nil)
	var boxed any
	var ok bool
	boxed, ok = pair()
	_ = boxed
	_ = ok
	_ = r.Lookup
	go r.Lookup(nil, nil)
	defer r.Lookup(nil, nil)
}

func pair() (*implementation.Tree, bool) { return &implementation.Tree{}, true }

type Dynamic interface {
	Read() (im.StringProjection, bool)
}

func dynamic(r Dynamic) { _, _ = r.Read() }

func retainedBuilder() []byte {
	values := []byte{1}
	_ = im.FreezeSlice(values)
	return values
}

// frozenWrite retains the source rules after a declared pointer result.
func frozenWrite() {
	var reader implementation.TreeReader = &implementation.Tree{}
	local := implementation.NewTreeReaderProjection(im.ProjectionOf(reader))
	pointer := local.TreePointer()
	pointer.Weight = 2
}
