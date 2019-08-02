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

/*
Package awsiotdev implements offline queueing and reconnecting features of MQTT protocol.
*/
package awsiotdev

import (
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/seqsense/aws-iot-device-sdk-go/v2/awsiotprotocol"
	"github.com/seqsense/aws-iot-device-sdk-go/v2/subqueue"
)

const (
	stateUpdaterChCap = 100
	subscribeChCap    = 100
)

// DeviceClient inplements mqtt.Client interface.
// Publishing messages and subscribing topics are queued in the DeviceClient if
// the network connection is lost. They are re-tried after the connection is
// resumed.
type DeviceClient struct {
	opt           *Options
	mqttOpt       *mqtt.ClientOptions
	cli           mqtt.Client
	stateUpdateCh chan deviceState
	subscribeCh   chan *subqueue.Subscription
	dbg           *debugOut
}

// New returns new MQTT client with offline queueing and reconnecting.
// Returned client is not connected to the broker until calling Connect().
func New(opt *Options) *DeviceClient {
	p, err := awsiotprotocol.ByURL(opt.URL)
	if err != nil {
		panic(err)
	}
	mqttOpt, err := p.NewClientOptions(
		&awsiotprotocol.Config{
			KeyPath:  opt.KeyPath,
			CertPath: opt.CertPath,
			CaPath:   opt.CaPath,
			ClientID: opt.ClientID,
			URL:      opt.URL,
		},
	)
	if err != nil {
		panic(err)
	}

	d := &DeviceClient{
		opt:           opt,
		mqttOpt:       mqttOpt,
		cli:           nil,
		stateUpdateCh: make(chan deviceState, stateUpdaterChCap),
		subscribeCh:   make(chan *subqueue.Subscription, subscribeChCap),
		dbg:           &debugOut{opt.Debug},
	}

	connectionLost := func(client mqtt.Client, err error) {
		d.dbg.printf("Connection lost (%s)\n", err.Error())
		if d.opt.OnConnectionLost != nil {
			d.opt.OnConnectionLost(d.opt, d.mqttOpt, err)
		}

		d.stateUpdateCh <- inactive
	}
	onConnect := func(client mqtt.Client) {
		d.dbg.printf("Connection established\n")
		d.stateUpdateCh <- established
	}

	d.mqttOpt.OnConnectionLost = connectionLost
	d.mqttOpt.OnConnect = onConnect
	if opt.Will != nil {
		d.mqttOpt.SetWill(opt.Will.Topic, opt.Will.Payload, opt.Qos, opt.Retain)
	}
	d.mqttOpt.SetKeepAlive(opt.Keepalive)
	d.mqttOpt.SetAutoReconnect(true)
	// This enables to use persisted session feature of MQTT.
	// cf. https://docs.aws.amazon.com/iot/latest/developerguide/mqtt-persistent-sessions.html
	d.mqttOpt.SetCleanSession(false)
	d.mqttOpt.SetConnectTimeout(time.Second * 5)

	d.cli = newClient(d.mqttOpt)

	return d
}

// Connect create a connection to the broker.
// Returned token indicates success immediately.
// Subscription requests and published messages are queued until actual connection establish.
func (s *DeviceClient) Connect() mqtt.Token {
	go connectionHandler(s)

	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()

	for {
		var token mqtt.Token
		if token = s.cli.Connect(); token.Wait() && token.Error() == nil {
			break
		}
		s.dbg.printf("Failed to connect (%s)\n", token.Error())
		<-tick.C
	}
	return &mqtt.DummyToken{}
}

var newClient = func(opt *mqtt.ClientOptions) mqtt.Client {
	return mqtt.NewClient(opt)
}

// Disconnect ends the connection to the broker.
// Currently, it disconnects immediately and quiesce argument is ignored.
func (s *DeviceClient) Disconnect(quiesce uint) {
	s.stateUpdateCh <- terminating
}

// Publish publishes a message.
// Currently, qos and retained arguments are ignored and ones specified in the options are used.
func (s *DeviceClient) Publish(topic string, qos byte, retained bool, payload interface{}) mqtt.Token {
	return s.cli.Publish(topic, qos, retained, payload)
}

// Subscribe requests a new subscription for the specified topic.
// Currently, qos argument is ignored and one specified in the options is used.
func (s *DeviceClient) Subscribe(topic string, qos byte, cb mqtt.MessageHandler) mqtt.Token {
	s.subscribeCh <- &subqueue.Subscription{Type: subqueue.Subscribe, Topic: topic, Cb: cb}
	return &mqtt.DummyToken{}
}

// SubscribeMultiple requests new subscription for multiple topics.
// Currently, specified qos is ignored and one specified in the options is used.
func (s *DeviceClient) SubscribeMultiple(filters map[string]byte, callback mqtt.MessageHandler) mqtt.Token {
	for topic, qos := range filters {
		s.Subscribe(topic, qos, callback)
	}
	return &mqtt.DummyToken{}
}

// Unsubscribe ends the subscriptions for the specified topics.
func (s *DeviceClient) Unsubscribe(topics ...string) mqtt.Token {
	for _, topic := range topics {
		s.subscribeCh <- &subqueue.Subscription{Type: subqueue.Unsubscribe, Topic: topic, Cb: nil}
	}
	return &mqtt.DummyToken{}
}

// AddRoute is not supported in this package at now.
func (s *DeviceClient) AddRoute(topic string, callback mqtt.MessageHandler) {
	panic("awsiotdev doesn't support AddRoute")
}

// IsConnected returns a bool whether the client is connected to the broker or not.
func (s *DeviceClient) IsConnected() bool {
	return s.cli.IsConnected()
}

// IsConnectionOpen returns a bool whether the client has an active connection to the broker.
// It is not supported in the latest paho.mqtt.golang release, but will be supported in the future release.
func (s *DeviceClient) IsConnectionOpen() bool {
	// paho.mqtt.golang v1.1.1 don't have it.
	// this will be added in the next version.
	return true // since offline queued
}

// OptionsReader returns a ClientOptionsReader of the internal MQTT client.
func (s *DeviceClient) OptionsReader() mqtt.ClientOptionsReader {
	return s.cli.OptionsReader()
}
