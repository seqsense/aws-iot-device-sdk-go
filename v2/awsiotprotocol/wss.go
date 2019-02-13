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
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Wss implements Protocol interface for MQTT over Websocket.
type Wss struct {
}

// Name returns the protocol name.
func (s Wss) Name() string {
	return "wss"
}

// NewClientOptions returns MQTT connection options.
func (s Wss) NewClientOptions(opt *Config) (*mqtt.ClientOptions, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(opt.URL)
	opts.SetClientID(opt.ClientID)
	opts.SetAutoReconnect(false) // use custom reconnection algorithm with offline queueing

	return opts, nil
}
