package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/at-wat/mqtt-go"
	"github.com/seqsense/aws-iot-device-sdk-go/v4"
	"github.com/seqsense/aws-iot-device-sdk-go/v4/shadow"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if len(os.Args) != 3 {
		println("usage: shadow AWS_IOT_ENDPOINT THING_NAME")
		println("")
		println("This example updates and deletes AWS IoT Thing Shadow.")
		println("THING_NAME must be registered to your account of AWS IoT beforehand.")
		println("")
		println("Following files must be placed under the current working directory:")
		println("         root-CA.crt: root CA certificate")
		println(" certificate.pem.crt: client certificate associated to THING_NAME")
		println("     private.pem.key: private key associated to THING_NAME")
		os.Exit(1)
	}
	host := os.Args[1]
	thingName := os.Args[2]

	for _, file := range []string{
		"root-CA.crt",
		"certificate.pem.crt",
		"private.pem.key",
	} {
		_, err := os.Stat(file)
		if os.IsNotExist(err) {
			println(file, "not found")
			os.Exit(1)
		}
	}

	cli, err := awsiotdev.New(
		thingName,
		&mqtt.URLDialer{
			URL: fmt.Sprintf("mqtts://%s:8883", host),
			Options: []mqtt.DialOption{
				mqtt.WithTLSCertFiles(
					host,
					"root-CA.crt",
					"certificate.pem.crt",
					"private.pem.key",
				),
				mqtt.WithConnStateHandler(func(s mqtt.ConnState, err error) {
					fmt.Printf("%s: %v\n", s, err)
				}),
			},
		},
		mqtt.WithReconnectWait(500*time.Millisecond, 2*time.Second),
	)
	if err != nil {
		panic(err)
	}
	cli.Handle(mqtt.HandlerFunc(func(m *mqtt.Message) {
		fmt.Printf("Message dropped: %v\n", *m)
	}))

	if _, err := cli.Connect(ctx,
		thingName,
		mqtt.WithKeepAlive(30),
	); err != nil {
		panic(err)
	}

	s, err := shadow.New(ctx, cli)
	if err != nil {
		panic(err)
	}
	s.OnError(func(err error) {
		fmt.Printf("async error: %v\n", err)
	})
	cli.Handle(s)

	fmt.Print("> update desire\n")
	doc, err := s.Desire(ctx, sampleState{Value: 1})
	if err != nil {
		panic(err)
	}
	fmt.Printf("document: %v\n", doc)

	fmt.Print("> update report\n")
	doc, err = s.Report(ctx, sampleState{Value: 2})
	if err != nil {
		panic(err)
	}
	fmt.Printf("document: %v\n", doc)

	fmt.Print("> get document\n")
	doc, err = s.Get(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Printf("document: %v\n", doc)

	fmt.Print("> delete\n")
	err = s.Delete(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Printf("document: %v\n", s.Document())
}

type sampleState struct {
	Value  int
	Struct struct {
		NestedValue int
	}
}
