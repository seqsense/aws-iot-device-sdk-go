package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	awsiot "github.com/seqsense/aws-iot-device-sdk-go"
	"github.com/seqsense/aws-iot-device-sdk-go/options"
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
	host        = flag.String("host", "hoge.iot.ap-northeast-1.amazonaws.com", "AWS IoT host")
)

func main() {
	flag.Parse()

	o := &options.Options{
		KeyPath:               *privatePath,
		CertPath:              *certPath,
		CaPath:                *caPath,
		ClientId:              *thingName,
		Region:                *region,
		BaseReconnectTime:     time.Millisecond * 50,
		MaximumReconnectTime:  time.Second * 2,
		MinimumConnectionTime: time.Second * 2,
		Keepalive:             time.Second * 2,
		Protocol:              "mqtts",
		Host:                  *host,
		Debug:                 false,
		Qos:                   1,
		Retain:                false,
		Will:                  &options.TopicPayload{"notification", "{\"status\", \"dead\"}"},
	}
	cli := awsiot.New(o)
	time.Sleep(2 * time.Second)

	cli.Subscribe("test", messageHandler)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-sig:
			return
		}
	}
}
