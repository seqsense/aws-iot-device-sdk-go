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

package main

import (
	"fmt"
	"reflect"
	"strings"
)

func prettyDump(d interface{}) string {
	return prettyDumpImpl(reflect.ValueOf(d), 0)
}

func prettyDumpImpl(v reflect.Value, depth int) string {
	if depth > 1 {
		return fmt.Sprintf("\t%+v", v.Interface())
	}
	depth++
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	indent := strings.Repeat("  ", depth)
	var out string
	switch v.Kind() {
	case reflect.Slice:
		out = "\n"
		for i := 0; i < v.Len(); i++ {
			out += indent + "- " + prettyDumpImpl(v.Index(i), depth) + "\n"
		}
	case reflect.Map:
		out = "\n"
		for _, k := range v.MapKeys() {
			out += indent + k.String() + ": " + prettyDumpImpl(v.MapIndex(k), depth) + "\n"
		}
		return out
	case reflect.Struct:
		out = "\n"
		for i := 0; i < v.NumField(); i++ {
			out += indent + v.Type().Field(i).Name + ": " + prettyDumpImpl(v.Field(i), depth) + "\n"
		}
	default:
		if !v.IsValid() {
			return ""
		}
		return fmt.Sprintf("\t%+v", v.Interface())
	}
	return strings.Replace(out, "\n\n", "\n", -1)
}
