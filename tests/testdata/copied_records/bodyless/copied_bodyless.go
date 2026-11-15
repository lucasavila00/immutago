package bodyless

import "example.com/consumer/api"

type Holder struct {
	ID    string
	Point api.Point
}

type Local api.Point

func decode(*api.Point) error
func decodeHolder(*Holder) error
func pair(api.Point) (api.Point, bool)
func (*Local) Decode([]byte) error
