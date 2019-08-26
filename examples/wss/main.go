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

/**
  Usage: go run . -host hoge-ats.iot.ap-northeast-1.amazonaws.com
*/

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	awsiot "github.com/seqsense/aws-iot-device-sdk-go/v3"
	presigner "github.com/seqsense/aws-iot-device-sdk-go/v3/presigner"
)

func messageHandler(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("Message received\n")
	fmt.Printf("  topic: %s\n", msg.Topic())
	fmt.Printf("  payload: %s\n", msg.Payload())
}

var (
	thingName = flag.String("thing-name", "sample", "Thing name")
	region    = flag.String("region", "ap-northeast-1", "AWS region")
	host      = flag.String("host", "hoge-ats.iot.ap-northeast-1.amazonaws.com", "AWS IoT endpoint")
)

func main() {
	flag.Parse()
	sess := session.Must(session.NewSession())
	signer := presigner.New(sess, &aws.Config{Region: aws.String(*region)})
	endpoint, _ := signer.PresignWssNow(*host)

	o := &awsiot.Options{
		ClientID:                 *thingName,
		Region:                   *region,
		BaseReconnectTime:        time.Millisecond * 50,
		MaximumReconnectTime:     time.Second * 2,
		MinimumConnectionTime:    time.Second * 2,
		Keepalive:                time.Second * 2,
		URL:                      endpoint,
		Debug:                    true,
		Qos:                      1,
		Retain:                   false,
		Will:                     &awsiot.TopicPayload{Topic: "notification", Payload: "{\"status\": \"dead\"}"},
		OfflineQueueing:          true,
		OfflineQueueMaxSize:      100,
		OfflineQueueDropBehavior: "oldest",
		AutoResubscribe:          true,
		OnConnectionLost: func(opt *awsiot.Options, err error) {
			fmt.Printf("Connection lost handler function called\n")
			newEndpoint, err := signer.PresignWssNow(*host)
			if err != nil {
				panic(err)
			}
			opt.URL = newEndpoint
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
