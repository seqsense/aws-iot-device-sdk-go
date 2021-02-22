package shadow

import (
	"errors"
	"reflect"
)

func stateDiff(base, in interface{}) (interface{}, bool) {
	keys, hasChild := attributeKeys(base)
	if !hasChild {
		if reflect.DeepEqual(base, in) {
			return nil, false
		}
		return in, true
	}
	keysIn, hasChildIn := attributeKeys(in)
	if !hasChildIn {
		return in, true
	}
	keysInMap := make(map[string]struct{})
	for _, k := range keysIn {
		keysInMap[k] = struct{}{}
	}
	out := make(map[string]interface{})
	for _, k := range keys {
		if _, ok := keysInMap[k]; !ok {
			continue
		}
		delete(keysInMap, k)
		a, err := attributeByKey(base, k)
		if err != nil {
			return in, true
		}
		b, err := attributeByKey(in, k)
		if err != nil {
			continue
		}
		d, difer := stateDiff(a, b)
		if difer {
			out[k] = d
		}
	}
	for k := range keysInMap {
		b, err := attributeByKey(in, k)
		if err != nil {
			continue
		}
		out[k] = b
	}
	if len(out) == 0 {
		return nil, false
	}
	return out, true
}

var errInvalidAttribute = errors.New("invalid attribute key")

func attributeKeys(a interface{}) ([]string, bool) {
	v := reflect.ValueOf(a)
	if !v.IsValid() {
		return nil, false
	}
	t := v.Type()
	switch t.Kind() {
	case reflect.Map:
		out := make([]string, v.Len())
		keys := v.MapKeys()
		for i := range out {
			out[i] = keys[i].String()
		}
		return out, true
	case reflect.Struct:
		out := make([]string, v.NumField())
		for i := range out {
			out[i] = t.Field(i).Name
		}
		return out, true
	case reflect.Ptr:
		return attributeKeys(v.Elem().Interface())
	}
	return nil, false
}

func attributeByKey(a interface{}, k string) (interface{}, error) {
	v := reflect.ValueOf(a)
	if !v.IsValid() {
		return nil, errInvalidAttribute
	}
	t := v.Type()
	switch t.Kind() {
	case reflect.Struct:
		return v.FieldByName(k).Interface(), nil
	case reflect.Map:
		val := v.MapIndex(reflect.ValueOf(k))
		if !val.IsValid() {
			return nil, nil
		}
		return val.Interface(), nil
	case reflect.Ptr:
		return attributeByKey(v.Elem().Interface(), k)
	}
	return nil, errInvalidAttribute
}
