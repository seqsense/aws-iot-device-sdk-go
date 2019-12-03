// Copyright 2018 SEQSENSE, Inc.
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

package awsiotprotocol

import (
	"net/url"
	"strings"
	"testing"
)

func TestByURL(t *testing.T) {
	testcase := []struct {
		input    string
		expected Protocol
	}{
		{input: "mqtts://example.com", expected: Mqtts{}},
		{input: "wss://example.com", expected: Wss{}},
	}
	for _, v := range testcase {
		actual, _ := ByURL(v.input)
		if actual != v.expected {
			t.Errorf("awsiotprotocol.ByURL failed.\ninput: %#v\nactual: %#v\nexpected: %#v\n", v.input, actual, v.expected)
		}
	}
}

func TestByURLFailure(t *testing.T) {
	t.Run("Invalid URL", func(t *testing.T) {
		input := "@@://@@@/@@.@@@"
		_, err := ByURL(input)
		if err == nil {
			t.Errorf("awsiotprotocol.ByURL should fail with invalid input.\ninput: %#v", input)
		} else if _, ok := err.(*url.Error); !ok {
			t.Errorf("awsiotprotocol.ByURL should fail with url.Error type:\ninput: %#v\nactual error: %#v", input, err)
		}
	})
	t.Run("Unsupported protocol", func(t *testing.T) {
		input := "https://non-supported.protocol.com"
		expectedErrorMessage := "Protocol \"https\" is not supported"
		_, err := ByURL(input)
		if err == nil {
			t.Errorf("awsiotprotocol.ByURL should fail with invalid input.\ninput: %#v", input)
		} else if !strings.Contains(err.Error(), expectedErrorMessage) {
			t.Errorf("awsiotprotocol.ByURL should fail with error message which contains the following:\ninput: %#v\nexpected error message: %#v\nactual error message: %#v", input, expectedErrorMessage, err.Error())
		}
	})
}
