package tunnel

import (
	"fmt"
	"io"

	"github.com/seqsense/aws-iot-device-sdk-go/v4/tunnel/msg"
)

func readProxy(ws, conn io.ReadWriter, streamID int32, eh ErrorHandler) {
	b := make([]byte, 8192)
	for {
		n, err := conn.Read(b)
		if err != nil {
			if err == io.EOF {
				return
			}
			if eh != nil {
				eh.HandleError(fmt.Errorf("connection closed: %v", err))
			}
			return
		}
		if err := msg.WriteMessage(ws, &msg.Message{
			Type:     msg.Message_DATA,
			StreamId: streamID,
			Payload:  b[:n],
		}); err != nil {
			if eh != nil {
				eh.HandleError(fmt.Errorf("message send failed: %v", err))
			}
			return
		}
	}
}
