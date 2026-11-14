package logx

import (
	"fmt"
	"reflect"
)

var Saved any

func Info(message string, value any) { fmt.Println(message, reflect.TypeOf(value).Kind(), value) }
func Values(first, second any)       { fmt.Println(first, second) }
func Identity[T any](value T) T      { return value }
