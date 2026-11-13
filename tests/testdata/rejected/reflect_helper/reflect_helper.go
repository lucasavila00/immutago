package reflect_helper

import "reflect" // want "unsupported reflect import"

func Value() reflect.Value { return reflect.ValueOf(1) }
