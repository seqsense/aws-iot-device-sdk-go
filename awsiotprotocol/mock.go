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
	"fmt"
	"net/url"
	"strconv"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// MockProtocol implements Protocol interface without actual connection.
// This is for testing.
type MockProtocol struct {
}

// Name returns the protocol name.
func (s MockProtocol) Name() string {
	return "mock"
}

// NewClientOptions returns MQTT connection options.
func (s MockProtocol) NewClientOptions(opt *Config) (*mqtt.ClientOptions, error) {
	url, _ := url.Parse(opt.URL)
	host := url.Hostname()
	port, _ := strconv.Atoi(url.Port())
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("mock://%s:%d", host, port))

	return opts, nil
}
