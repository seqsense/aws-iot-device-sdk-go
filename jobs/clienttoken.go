package jobs

import (
	"reflect"
)

func clientToken(i interface{}) (string, bool) {
	v := reflect.ValueOf(i).Elem().FieldByName("ClientToken")
	if !v.IsValid() {
		return "", false
	}
	if v.Kind() != reflect.String {
		return "", false
	}
	return v.String(), true
}
