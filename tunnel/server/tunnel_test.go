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

package server

import (
	"reflect"
	"testing"
)

func TestAddRemove(t *testing.T) {
	h := NewTunnelHandler()

	tis := []*tunnelInfo{
		{
			thingName:       "t1",
			destAccessToken: "token1",
			srcAccessToken:  "token2",
			cancel:          func() {},
		},
		{
			thingName:       "tRemoved",
			destAccessToken: "tokenRemoved1",
			srcAccessToken:  "tokenRemoved2",
			cancel:          func() {},
		},
		{
			thingName:       "t2",
			destAccessToken: "token3",
			srcAccessToken:  "token4",
			cancel:          func() {},
		},
	}
	for _, ti := range tis {
		_, err := h.add(ti)
		if err != nil {
			t.Fatal(err)
		}
	}
	h.remove("00000001")

	tunnelsExpected := map[string]*tunnelInfo{
		"00000000": tis[0],
		"00000002": tis[2],
	}
	destTokenExpected := map[string]*tunnelInfo{
		"token1": tis[0],
		"token3": tis[2],
	}
	srcTokenExpected := map[string]*tunnelInfo{
		"token2": tis[0],
		"token4": tis[2],
	}

	if !reflect.DeepEqual(tunnelsExpected, h.tunnels) {
		t.Errorf("Expected tunnels: %v, got: %v", tunnelsExpected, h.tunnels)
	}
	if !reflect.DeepEqual(destTokenExpected, h.destToken) {
		t.Errorf("Expected tunnels: %v, got: %v", destTokenExpected, h.destToken)
	}
	if !reflect.DeepEqual(srcTokenExpected, h.srcToken) {
		t.Errorf("Expected tunnels: %v, got: %v", srcTokenExpected, h.srcToken)
	}
}
