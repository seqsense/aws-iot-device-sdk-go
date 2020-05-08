package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	mqtt "github.com/at-wat/mqtt-go"
	"github.com/aws/aws-sdk-go/aws/session"
	awsiot "github.com/seqsense/aws-iot-device-sdk-go/v4"
	"github.com/seqsense/aws-iot-device-sdk-go/v4/tunnel/server"
)

var (
	mqttEndpoint = flag.String("mqtt-endpoint", "", "AWS IoT endpoint")
	apiAddr      = flag.String("api-addr", ":80", "Address and port of API endpoint")
	tunnelAddr   = flag.String("tunnel-addr", ":80", "Address and port of proxy WebSocket endpoint")
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	flag.Parse()

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
		if _, err := cli.Connect(context.Background(),
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

	var wg sync.WaitGroup
	chErr := make(chan error, len(servers))
	for _, s := range servers {
		wg.Add(1)
		go func(s *http.Server) {
			chErr <- s.ListenAndServe()
			wg.Done()
		}(s)
	}

	switch err := <-chErr; err {
	case http.ErrServerClosed, nil:
	default:
		log.Print(err)
	}
	for _, s := range servers {
		if err := s.Close(); err != nil {
			log.Print(err)
		}
	}
	wg.Wait()
}
