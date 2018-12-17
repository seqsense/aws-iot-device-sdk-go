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

func TestMqttsName(t *testing.T) {
	name := Mqtts{}.Name()

	if name != "mqtts" {
		t.Fatalf("MQTTS protocol name is bad")
	}
}

func TestMqttsNewClientOptions(t *testing.T) {
	opt := &Config{
		URL:      "mqtts://example.com:8882",
		ClientID: "mqttsclientid",
		CaPath:   "../sample/samplecerts/cafile.pem",
		CertPath: "../sample/samplecerts/client-crt.pem",
		KeyPath:  "../sample/samplecerts/client-key.pem",
	}
	mqttOpts, _ := Mqtts{}.NewClientOptions(opt)

	if mqttOpts == nil {
		t.Fatalf("MQTT.ClientOptions is nil")
	}
	if mqttOpts.Servers[0].Scheme != "ssl" || mqttOpts.Servers[0].Host != "example.com:8882" {
		t.Fatalf("MQTT Broker does not add correctly")
	}
	if mqttOpts.ClientID != "mqttsclientid" {
		t.Fatalf("MQTT ClientID does not set correctly")
	}
}
