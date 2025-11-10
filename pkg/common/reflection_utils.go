package common

import "reflect"

func Len(v any) int {
	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map, reflect.String, reflect.Chan:
		return val.Len()
	default:
		return 0
	}
}
