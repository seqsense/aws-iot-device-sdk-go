// Copyright 2018 SEQSENSE, Inc.
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

package presigner

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
)

func TestPresignWss(t *testing.T) {
	os.Clearenv()
	os.Setenv("AWS_ACCESS_KEY_ID", "AKAAAAAAAAAAAAAAAAAA")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "1111111111111111111111111111111111111111")
	os.Setenv("AWS_SESSION_TOKEN", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	os.Setenv("AWS_REGION", "world-1")

	const expected = "wss://test.iot.world-1.amazonaws.com/mqtt?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=AKAAAAAAAAAAAAAAAAAA%2F19700101%2Fworld-1%2Fiotdevicegateway%2Faws4_request&X-Amz-Date=19700101T000000Z&X-Amz-Expires=86400&X-Amz-SignedHeaders=host&X-Amz-Signature=4cfbc8acc899f7aac3153cd17c94204d6989f86d8cb1173e46143512270c89c2&X-Amz-Security-Token=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	ctx := context.TODO()

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		t.Fatal(err)
	}
	ps := New(&cfg)
	wssURL, err := ps.PresignWss(ctx, "test.iot.world-1.amazonaws.com", time.Hour*24, time.Unix(0, 0))
	if err != nil {
		t.Error(err)
	}

	if wssURL != expected {
		t.Errorf("Presigned URL is wrong if session token is provided.\nactual:   %s\nexpected: %s",
			wssURL, expected)
	}
}

func TestPresignWss_WithoutSessionToken(t *testing.T) {
	os.Clearenv()
	os.Setenv("AWS_ACCESS_KEY_ID", "AKAAAAAAAAAAAAAAAAAA")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "1111111111111111111111111111111111111111")
	os.Setenv("AWS_REGION", "world-1")

	const expected = "wss://test.iot.world-1.amazonaws.com/mqtt?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=AKAAAAAAAAAAAAAAAAAA%2F19700101%2Fworld-1%2Fiotdevicegateway%2Faws4_request&X-Amz-Date=19700101T000000Z&X-Amz-Expires=86400&X-Amz-SignedHeaders=host&X-Amz-Signature=4cfbc8acc899f7aac3153cd17c94204d6989f86d8cb1173e46143512270c89c2"

	ctx := context.TODO()

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		t.Fatal(err)
	}
	ps := New(&cfg)
	wssURL, err := ps.PresignWss(ctx, "test.iot.world-1.amazonaws.com", time.Hour*24, time.Unix(0, 0))
	if err != nil {
		t.Error(err)
	}

	if wssURL != expected {
		t.Errorf("Presigned URL is wrong if session token is provided.\nactual:   %s\nexpected: %s",
			wssURL, expected)
	}
}

func ExamplePresigner_PresignWssNow() {
	ctx := context.TODO()

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		panic(err)
	}
	ps := New(&cfg)
	wssURL, err := ps.PresignWssNow(ctx, "test.iot.world-1.amazonaws.com")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s\n", wssURL)
}
