package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Marker
	X int
}

func hiddenStruct() { var value Good; holder := struct{ P *int }{P: &value.X}; *holder.P = 2 }     // want "borrow|shared reference|unsupported"
func hiddenSlice()  { var value Good; holder := []*int{&value.X}; *holder[0] = 2 }                 // want "borrow|shared reference|unsupported"
func hiddenMap()    { var value Good; holder := map[string]*int{"x": &value.X}; *holder["x"] = 2 } // want "borrow|shared reference|unsupported"
