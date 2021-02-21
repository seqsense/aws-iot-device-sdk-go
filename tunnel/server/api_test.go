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

package server

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	ist "github.com/aws/aws-sdk-go/service/iotsecuretunneling"

	"github.com/seqsense/aws-iot-device-sdk-go/v5/internal/ioterr"
)

func TestAPI(t *testing.T) {
	var wg sync.WaitGroup
	defer wg.Wait()

	tunnelHandler := NewTunnelHandler()
	apiHandler := NewAPIHandler(tunnelHandler, nil)
	mux := http.NewServeMux()
	mux.Handle("/", apiHandler)
	mux.Handle("/tunnel", tunnelHandler)

	s := &http.Server{
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	defer func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	}()

	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		switch err := s.Serve(ln); err {
		case http.ErrServerClosed, nil:
		default:
			t.Error(err)
		}
	}()

	sess := session.Must(session.NewSession(&aws.Config{
		Region:           aws.String("nothing"),
		DisableSSL:       aws.Bool(true),
		EndpointResolver: newEndpointForFunc(ln.Addr().(*net.TCPAddr).Port),
		Credentials: credentials.NewStaticCredentials(
			"ASIAZZZZZZZZZZZZZZZZ",
			"0000000000000000000000000000000000000000",
			"",
		),
	}))
	api := ist.New(sess)
	out, err := api.OpenTunnel(&ist.OpenTunnelInput{
		Description: aws.String("desc"),
		DestinationConfig: &ist.DestinationConfig{
			Services: []*string{
				aws.String("ssh"),
			},
			ThingName: aws.String("thing"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%v", out)
}

func TestAPI_Validate(t *testing.T) {
	h := apiHandler{}

	var err error
	var ie *ioterr.Error
	var re request.ErrInvalidParams

	_, err = h.openTunnel(&ist.OpenTunnelInput{
		Tags: []*ist.Tag{},
	})
	if !errors.As(err, &ie) {
		t.Errorf("Expected error type: %T, got: %T", ie, err)
	}
	if !errors.As(err, &re) {
		t.Errorf("Expected error type: %T, got: %T", re, err)
	}

	_, err = h.closeTunnel(&ist.CloseTunnelInput{})
	if !errors.As(err, &ie) {
		t.Errorf("Expected error type: %T, got: %T", ie, err)
	}
	if !errors.As(err, &re) {
		t.Errorf("Expected error type: %T, got: %T", re, err)
	}
}

func newEndpointForFunc(port int) endpoints.Resolver {
	return endpoints.ResolverFunc(func(service, region string, opts ...func(*endpoints.Options)) (endpoints.ResolvedEndpoint, error) {
		return endpoints.ResolvedEndpoint{
			URL:                fmt.Sprintf("http://localhost:%d", port),
			PartitionID:        "clone",
			SigningRegion:      region,
			SigningName:        service,
			SigningNameDerived: true,
			SigningMethod:      "v4",
		}, nil
	})
}
