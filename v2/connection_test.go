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

package awsiotdev

import (
	"errors"
	"net/url"
	"reflect"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func TestDeviceClient(t *testing.T) {
	newClient = func(opt *mqtt.ClientOptions) mqtt.Client {
		return &MockClient{}
	}
	o := &Options{
		Keepalive:       time.Second * 2,
		URL:             "mock://",
		OfflineQueueing: true,
	}
	cli := New(o)

	if cli.cli == nil {
		t.Fatalf("Mqtt client is not initialized")
	}

	cli.Connect()

	subscribedMsg := ""
	cli.Subscribe("test", 1,
		func(client mqtt.Client, msg mqtt.Message) {
			subscribedMsg = string(msg.Payload())
		},
	)
	cli.Publish("test", 1, false, "test message")

	time.Sleep(time.Millisecond * 100)

	// Not yet connected. Must not be published and subscribed
	if cli.cli != nil && cli.cli.(*MockClient).subscribeNum != 0 {
		t.Fatalf("Subscribed before reconnection")
	}

	// Already published
	if cli.cli.(*MockClient).publishNum != 1 {
		t.Fatalf("Publish is not processed (%d)", cli.cli.(*MockClient).publishNum)
	}
	if cli.cli.(*MockClient).publishedMsg != "test message" {
		t.Fatalf("Published message is wrong (%s)", cli.cli.(*MockClient).publishedMsg)
	}

	// Establish connection
	cli.mqttOpt.OnConnect(cli.cli)
	time.Sleep(time.Millisecond * 100)

	if cli.cli.(*MockClient).subscribeNum != 1 {
		t.Fatalf("Queued subscription is not processed (%d)", cli.cli.(*MockClient).subscribeNum)
	}

	// Receive one message
	cli.cli.(*MockClient).cbMessage(cli.cli, &MockMessage{topic: "to", payload: []byte("a123")})
	time.Sleep(time.Millisecond * 100)
	if subscribedMsg != "a123" {
		t.Fatalf("Subscribed message payload is wrong (%s)", subscribedMsg)
	}

	// Disconnect
	cli.mqttOpt.OnConnectionLost(cli.cli, errors.New("Disconnect test"))
	time.Sleep(time.Millisecond * 100)

	// Establish connection (subscribeNum and publishNum is cleared)
	cli.mqttOpt.OnConnect(cli.cli)
	time.Sleep(time.Millisecond * 100)

	// Subscription must be back
	if cli.cli.(*MockClient).subscribeNum != 1 {
		t.Fatalf("Re-subscribe is not processed (%d)", cli.cli.(*MockClient).subscribeNum)
	}

	// Receive one message
	cli.cli.(*MockClient).cbMessage(cli.cli, &MockMessage{topic: "to", payload: []byte("b456")})
	time.Sleep(time.Millisecond * 100)
	if subscribedMsg != "b456" {
		t.Fatalf("Subscribed message payload is wrong (%s)", subscribedMsg)
	}

	// Another subscribe request
	subscribedMsg2 := ""
	cli.Subscribe("test2", 1,
		func(client mqtt.Client, msg mqtt.Message) {
			subscribedMsg2 = string(msg.Payload())
		},
	)

	time.Sleep(time.Millisecond * 100)

	if cli.cli.(*MockClient).subscribeNum != 2 {
		t.Fatalf("Subscription is not processed (%d)", cli.cli.(*MockClient).subscribeNum)
	}

	// Receive one message
	cli.cli.(*MockClient).cbMessage(cli.cli, &MockMessage{topic: "to", payload: []byte("c789")})
	time.Sleep(time.Millisecond * 100)
	if subscribedMsg2 != "c789" {
		t.Fatalf("Subscribed message payload is wrong (%s)", subscribedMsg)
	}

	// Disconnect
	cli.Disconnect(0)
	time.Sleep(time.Millisecond * 100)
	if cli.cli.(*MockClient).disconnectNum != 1 {
		t.Fatalf("Disconnect is not processed (%d)", cli.cli.(*MockClient).disconnectNum)
	}
}

func TestConnectionLostHandler(t *testing.T) {
	newClient = func(opt *mqtt.ClientOptions) mqtt.Client {
		return &MockClient{}
	}
	connectionLostHandled := false
	var connectionLostError error
	newBrokerURL, _ := url.Parse("mock://newbrokerurl")
	o := &Options{
		Keepalive:       time.Second * 2,
		URL:             "mock://",
		OfflineQueueing: true,
		Debug:           false,
		OnConnectionLost: func(opt *Options, mqttOpt *mqtt.ClientOptions, err error) {
			connectionLostHandled = true
			connectionLostError = err
			opt.Debug = true
			mqttOpt.Servers = []*url.URL{newBrokerURL}
		},
	}
	cli := New(o)
	cli.Connect()

	// Establish connection
	cli.mqttOpt.OnConnect(cli.cli)
	time.Sleep(time.Millisecond * 100)

	if connectionLostHandled != false {
		t.Fatal("ConnectionLostHandler is called before connection lost")
	}

	// Disconnect
	disconnectedError := errors.New("Disconnect test")
	cli.mqttOpt.OnConnectionLost(cli.cli, disconnectedError)
	time.Sleep(time.Millisecond * 100)

	// Establish connection
	cli.mqttOpt.OnConnect(cli.cli)
	time.Sleep(time.Millisecond * 100)

	if connectionLostHandled != true {
		t.Fatal("ConnectionLostHandler was not called on connection lost")
	}

	if connectionLostError != disconnectedError {
		t.Fatal("ConnectionLostHandler was not received connection lost err")
	}

	if cli.opt.Debug != true {
		t.Fatal("awsiotdev.Options not changed")
	}

	if !reflect.DeepEqual(cli.mqttOpt.Servers, []*url.URL{newBrokerURL}) {
		t.Fatal("mqtt.ClientOptions client options not changed")
	}
}

type MockMessage struct {
	topic   string
	payload []byte
}

func (s *MockMessage) Duplicate() bool {
	return false
}
func (s *MockMessage) Qos() byte {
	return 1
}
func (s *MockMessage) Retained() bool {
	return false
}
func (s *MockMessage) Topic() string {
	return s.topic
}
func (s *MockMessage) MessageID() uint16 {
	return 0
}
func (s *MockMessage) Payload() []byte {
	return s.payload
}
func (s *MockMessage) Ack() {
}

type MockClient struct {
	publishNum    int
	publishedMsg  string
	subscribeNum  int
	cbMessage     mqtt.MessageHandler
	disconnectNum int
}

func (s *MockClient) Connect() mqtt.Token {
	return &mqtt.DummyToken{}
}
func (s *MockClient) Disconnect(quiesce uint) {
	s.disconnectNum++
}
func (s *MockClient) Publish(topic string, qos byte, retained bool, payload interface{}) mqtt.Token {
	s.publishNum++
	s.publishedMsg = payload.(string)
	return &mqtt.DummyToken{}
}
func (s *MockClient) Subscribe(topic string, qos byte, cb mqtt.MessageHandler) mqtt.Token {
	s.subscribeNum++
	s.cbMessage = cb
	return &mqtt.DummyToken{}
}
func (s *MockClient) SubscribeMultiple(filters map[string]byte, callback mqtt.MessageHandler) mqtt.Token {
	for topic, qos := range filters {
		s.Subscribe(topic, qos, callback)
	}
	return &mqtt.DummyToken{}
}
func (s *MockClient) Unsubscribe(topics ...string) mqtt.Token {
	return &mqtt.DummyToken{}
}
func (s *MockClient) AddRoute(topic string, callback mqtt.MessageHandler) {
}
func (s *MockClient) IsConnected() bool {
	return true
}
func (s *MockClient) IsConnectionOpen() bool {
	return true
}
func (s *MockClient) OptionsReader() mqtt.ClientOptionsReader {
	return mqtt.ClientOptionsReader{}
}
