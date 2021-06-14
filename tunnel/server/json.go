// Copyright 2021 SEQSENSE, Inc.
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

package server

import (
	"encoding/json"
	"reflect"
	"strings"
	"unicode"
)

type response struct {
	value interface{}
}

func (j *response) MarshalJSON() ([]byte, error) {
	v := reflect.ValueOf(j.value)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	t := v.Type()
	n := v.NumField()
	b := make([]byte, 0, 64)
	b = append(b, '{')
	var continued bool
	for i := 0; i < n; i++ {
		sf, v := t.Field(i), v.Field(i)
		if !unicode.IsUpper([]rune(sf.Name)[0]) {
			continue
		}
		name := strings.ToLower(sf.Name[0:1]) + sf.Name[1:]
		if continued {
			b = append(b, ',')
		} else {
			continued = true
		}
		b = append(b, '"')
		b = append(b, []byte(name)...)
		b = append(b, []byte("\":")...)
		child, err := json.Marshal(v.Interface())
		if err != nil {
			return nil, err
		}
		b = append(b, child...)
	}
	b = append(b, '}')

	return b, nil
}
