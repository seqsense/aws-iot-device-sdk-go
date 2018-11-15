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

// Package awsiotprotocol is internal implementation of awsiotdev package.
// It implements MQTT connection type loader.
package awsiotprotocol

import (
	"fmt"
	"net/url"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var (
	protocols = make(map[string]Protocol)
)

// Protocol is the interface to describe MQTT connection type.
type Protocol interface {
	Name() string
	NewClientOptions(opt *Config) (*mqtt.ClientOptions, error)
}

func init() {
	registerProtocol(Mqtts{})
	registerProtocol(Wss{})
	registerProtocol(MockProtocol{})
}

// ByUrl selects MQTT connection type by URL string.
func ByUrl(u string) (Protocol, error) {
	url, err := url.Parse(u)
	if err != nil {
		return nil, err
	}

	p, ok := protocols[url.Scheme]
	if !ok {
		return nil, fmt.Errorf("Protocol \"%s\" is not supported", url.Scheme)
	}
	return p, nil
}

func registerProtocol(p Protocol) {
	n := p.Name()
	protocols[n] = p
}
