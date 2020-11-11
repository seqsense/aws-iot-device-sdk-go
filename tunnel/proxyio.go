// Copyright 2020 SEQSENSE, Inc.
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
