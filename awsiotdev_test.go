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

package awsiotdev

import (
	"testing"

	"github.com/at-wat/mqtt-go"
)

func TestNew(t *testing.T) {
	const name = "thing_name1"
	d, err := New(name, mqtt.DialerFunc(func() (mqtt.ClientCloser, error) {
		return &mqtt.BaseClient{}, nil
	}))
	if err != nil {
		t.Fatal(err)
	}

	if d.ThingName() != name {
		t.Errorf("ThingName differs, expected: %s, got: %s", name, d.ThingName())
	}
}
