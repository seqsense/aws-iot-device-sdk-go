package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/at-wat/mqtt-go"
	"github.com/seqsense/aws-iot-device-sdk-go/v4/tunnel"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if len(os.Args) != 2 {
		println("usage: tunnel aws_iot_endpoint")
		os.Exit(1)
	}
	host := os.Args[1]

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

	cli, err := mqtt.NewReconnectClient(
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
	if _, err := cli.Connect(ctx,
		"sample",
		mqtt.WithKeepAlive(30),
	); err != nil {
		panic(err)
	}

	t, err := tunnel.New(ctx, cli, "sample", map[string]tunnel.Dialer{
		"ssh": func() (io.ReadWriteCloser, error) {
			return net.Dial("tcp", "localhost:22")
		},
	})
	if err != nil {
		panic(err)
	}
	cli.Handle(t)

	select {}
}
