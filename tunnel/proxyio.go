package tunnel

import (
	"fmt"
	"io"

	"google.golang.org/protobuf/proto"

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
		ms := &msg.Message{
			Type:     msg.Message_DATA,
			StreamId: streamID,
			Payload:  b[:n],
		}
		bs, err := proto.Marshal(ms)
		if err != nil {
			if eh != nil {
				eh.HandleError(fmt.Errorf("marshal failed: %v", err))
			}
			continue
		}
		l := len(bs)
		if _, err := ws.Write(
			append([]byte{
				byte(l >> 8), byte(l),
			}, bs...),
		); err != nil {
			if eh != nil {
				eh.HandleError(fmt.Errorf("send failed: %v", err))
			}
			return
		}
	}
}
