package reflection

import "reflect"

func Len(v any) int {
	val := reflect.ValueOf(v)
	valKind := val.Kind()

	if valKind == reflect.Ptr {
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map, reflect.String, reflect.Chan:
		return val.Len()
	default:
		return 0
	}
}
