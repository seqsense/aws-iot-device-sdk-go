package msg

import (
	"io"

	"google.golang.org/protobuf/proto"
)

// WriteMessage marshals the message and sends to the writer.
func WriteMessage(w io.Writer, m *Message) error {
	bs, err := proto.Marshal(m)
	if err != nil {
		return err
	}
	l := len(bs)
	b := append([]byte{byte(l >> 8), byte(l)}, bs...)
	for p := 0; p < len(b); {
		n, err := w.Write(b[p:])
		if err != nil {
			return err
		}
		p += n
	}
	return nil
}
