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
	"net/url"

	"github.com/aws/aws-sdk-go/aws/client"

	"github.com/at-wat/mqtt-go"
	presigner "github.com/seqsense/aws-iot-device-sdk-go/v3/presigner"
)

type presignDialer struct {
	signer   *presigner.Presigner
	endpoint string
	opts     []mqtt.DialOption
}

// NewPresignDialer returns WebSockets Dialer with AWS v4 presigned URL.
func NewPresignDialer(sess client.ConfigProvider, endpoint string, opts ...mqtt.DialOption) (mqtt.Dialer, error) {
	return &presignDialer{
		signer:   presigner.New(sess),
		endpoint: endpoint,
		opts:     opts,
	}, nil
}

func (d *presignDialer) Dial() (mqtt.ClientCloser, error) {
	// Presign URL here.
	url, err := d.signer.PresignWssNow(d.endpoint)
	if err != nil {
		return nil, err
	}
	cli, err := mqtt.Dial(url, d.opts...)
	if err != nil {
		return nil, err
	}
	return cli, nil
}

// NewDialer creates default dialer for the given URL for AWS IoT.
// Supported protocols are mqtts and wss (with presigned URL).
func NewDialer(sess client.ConfigProvider, urlStr string, opts ...mqtt.DialOption) (mqtt.Dialer, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}
	switch u.Scheme {
	case "mqtts":
		return &mqtt.URLDialer{URL: urlStr, Options: opts}, nil
	case "wss":
		return NewPresignDialer(sess, u.Host)
	default:
		return nil, mqtt.ErrUnsupportedProtocol
	}
}
