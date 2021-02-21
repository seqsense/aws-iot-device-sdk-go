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

// Package awsiotdev implements AWS IoT presigned WebSockets dialer for github.com/at-wat/mqtt-go, and AWS IoT functionalities.
package awsiotdev

import (
	"github.com/at-wat/mqtt-go"

	"github.com/seqsense/aws-iot-device-sdk-go/v5/internal/ioterr"
)

// Device is an AWS IoT device.
type Device interface {
	mqtt.Client
	ThingName() string
}

type device struct {
	mqtt.Client
	thingName string
}

// New creates AWS IoT device interface.
func New(thingName string, dialer mqtt.Dialer, opts ...mqtt.ReconnectOption) (Device, error) {
	cli, err := mqtt.NewReconnectClient(dialer, opts...)
	if err != nil {
		return nil, ioterr.New(err, "creating MQTT connection")
	}
	return &device{
		Client:    cli,
		thingName: thingName,
	}, nil
}

func (d *device) ThingName() string {
	return d.thingName
}
