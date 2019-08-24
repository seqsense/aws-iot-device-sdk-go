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
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/seqsense/aws-iot-device-sdk-go/v2/internal/pubqueue"
)

func TestClientInterface(t *testing.T) {
	var cli mqtt.Client
	cli = &DeviceClient{}
	_ = cli
}

func TestPublish(t *testing.T) {
	t.Run("QoS 0", func(t *testing.T) {
		ch := make(chan *pubqueue.Data)
		defer close(ch)
		cli := DeviceClient{
			cli:       &MockClient{},
			publishCh: ch,
		}
		testMsg := "test message"
		go cli.Publish("test", 0, false, testMsg)
		time.Sleep(time.Millisecond * 100)
		if len(cli.publishCh) != 0 {
			t.Fatal("Message is unexpectedly offline-buffered")
		}
		if cli.cli.(*MockClient).publishNum != 1 {
			t.Fatalf("Publish operation is not processed (%d)", cli.cli.(*MockClient).publishNum)
		}
		if cli.cli.(*MockClient).publishedMsg != testMsg {
			t.Fatalf("Published message is wrong. expected: %s, actual: %s", testMsg, cli.cli.(*MockClient).publishedMsg)
		}
	})
	t.Run("QoS 1", func(t *testing.T) {
		ch := make(chan *pubqueue.Data)
		defer close(ch)
		cli := DeviceClient{
			cli:       &MockClient{},
			publishCh: ch,
		}

		testTopic := "testTopic"
		testMsg := "test message"
		go cli.Publish(testTopic, 1, false, testMsg)
		select {
		case msg := <-cli.publishCh:
			if msg.Topic != testTopic {
				t.Errorf("Published topic is wrong. expected: %s, actual: %s", testTopic, msg.Topic)
			}
			if msg.Payload.(string) != testMsg {
				t.Errorf("Published message is wrong. expected: %s, actual: %s", testMsg, msg.Payload)
			}
		case <-time.After(time.Second):
			t.Fatal("Message is not offline-buffered")
		}
	})
}
