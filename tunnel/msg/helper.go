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

package msg

import (
	"io"

	"google.golang.org/protobuf/proto"

	"github.com/seqsense/aws-iot-device-sdk-go/v4/internal/ioterr"
)

// WriteMessage marshals the message and sends to the writer.
func WriteMessage(w io.Writer, m *Message) error {
	bs, err := proto.Marshal(m)
	if err != nil {
		return ioterr.New(err, "marshaling message")
	}
	l := len(bs)
	b := append([]byte{byte(l >> 8), byte(l)}, bs...)
	for p := 0; p < len(b); {
		n, err := w.Write(b[p:])
		if err != nil {
			return ioterr.New(err, "writing message")
		}
		p += n
	}
	return nil
}
