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

package tunnel

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	mqtt "github.com/at-wat/mqtt-go"
	mockmqtt "github.com/at-wat/mqtt-go/mock"

	"github.com/seqsense/aws-iot-device-sdk-go/v5/internal/ioterr"
)

type mockDevice struct {
	*mockmqtt.Client
}

func (d *mockDevice) ThingName() string {
	return "test"
}

func TestNew(t *testing.T) {
	errDummy := errors.New("dummy error")

	t.Run("OptionError", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		cli := &mockDevice{&mockmqtt.Client{}}
		_, err := New(ctx, cli, map[string]Dialer{}, func(opts *Options) error {
			return errDummy
		})
		var ie *ioterr.Error
		if !errors.As(err, &ie) {
			t.Errorf("Expected error type: %T, got: %T", ie, err)
		}
		if !errors.Is(err, errDummy) {
			t.Errorf("Expected error: %v, got: %v", errDummy, err)
		}
	})
	t.Run("SubscribeError", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		cli := &mockDevice{&mockmqtt.Client{
			SubscribeFn: func(ctx context.Context, subs ...mqtt.Subscription) ([]mqtt.Subscription, error) {
				return nil, errDummy
			},
		}}
		_, err := New(ctx, cli, map[string]Dialer{})
		var ie *ioterr.Error
		if !errors.As(err, &ie) {
			t.Errorf("Expected error type: %T, got: %T", ie, err)
		}
		if !errors.Is(err, errDummy) {
			t.Errorf("Expected error: %v, got: %v", errDummy, err)
		}
	})
	t.Run("InvalidTopicFunc", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		cli := &mockDevice{&mockmqtt.Client{}}
		_, err := New(ctx, cli, map[string]Dialer{}, func(opts *Options) error {
			opts.TopicFunc = func(operation string) string {
				return "##"
			}
			return nil
		})
		var ie *ioterr.Error
		if !errors.As(err, &ie) {
			t.Errorf("Expected error type: %T, got: %T", ie, err)
		}
		if !errors.Is(err, mqtt.ErrInvalidTopicFilter) {
			t.Errorf("Expected error: %v, got: %v", mqtt.ErrInvalidTopicFilter, err)
		}
	})

}

func TestHandlers(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	cli := &mockDevice{&mockmqtt.Client{}}
	tu, err := New(ctx, cli, map[string]Dialer{})
	if err != nil {
		t.Fatal(err)
	}

	chErr := make(chan error, 1)
	tu.OnError(func(err error) {
		chErr <- err
	})
	cli.Handle(tu)

	t.Run("Notify", func(t *testing.T) {
		t.Run("NonJSON", func(t *testing.T) {
			cli.Serve(&mqtt.Message{
				Topic:   "$aws/things/test/tunnels/notify",
				Payload: []byte("aaaa{"),
			})
			select {
			case <-ctx.Done():
				t.Fatal("Timeout")
			case err := <-chErr:
				var ie *ioterr.Error
				if !errors.As(err, &ie) {
					t.Errorf("Expected error type: %T, got: %T", ie, err)
				}
				var je *json.SyntaxError
				if !errors.As(err, &je) {
					t.Errorf("Expected error type: %T, got: %T", je, err)
				}
			}
		})
		t.Run("NonDestination", func(t *testing.T) {
			cli.Serve(&mqtt.Message{
				Topic:   "$aws/things/test/tunnels/notify",
				Payload: []byte(`{"ClientMode": "source"}`),
			})
			select {
			case <-ctx.Done():
				t.Fatal("Timeout")
			case err := <-chErr:
				var ie *ioterr.Error
				if !errors.As(err, &ie) {
					t.Errorf("Expected error type: %T, got: %T", ie, err)
				}
				if !errors.Is(err, ErrInvalidClientMode) {
					t.Errorf("Expected error: %v, got: %v", ErrInvalidClientMode, err)
				}
			}
		})
	})
}

func TestHandlers_InvalidResponse(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	var cli *mockDevice
	cli = &mockDevice{Client: &mockmqtt.Client{}}

	s, err := New(ctx, cli, map[string]Dialer{})
	if err != nil {
		t.Fatal(err)
	}
	chErr := make(chan error, 1)
	s.OnError(func(err error) { chErr <- err })
	cli.Handle(s)

	cli.Serve(&mqtt.Message{
		Topic:   s.(*tunnel).topic("notify"),
		Payload: []byte{0xff, 0xff, 0xff},
	})

	select {
	case err := <-chErr:
		var ie *ioterr.Error
		if !errors.As(err, &ie) {
			t.Errorf("Expected error type: %T, got: %T", ie, err)
		}
	case <-ctx.Done():
		t.Fatal("Timeout")
	}
}
