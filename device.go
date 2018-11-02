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
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/seqsense/aws-iot-device-sdk-go/awsiotprotocol"
	"github.com/seqsense/aws-iot-device-sdk-go/pubqueue"
	"github.com/seqsense/aws-iot-device-sdk-go/subqueue"
)

const (
	stateUpdaterChCap = 100
	publishChCap      = 100
	subscribeChCap    = 100
)

type DeviceClient struct {
	opt             *Options
	mqttOpt         *mqtt.ClientOptions
	cli             mqtt.Client
	reconnectPeriod time.Duration
	stateUpdateCh   chan deviceState
	stableTimerCh   chan bool
	publishCh       chan *pubqueue.Data
	subscribeCh     chan *subqueue.Subscription
}

func New(opt *Options) *DeviceClient {
	p, err := awsiotprotocol.ByName(opt.Protocol)
	if err != nil {
		panic(err)
	}
	mqttOpt, err := p.NewClientOptions(
		&awsiotprotocol.Config{
			KeyPath:  opt.KeyPath,
			CertPath: opt.CertPath,
			CaPath:   opt.CaPath,
			ClientId: opt.ClientId,
			Host:     opt.Host,
		},
	)
	if err != nil {
		panic(err)
	}

	d := &DeviceClient{
		opt:             opt,
		mqttOpt:         mqttOpt,
		cli:             nil,
		reconnectPeriod: opt.BaseReconnectTime,
		stateUpdateCh:   make(chan deviceState, stateUpdaterChCap),
		stableTimerCh:   make(chan bool),
		publishCh:       make(chan *pubqueue.Data, publishChCap),
		subscribeCh:     make(chan *subqueue.Subscription, subscribeChCap),
	}

	connectionLost := func(client mqtt.Client, err error) {
		log.Printf("Connection lost (%s)\n", err.Error())
		d.stateUpdateCh <- inactive
	}
	onConnect := func(client mqtt.Client) {
		log.Printf("Connection established\n")
		d.stateUpdateCh <- established
	}

	d.mqttOpt.OnConnectionLost = connectionLost
	d.mqttOpt.OnConnect = onConnect
	if opt.Will != nil {
		d.mqttOpt.SetWill(opt.Will.Topic, opt.Will.Payload, opt.Qos, opt.Retain)
	}
	d.mqttOpt.SetKeepAlive(opt.Keepalive)
	d.mqttOpt.SetAutoReconnect(false) // MQTT AutoReconnect doesn't work well for mqtts
	d.mqttOpt.SetConnectTimeout(time.Second * 5)

	d.connect()
	go connectionHandler(d)

	return d
}

func (s *DeviceClient) connect() {
	s.cli = mqtt.NewClient(s.mqttOpt)

	token := s.cli.Connect()
	go func() {
		token.Wait()
		if token.Error() != nil {
			log.Printf("Failed to connect (%s)\n", token.Error())
			s.stateUpdateCh <- inactive
		}
	}()
}

func (s *DeviceClient) Disconnect() {
	s.stateUpdateCh <- terminating
}

func (s *DeviceClient) Publish(topic string, payload interface{}) {
	s.publishCh <- &pubqueue.Data{Topic: topic, Payload: payload}
}

func (s *DeviceClient) Subscribe(topic string, cb mqtt.MessageHandler) {
	s.subscribeCh <- &subqueue.Subscription{Type: subqueue.Subscribe, Topic: topic, Cb: cb}
}
func (s *DeviceClient) Unsubscribe(topic string) {
	s.subscribeCh <- &subqueue.Subscription{Type: subqueue.Unsubscribe, Topic: topic, Cb: nil}
}
