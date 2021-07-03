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
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"testing"
	"time"
)

func getPorts(t *testing.T, i int) []int {
	ports := make([]int, i)
	for i := range ports {
		conn, err := net.Listen("tcp", ":0")
		if err != nil {
			t.Fatal(err)
		}
		ports[i] = conn.Addr().(*net.TCPAddr).Port
		conn.Close()
	}
	return ports
}

func TestApp(t *testing.T) {
	ports := getPorts(t, 3)

	testCases := map[string]struct {
		opts []string
		urls []string
	}{
		"SinglePort": {
			opts: []string{
				"test",
				fmt.Sprintf("-tunnel-addr=:%d", ports[0]),
				fmt.Sprintf("-api-addr=:%d", ports[0]),
			},
			urls: []string{
				fmt.Sprintf("http://localhost:%d/healthcheck", ports[0]),
			},
		},
		"SeparatePort": {
			opts: []string{
				"test",
				fmt.Sprintf("-tunnel-addr=:%d", ports[1]),
				fmt.Sprintf("-api-addr=:%d", ports[2]),
			},
			urls: []string{
				fmt.Sprintf("http://localhost:%d/healthcheck", ports[1]),
				fmt.Sprintf("http://localhost:%d/healthcheck", ports[2]),
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			chExit := make(chan struct{})

			go func() {
				if err := app(ctx, testCase.opts); err != nil {
					t.Error(err)
				}
				close(chExit)
			}()

			select {
			case <-time.After(100 * time.Millisecond):
			case <-chExit:
				t.Fatal("Application exit")
			}
			for _, url := range testCase.urls {
				resp, err := http.Get(url)
				if err != nil {
					t.Errorf("Healthcheck failed: %v", err)
					continue
				}
				if resp.StatusCode != http.StatusOK {
					t.Errorf("Healthcheck failed: %v", resp)
				}
			}
			cancel()

			<-chExit
		})
	}
}

func TestApp_generateTestToken(t *testing.T) {
	ports := getPorts(t, 1)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	chExit := make(chan struct{})

	buf := &bytes.Buffer{}
	log.SetOutput(buf)
	origFlags := log.Flags()
	log.SetFlags(0)
	defer func() {
		log.SetOutput(os.Stdout)
		log.SetFlags(origFlags)
	}()

	go func() {
		if err := app(ctx,
			[]string{
				"test",
				fmt.Sprintf("-tunnel-addr=:%d", ports[0]),
				fmt.Sprintf("-api-addr=:%d", ports[0]),
				"-generate-test-token",
			},
		); err != nil {
			t.Error(err)
		}
		close(chExit)
	}()

	select {
	case <-time.After(100 * time.Millisecond):
	case <-chExit:
		t.Fatal("Application exit")
	}
	cancel()

	raw := buf.Bytes()
	data := raw[bytes.Index(raw, []byte{'{'}):]
	t.Log(string(data))
	out := struct {
		DestinationAccessToken string `json:"destinationAccessToken"`
		SourceAccessToken      string `json:"sourceAccessToken"`
		TunnelARN              string `json:"tunnelArn"`
		TunnelID               string `json:"tunnelId"`
	}{}
	err := json.Unmarshal(data, &out)
	if err != nil {
		t.Fatal(err)
	}

	if len(out.DestinationAccessToken) == 0 {
		t.Errorf("DestinationAccessToken must be returned")
	}
	if len(out.SourceAccessToken) == 0 {
		t.Errorf("SourceAccessToken must be returned")
	}
	if len(out.TunnelARN) == 0 {
		t.Errorf("TunnelArn must be returned")
	}
	if len(out.TunnelID) == 0 {
		t.Errorf("TunnelId must be returned")
	}

	<-chExit
}

func TestApp_error(t *testing.T) {
	ports := getPorts(t, 2)
	t.Run("DialTimeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		os.Clearenv()
		os.Setenv("AWS_DEFAULT_REGION", "none-none-1")
		os.Setenv("AWS_ACCESS_KEY_ID", "AKIAAAAAAAAAAAAAAAAA")
		os.Setenv("SECRET_ACCESS_KEY", "AAAAAAAAAAAAAAAAAAAAAAAAAAAA")

		err := app(ctx, []string{
			"test",
			fmt.Sprintf("-tunnel-addr=:%d", ports[0]),
			fmt.Sprintf("-api-addr=:%d", ports[0]),
			fmt.Sprintf("-mqtt-endpoint=localhost:%d", ports[1]),
		})
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Expected error: '%v', got: '%v'", context.DeadlineExceeded, err)
		}
	})
}
