package tunnel

import (
	"testing"
)

func TestClientMode_String(t *testing.T) {
	str := "test_string"
	m := ClientMode(str)
	if m.String() != str {
		t.Error("ClientMode.String() differs from its value")
	}
}
