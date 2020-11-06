package msg

import (
	"bytes"
	"errors"
	"testing"

	"github.com/seqsense/aws-iot-device-sdk-go/v4/internal/ioterr"
)

func TestWriteMessage(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		m := &Message{Type: Message_STREAM_START}
		buf := &bytes.Buffer{}

		if err := WriteMessage(buf, m); err != nil {
			t.Fatal(err)
		}
		expected := []byte{0x02}
		if bytes.Equal(expected, buf.Bytes()) {
			t.Errorf("Expected: %v, got: %v", expected, buf.Bytes())
		}
	})
	t.Run("WriterError", func(t *testing.T) {
		m := &Message{Type: Message_STREAM_START}

		errDummy := errors.New("write error")
		buf := writerFunc(func(b []byte) (int, error) {
			return 0, errDummy
		})

		err := WriteMessage(buf, m)
		if !errors.Is(err, errDummy) {
			t.Errorf("Expected error: %v, got: %v", errDummy, err)
		}
		var ie *ioterr.Error
		if !errors.As(err, &ie) {
			t.Errorf("Expected error type: ioterr.Error, actual: %T", err)
		}
	})
}

type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(b []byte) (int, error) {
	return f(b)
}
