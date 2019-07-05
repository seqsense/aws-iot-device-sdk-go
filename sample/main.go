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

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	awsiot "github.com/seqsense/aws-iot-device-sdk-go/v2"
)

var message mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("Message received\n")
	fmt.Printf("  topic: %s\n", msg.Topic())
	fmt.Printf("  payload: %s\n", msg.Payload())
}

func messageHandler(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("Message received\n")
	fmt.Printf("  topic: %s\n", msg.Topic())
	fmt.Printf("  payload: %s\n", msg.Payload())
}

var (
	privatePath = flag.String("private-key-path", "private.pem.key", "Path to private key pem file")
	certPath    = flag.String("certificate-path", "certificate.pem.crt", "Path to certificate pem file")
	caPath      = flag.String("ca-path", "CAfile.pem", "Path to CA certificate pem file")
	thingName   = flag.String("thing-name", "sample", "Thing name")
	region      = flag.String("region", "ap-northeast-1", "AWS region")
	url         = flag.String("url", "mqtts://hoge.iot.ap-northeast-1.amazonaws.com", "AWS IoT endpoint")
)

func main() {
	flag.Parse()

	o := &awsiot.Options{
		KeyPath:                  *privatePath,
		CertPath:                 *certPath,
		CaPath:                   *caPath,
		ClientID:                 *thingName,
		Region:                   *region,
		BaseReconnectTime:        time.Millisecond * 50,
		MaximumReconnectTime:     time.Second * 2,
		MinimumConnectionTime:    time.Second * 2,
		Keepalive:                time.Second * 2,
		URL:                      *url,
		Debug:                    true,
		Qos:                      1,
		Retain:                   false,
		Will:                     &awsiot.TopicPayload{Topic: "notification", Payload: "{\"status\": \"dead\"}"},
		OfflineQueueing:          true,
		OfflineQueueMaxSize:      100,
		OfflineQueueDropBehavior: "oldest",
		AutoResubscribe:          true,
		OnConnectionLost: func(opt *awsiot.Options, mqttOpt *mqtt.ClientOptions, err error) {
			fmt.Printf("Connection lost handler function called\n")
		},
	}
	cli := awsiot.New(o)
	cli.Connect()
	cli.Subscribe("test", 1, messageHandler)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	tick := time.NewTicker(time.Second * 5)

	for {
		select {
		case <-sig:
			return
		case <-tick.C:
			cli.Publish("notification", 1, false, "{\"status\": \"tick\"}")
		}
	}
}
