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
	"testing"
)

func TestWssName(t *testing.T) {
	name := Wss{}.Name()

	if name != "wss" {
		t.Fatalf("Wss protocol name is bad")
	}
}

func TestWssNewClientOptions(t *testing.T) {
	opt := &Config{
		URL:      "wss://example.com:443",
		ClientID: "wssclientid",
	}
	mqttOpts, _ := Wss{}.NewClientOptions(opt)

	if mqttOpts == nil {
		t.Fatalf("MQTT.ClientOptions is nil")
	}
	if mqttOpts.Servers[0].Scheme != "wss" || mqttOpts.Servers[0].Host != "example.com:443" {
		t.Fatalf("Broker is not added")
	}
	if mqttOpts.ClientID != "wssclientid" {
		t.Fatalf("ClientID is not set")
	}
	if mqttOpts.AutoReconnect != false {
		t.Fatalf("AutoReconnect flag is not set")
	}
}
