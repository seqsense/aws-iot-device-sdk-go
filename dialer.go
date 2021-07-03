// Copyright 2019 SEQSENSE, Inc.
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

package awsiotdev

import (
	"context"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/at-wat/mqtt-go"

	"github.com/seqsense/aws-iot-device-sdk-go/v6/internal/ioterr"
	presigner "github.com/seqsense/aws-iot-device-sdk-go/v6/presigner"
)

type presignDialer struct {
	signer   *presigner.Presigner
	endpoint string
	opts     []mqtt.DialOption
}

// NewPresignDialer returns WebSockets Dialer with AWS v4 presigned URL.
func NewPresignDialer(cfg *aws.Config, endpoint string, opts ...mqtt.DialOption) (mqtt.Dialer, error) {
	return &presignDialer{
		signer:   presigner.New(cfg),
		endpoint: endpoint,
		opts:     opts,
	}, nil
}

func (d *presignDialer) DialContext(ctx context.Context) (*mqtt.BaseClient, error) {
	url, err := d.signer.PresignWssNow(ctx, d.endpoint)
	if err != nil {
		return nil, ioterr.New(err, "presigning wss URL")
	}
	cli, err := mqtt.DialContext(ctx, url, d.opts...)
	if err != nil {
		return nil, ioterr.New(err, "dialing")
	}
	return cli, nil
}

// NewDialer creates default dialer for the given URL for AWS IoT.
// Supported protocols are mqtts and wss (with presigned URL).
func NewDialer(cfg *aws.Config, urlStr string, opts ...mqtt.DialOption) (mqtt.Dialer, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, ioterr.New(err, "parsing server URL")
	}
	switch u.Scheme {
	case "mqtts":
		return &mqtt.URLDialer{URL: urlStr, Options: opts}, nil
	case "wss":
		return NewPresignDialer(cfg, u.Host)
	default:
		return nil, ioterr.New(mqtt.ErrUnsupportedProtocol, "new dialer")
	}
}
