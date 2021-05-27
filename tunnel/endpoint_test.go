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
	"testing"
)

func TestServerNameFromEndpoint(t *testing.T) {
	testCases := []struct {
		scheme string
		input  string
		output string
	}{
		{
			scheme: "wss",
			input:  "hostname:123",
			output: "hostname:123",
		},
		{
			scheme: "wss",
			input:  "hostname:443",
			output: "hostname",
		},
		{
			scheme: "wss",
			input:  "hostname",
			output: "hostname",
		},
		{
			scheme: "ws",
			input:  "hostname:443",
			output: "hostname:443",
		},
		{
			scheme: "ws",
			input:  "hostname:80",
			output: "hostname",
		},
		{
			scheme: "ws",
			input:  "hostname",
			output: "hostname",
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(fmt.Sprintf("%s://%s", testCase.scheme, testCase.input), func(t *testing.T) {
			out := serverNameFromEndpoint(testCase.scheme, testCase.input)
			if out != testCase.output {
				t.Errorf(
					"Expected ServerName %s for endpoint %s scheme %s, got %s",
					testCase.output,
					testCase.input,
					testCase.scheme,
					out,
				)
			}
		})
	}
}
