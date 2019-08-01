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
	"fmt"
	"net/url"
	"reflect"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func TestDeviceClient(t *testing.T) {
	connectionNum := 0

	newClient = func(opt *mqtt.ClientOptions) mqtt.Client {
		connectionNum++
		return &MockClient{}
	}
	o := &Options{
		BaseReconnectTime:        time.Millisecond * 10,
		MaximumReconnectTime:     time.Millisecond * 50,
		MinimumConnectionTime:    time.Millisecond * 50,
		Keepalive:                time.Second * 2,
		URL:                      "mock://",
		OfflineQueueing:          true,
		OfflineQueueMaxSize:      100,
		OfflineQueueDropBehavior: "oldest",
		AutoResubscribe:          true,
	}
	cli := New(o)

	if connectionNum != 0 {
		t.Fatalf("Connected before connection (%d)", connectionNum)
	}
	cli.Connect()
	if cli.cli == nil {
		t.Fatalf("Mqtt client is not initialized")
	}
	if connectionNum != 1 {
		t.Fatalf("Connected not once (%d)", connectionNum)
	}

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
	if cli.cli != nil && cli.cli.(*MockClient).publishNum != 0 {
		t.Fatalf("Published before reconnection")
	}

	// Establish connection
	cli.mqttOpt.OnConnect(cli.cli)
	time.Sleep(time.Millisecond * 100)

	if cli.cli.(*MockClient).subscribeNum != 1 {
		t.Fatalf("Queued subscription is not processed (%d)", cli.cli.(*MockClient).subscribeNum)
	}
	if cli.cli.(*MockClient).publishNum != 1 {
		t.Fatalf("Queued publish is not processed (%d)", cli.cli.(*MockClient).publishNum)
	}
	if cli.cli.(*MockClient).publishedMsg != "test message" {
		t.Fatalf("Published message is wrong (%s)", cli.cli.(*MockClient).publishedMsg)
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
	if connectionNum != 2 {
		t.Fatalf("Connected not twice (%d)", connectionNum)
	}
	cli.Publish("test", 1, false, "test message2")

	// Must not published yet
	if cli.cli.(*MockClient).publishNum != 0 {
		t.Fatalf("Queued publish is not processed (%d)", cli.cli.(*MockClient).publishNum)
	}

	// Establish connection (subscribeNum and publishNum is cleared)
	cli.mqttOpt.OnConnect(cli.cli)
	time.Sleep(time.Millisecond * 100)

	// Subscription must be back
	if cli.cli.(*MockClient).subscribeNum != 1 {
		t.Fatalf("Re-subscribe is not processed (%d)", cli.cli.(*MockClient).subscribeNum)
	}
	// Message requested during connection lost must be published
	if cli.cli.(*MockClient).publishNum != 1 {
		t.Fatalf("Queued publish is not processed (%d)", cli.cli.(*MockClient).publishNum)
	}
	if cli.cli.(*MockClient).publishedMsg != "test message2" {
		t.Fatalf("Published message is wrong (%s)", cli.cli.(*MockClient).publishedMsg)
	}

	// Receive one message
	cli.cli.(*MockClient).cbMessage(cli.cli, &MockMessage{topic: "to", payload: []byte("b456")})
	time.Sleep(time.Millisecond * 100)
	if subscribedMsg != "b456" {
		t.Fatalf("Subscribed message payload is wrong (%s)", subscribedMsg)
	}

	// Another subscribe and publish request
	subscribedMsg2 := ""
	cli.Subscribe("test2", 1,
		func(client mqtt.Client, msg mqtt.Message) {
			subscribedMsg2 = string(msg.Payload())
		},
	)
	cli.Publish("test2", 1, false, "test message3")

	time.Sleep(time.Millisecond * 100)

	if cli.cli.(*MockClient).subscribeNum != 2 {
		t.Fatalf("Subscription is not processed (%d)", cli.cli.(*MockClient).subscribeNum)
	}
	if cli.cli.(*MockClient).publishNum != 2 {
		t.Fatalf("Publish is not processed (%d)", cli.cli.(*MockClient).publishNum)
	}
	if cli.cli.(*MockClient).publishedMsg != "test message3" {
		t.Fatalf("Published message is wrong (%s)", cli.cli.(*MockClient).publishedMsg)
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
		BaseReconnectTime:        time.Millisecond * 10,
		MaximumReconnectTime:     time.Millisecond * 50,
		MinimumConnectionTime:    time.Millisecond * 50,
		Keepalive:                time.Second * 2,
		URL:                      "mock://",
		OfflineQueueing:          true,
		OfflineQueueMaxSize:      100,
		OfflineQueueDropBehavior: "oldest",
		AutoResubscribe:          true,
		Debug:                    false,
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

func TestMultipleSubscription(t *testing.T) {
	newClient = func(opt *mqtt.ClientOptions) mqtt.Client {
		return &MockClient{}
	}
	o := &Options{
		BaseReconnectTime:        time.Millisecond * 10,
		MaximumReconnectTime:     time.Millisecond * 50,
		MinimumConnectionTime:    time.Millisecond * 50,
		Keepalive:                time.Second * 2,
		URL:                      "mock://",
		OfflineQueueing:          true,
		OfflineQueueMaxSize:      100,
		OfflineQueueDropBehavior: "oldest",
		AutoResubscribe:          true,
	}
	cli := New(o)

	subs := 20      // number of dummy subscriptions
	iterations := 5 // number of state change iterations

	// Subscribes to many channels
	for i := 0; i <= subs; i++ {
		cli.Subscribe(fmt.Sprintf("%d", i), 1, nil)
	}

	cli.Connect()

	// This loop changes the state between stable and inactive multiple times,
	// and it invokes the subscription and unsubscription multiple times.
	// This causes `concurrent map writes` error if the subscription map isn't
	// locked correctly.
	for i := 0; i <= iterations; i++ {
		cli.stateUpdateCh <- stable
		time.Sleep(10 * time.Millisecond)
		cli.stateUpdateCh <- inactive
		time.Sleep(10 * time.Millisecond)
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
