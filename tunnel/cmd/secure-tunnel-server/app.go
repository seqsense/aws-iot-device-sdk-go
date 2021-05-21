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
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	mqtt "github.com/at-wat/mqtt-go"
	"github.com/aws/aws-sdk-go/aws/session"
	awsiot "github.com/seqsense/aws-iot-device-sdk-go/v5"
	"github.com/seqsense/aws-iot-device-sdk-go/v5/tunnel/server"
)

func app(ctx context.Context, args []string) {
	rand.Seed(time.Now().UnixNano())

	f := flag.NewFlagSet(args[0], flag.ExitOnError)
	var (
		mqttEndpoint      = f.String("mqtt-endpoint", "", "AWS IoT endpoint")
		apiAddr           = f.String("api-addr", ":80", "Address and port of API endpoint")
		tunnelAddr        = f.String("tunnel-addr", ":80", "Address and port of proxy WebSocket endpoint")
		generateTestToken = f.Bool("generate-test-token", false, "Generate a token for testing")
	)
	f.Parse(args[1:])

	var notifier *server.Notifier
	if *mqttEndpoint != "" {
		sess := session.Must(session.NewSession())
		dialer, err := awsiot.NewPresignDialer(sess, *mqttEndpoint,
			mqtt.WithConnStateHandler(func(s mqtt.ConnState, err error) {
				log.Printf("MQTT connection state changed (%s)", s)
			}),
		)
		if err != nil {
			log.Fatalf("Failed to create AWS IoT presign dialer (%s)", err.Error())
		}
		cli, err := mqtt.NewReconnectClient(
			dialer,
			mqtt.WithReconnectWait(50*time.Millisecond, 2*time.Second),
		)
		if err != nil {
			log.Fatalf("Failed to create MQTT client (%s)", err.Error())
		}

		ctxConnect, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		if _, err := cli.Connect(ctxConnect,
			fmt.Sprintf("secure-tunnel-server-%d", rand.Int()),
			mqtt.WithKeepAlive(30),
		); err != nil {
			log.Fatalf("Failed to start MQTT reconnect client (%s)", err.Error())
		}
		notifier = server.NewNotifier(cli)
	} else {
		log.Print("MQTT notification is disabled")
	}

	tunnelHandler := server.NewTunnelHandler()
	apiHandler := server.NewAPIHandler(tunnelHandler, notifier)

	if *generateTestToken {
		apiHandler.ServeHTTP(
			&noopResponseWriter{},
			&http.Request{
				Header: http.Header{
					"X-Amz-Target": {"IoTSecuredTunneling.OpenTunnel"},
				},
				Body: ioutil.NopCloser(
					bytes.NewReader([]byte(
						"{\"DestinationConfig\": {\"Services\": [\"ssh\"], \"ThingName\": \"test\"}}",
					)),
				),
			},
		)
	}

	servers := map[string]*http.Server{
		*apiAddr: {
			Addr:         *apiAddr,
			Handler:      http.NewServeMux(),
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		},
	}
	if *apiAddr != *tunnelAddr {
		servers[*tunnelAddr] = &http.Server{
			Addr:         *tunnelAddr,
			Handler:      http.NewServeMux(),
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		}
	}

	servers[*apiAddr].Handler.(*http.ServeMux).Handle("/", apiHandler)
	servers[*tunnelAddr].Handler.(*http.ServeMux).Handle("/tunnel", tunnelHandler)

	healthcheckHandler := func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, "ok")
	}
	servers[*apiAddr].Handler.(*http.ServeMux).HandleFunc("/healthcheck", healthcheckHandler)
	if *apiAddr != *tunnelAddr {
		servers[*tunnelAddr].Handler.(*http.ServeMux).HandleFunc("/healthcheck", healthcheckHandler)
	}

	var wg sync.WaitGroup
	chErr := make(chan error, len(servers))
	for _, s := range servers {
		wg.Add(1)
		go func(s *http.Server) {
			chErr <- s.ListenAndServe()
			wg.Done()
		}(s)
	}

	select {
	case err := <-chErr:
		switch err {
		case http.ErrServerClosed, nil:
		default:
			log.Print(err)
		}
	case <-ctx.Done():
	}
	for _, s := range servers {
		if err := s.Close(); err != nil {
			log.Print(err)
		}
	}
	wg.Wait()
}

type noopResponseWriter struct{}

func (*noopResponseWriter) Header() http.Header        { return make(http.Header) }
func (*noopResponseWriter) WriteHeader(statusCode int) {}
func (*noopResponseWriter) Write(b []byte) (int, error) {
	log.Println(string(b))
	return len(b), nil
}
