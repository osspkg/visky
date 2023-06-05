package app

import (
	"fmt"
	"reflect"

	"github.com/deweppro/go-sdk/random"
)

var errType = reflect.TypeOf(new(error)).Elem()

func getRefAddr(t reflect.Type) (string, bool) {
	if len(t.PkgPath()) > 0 {
		return t.PkgPath() + "." + t.Name(), true
	}
	switch t.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Ptr, reflect.Slice:
		if t.Implements(errType) {
			return "error", false
		}
		if len(t.Elem().PkgPath()) > 0 {
			return t.Elem().PkgPath() + "." + t.Elem().Name(), true
		}
	case reflect.Func:
		return random.String(30) + "." + t.String(), true
	}
	return t.String(), false
}

func typingRefPtr(vv []interface{}, call func(interface{}) error) ([]interface{}, error) {
	result := make([]interface{}, 0, len(vv))
	for _, v := range vv {
		ref := reflect.TypeOf(v)
		switch ref.Kind() {
		case reflect.Struct:
			in := reflect.New(ref).Interface()
			if err := call(in); err != nil {
				return nil, err
			}
			rv := reflect.ValueOf(in).Elem().Interface()
			result = append(result, rv)
		case reflect.Ptr:
			if err := call(v); err != nil {
				return nil, err
			}
			result = append(result, v)
		default:
			return nil, fmt.Errorf("supported type [%T]", v)
		}
	}
	return result, nil
}
