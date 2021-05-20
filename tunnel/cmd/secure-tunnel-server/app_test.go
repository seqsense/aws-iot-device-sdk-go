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
	"net"
	"net/http"
	"testing"
	"time"
)

func TestApp(t *testing.T) {
	var ports [3]int
	for i := range ports {
		conn, err := net.Listen("tcp", ":0")
		if err != nil {
			t.Fatal(err)
		}
		ports[i] = conn.Addr().(*net.TCPAddr).Port
		conn.Close()
	}

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
				app(ctx, testCase.opts)
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
