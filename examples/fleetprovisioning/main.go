// Copyright 2020 SEQSENSE, Inc.
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
	"context"
	"fmt"
	"os"
	"time"

	"github.com/at-wat/mqtt-go"
	"github.com/seqsense/aws-iot-device-sdk-go/v6"
	"github.com/seqsense/aws-iot-device-sdk-go/v6/fleetprovisioning"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if len(os.Args) != 4 {
		println("usage: fleetprovisioning AWS_IOT_ENDPOINT CLIENT_ID TEMPLATE_NAME")
		println("")
		println("This example registers a new thing in AWS IoT.")
		println("TEMPLATE_NAME must be created to your account of AWS IoT beforehand.")
		println("")
		println("Following files must be placed under the current working directory:")
		println("         root-CA.crt: root CA certificate")
		println(" certificate.pem.crt: client certificate with assigned policy to provision devices")
		println("  - see https://docs.aws.amazon.com/iot/latest/developerguide/provision-wo-cert.html")
		println("     private.pem.key: private key associated to above certificate")
		os.Exit(1)
	}
	host := os.Args[1]
	thingName := os.Args[2]
	templateName := os.Args[3]

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
		if err != nil {
			println(file, err)
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

	// Multiplex message handler to route messages to multiple features.
	var mux mqtt.ServeMux
	cli.Handle(&mux)

	f, err := fleetprovisioning.New(ctx, cli, templateName)
	if err != nil {
		panic(err)
	}
	f.OnError(func(err error) {
		fmt.Printf("async error: %v\n", err)
	})
	mux.Handle("#", f) // Handle messages.

	if _, err := cli.Connect(ctx,
		thingName,
		mqtt.WithKeepAlive(30),
	); err != nil {
		panic(err)
	}
	println("Connected to", host)

	_, _, _, certToken, err := f.CreateKeysAndCertificate(ctx)
	if err != nil {
		panic(err)
	}
	println("Created keys and certificate")

	cert, err := os.OpenFile("certificate-run.pem.crt", os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	key, err := os.OpenFile("private-run.pem.key", os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	f.WriteCertificate(cert)
	f.WritePrivateKey(key)
	cert.Close()
	key.Close()
	thingName, _, err = f.RegisterThing(ctx, certToken, map[string]string{"SerialNumber": "124"})
	if err != nil {
		panic(err)
	}
	println("Registered thing:", thingName)

	cli.Disconnect(ctx)

	cli, err = awsiotdev.New(
		thingName,
		&mqtt.URLDialer{
			URL: fmt.Sprintf("mqtts://%s:8883", host),
			Options: []mqtt.DialOption{
				mqtt.WithTLSCertFiles(
					host,
					"root-CA.crt",
					"certificate-run.pem.crt",
					"private-run.pem.key",
				),
				mqtt.WithConnStateHandler(func(s mqtt.ConnState, err error) {
					fmt.Printf("%s: %v\n", s, err)
				}),
			},
		},
		mqtt.WithReconnectWait(500*time.Millisecond, 2*time.Second),
	)

	cli.Handle(&mux)

	if _, err := cli.Connect(ctx,
		thingName,
		mqtt.WithKeepAlive(30),
	); err != nil {
		panic(err)
	}
	println("Connected to", host)

	select {}
}
