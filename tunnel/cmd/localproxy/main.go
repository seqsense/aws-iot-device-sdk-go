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
	"flag"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/seqsense/aws-iot-device-sdk-go/v5/tunnel"
)

var (
	accessToken     = flag.String("access-token", "", "Client access token")
	proxyEndpoint   = flag.String("proxy-endpoint", "", "Endpoint of proxy server (e.g. data.tunneling.iot.ap-northeast-1.amazonaws.com:443)")
	region          = flag.String("region", "", "Endpoint region. Exclusive flag with -proxy-endpoint")
	sourcePort      = flag.Int("source-listen-port", 0, "Assigns source mode and sets the port to listen")
	destinationApp  = flag.String("destination-app", "", "Assigns destination mode and set the endpoint in address:port format")
	noSSLHostVerify = flag.Bool("no-ssl-host-verify", false, "Turn off SSL host verification")
	proxyScheme     = flag.String("proxy-scheme", "wss", "Proxy server protocol scheme")
)

func main() {
	flag.Parse()

	if *accessToken == "" {
		log.Fatal("error: -access-token must be specified")
	}

	var endpoint string
	switch {
	case *proxyEndpoint != "" && *region == "":
		endpoint = *proxyEndpoint
	case *region != "" && *proxyEndpoint == "":
		endpoint = fmt.Sprintf("data.tunneling.iot.%s.amazonaws.com", *region)
	default:
		log.Fatal("error: one of -proxy-endpoint or -region must be specified")
	}

	proxyOpts := []tunnel.ProxyOption{
		func(opt *tunnel.ProxyOptions) error {
			opt.InsecureSkipVerify = *noSSLHostVerify
			opt.Scheme = *proxyScheme
			return nil
		},
		tunnel.WithErrorHandler(tunnel.ErrorHandlerFunc(func(err error) {
			log.Print(err)
		})),
	}

	switch {
	case *sourcePort > 0 && *destinationApp == "":
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", *sourcePort))
		if err != nil {
			log.Fatalf("error: %v", err)
		}
		err = tunnel.ProxySource(listener, endpoint, *accessToken, proxyOpts...)
		if err != nil {
			log.Fatalf("error: %v", err)
		}

	case *destinationApp != "" && *sourcePort == 0:
		err := tunnel.ProxyDestination(func() (io.ReadWriteCloser, error) {
			return net.Dial("tcp", *destinationApp)
		}, endpoint, *accessToken, proxyOpts...)
		if err != nil {
			log.Fatalf("error: %v", err)
		}

	default:
		log.Fatal("error: one of -source-listen-port or -destination-app must be specified")
	}
}
