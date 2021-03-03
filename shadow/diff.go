// Copyright 2020-2021 SEQSENSE, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package shadow

import (
	"errors"
	"reflect"
)

var errInvalidAttribute = errors.New("invalid attribute key")

func stateDiff(base, in interface{}) (interface{}, bool, error) {
	keys, hasChild, err := attributeKeys(base)
	if err != nil {
		return nil, false, err
	}
	if !hasChild {
		baseVal := reflect.ValueOf(base)
		inVal := reflect.ValueOf(in)
		if !baseVal.IsValid() {
			if !inVal.IsValid() {
				// Both nil
				return nil, false, nil
			}
			// One is nil
			return in, true, nil
		} else if !inVal.IsValid() {
			// One is nil
			return in, true, nil
		}
		switch baseVal.Type().Kind() {
		case reflect.Array, reflect.Slice:
			// Compare slice/array
			switch inVal.Type().Kind() {
			case reflect.Array, reflect.Slice:
				if baseVal.Len() != inVal.Len() {
					// Size differs
					return in, true, nil
				}
				for i := 0; i < baseVal.Len(); i++ {
					_, hasDiff, err := stateDiff(
						baseVal.Index(i).Interface(),
						inVal.Index(i).Interface(),
					)
					if err != nil {
						return nil, false, err
					}
					if hasDiff {
						return in, true, nil
					}
				}
			default:
				// Type differs
				return in, true, nil
			}
			return nil, false, nil
		default:
			// Compare primitive value
			if reflect.DeepEqual(base, in) {
				return nil, false, nil
			}
		}
		return in, true, nil
	}

	keysIn, hasChildIn, err := attributeKeys(in)
	if err != nil {
		return nil, false, err
	}
	if !hasChildIn {
		return in, true, nil
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
		a, _ := attributeByKey(base, k)
		b, err := attributeByKey(in, k)
		if err != nil {
			return nil, false, err
		}
		d, difer, err := stateDiff(a, b)
		if err != nil {
			return nil, false, err
		}
		if difer {
			out[k] = d
		}
	}
	for k := range keysInMap {
		b, _ := attributeByKey(in, k)
		out[k] = b
	}
	if len(out) == 0 {
		return nil, false, nil
	}
	return out, true, nil
}

func attributeKeys(a interface{}) ([]string, bool, error) {
	v := reflect.ValueOf(a)
	if !v.IsValid() {
		return nil, false, nil
	}
	t := v.Type()
	switch t.Kind() {
	case reflect.Map:
		out := make([]string, v.Len())
		keys := v.MapKeys()
		for i := range out {
			if keys[i].Kind() != reflect.String {
				return nil, false, ErrUnsupportedMapKeyType
			}
			out[i] = keys[i].String()
		}
		return out, true, nil
	case reflect.Struct:
		out := make([]string, v.NumField())
		for i := range out {
			out[i] = t.Field(i).Name
		}
		return out, true, nil
	case reflect.Ptr:
		return attributeKeys(v.Elem().Interface())
	}
	return nil, false, nil
}

func attributeByKey(a interface{}, k string) (interface{}, error) {
	ret, err := attributeByKeyImpl(reflect.ValueOf(a), k)
	if err != nil {
		return nil, err
	}
	return ret.Interface(), nil
}

func attributeByKeyImpl(v reflect.Value, k string) (reflect.Value, error) {
	if !v.IsValid() {
		return reflect.Value{}, errInvalidAttribute
	}
	t := v.Type()
	switch t.Kind() {
	case reflect.Struct:
		return v.FieldByName(k), nil
	case reflect.Map:
		val := v.MapIndex(reflect.ValueOf(k))
		if !val.IsValid() {
			return reflect.Value{}, nil
		}
		return val, nil
	case reflect.Ptr, reflect.Interface:
		return attributeByKeyImpl(v.Elem(), k)
	}
	return reflect.Value{}, errInvalidAttribute
}
