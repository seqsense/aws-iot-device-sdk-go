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
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	ist "github.com/aws/aws-sdk-go-v2/service/iotsecuretunneling"
	ist_types "github.com/aws/aws-sdk-go-v2/service/iotsecuretunneling/types"

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

	cfg := aws.Config{
		Region:           "nothing",
		EndpointResolver: newEndpointForFunc(ln.Addr().(*net.TCPAddr).Port),
		Credentials: credentials.NewStaticCredentialsProvider(
			"ASIAZZZZZZZZZZZZZZZZ",
			"0000000000000000000000000000000000000000",
			"",
		),
	}
	api := ist.NewFromConfig(cfg)
	out, err := api.OpenTunnel(context.TODO(), &ist.OpenTunnelInput{
		Description: aws.String("desc"),
		DestinationConfig: &ist_types.DestinationConfig{
			Services: []string{
				"ssh",
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

	_, err = h.openTunnel(&ist.OpenTunnelInput{
		Tags: []ist_types.Tag{},
	})
	if !errors.As(err, &ie) {
		t.Errorf("Expected error type: %T, got: %T", ie, err)
	}
	if !errors.Is(err, errInvalidRequest) {
		t.Errorf("Expected error: '%v', got: '%v'", errInvalidRequest, err)
	}

	_, err = h.closeTunnel(&ist.CloseTunnelInput{})
	if !errors.As(err, &ie) {
		t.Errorf("Expected error type: %T, got: %T", ie, err)
	}
	if !errors.Is(err, errInvalidRequest) {
		t.Errorf("Expected error: '%v', got: '%v'", errInvalidRequest, err)
	}
}

func newEndpointForFunc(port int) aws.EndpointResolver {
	return aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:           fmt.Sprintf("http://localhost:%d", port),
			PartitionID:   "clone",
			SigningRegion: region,
			SigningName:   service,
			SigningMethod: "v4",
		}, nil
	})
}
