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

/*
Package presigner implements AWS v4 presigner wrapper for AWS IoT websocket connection.
This presigner wrapper works around the AWS IoT websocket's problem in presigned URL with SESSION_TOKEN.
*/
package presigner

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/signer/v4"

	"github.com/seqsense/aws-iot-device-sdk-go/v4/internal/ioterr"
)

// Presigner is an AWS v4 signer wrapper for AWS IoT.
type Presigner struct {
	clientConfig client.Config
}

const serviceName = "iotdevicegateway"

// New returns new AWS v4 signer wrapper for AWS IoT.
func New(p client.ConfigProvider, cfgs ...*aws.Config) *Presigner {
	return &Presigner{
		clientConfig: p.ClientConfig(serviceName, cfgs...),
	}
}

// PresignWssNow generates presigned AWS IoT websocket URL for specified endpoint hostname.
// The URL is valid from now until 24 hours later which is the limit of AWS IoT Websocket connection.
func (a *Presigner) PresignWssNow(endpoint string) (string, error) {
	return a.PresignWss(endpoint, time.Hour*24, time.Now())
}

// PresignWss generates presigned AWS IoT websocket URL for specified endpoint hostname.
func (a *Presigner) PresignWss(endpoint string, expire time.Duration, from time.Time) (string, error) {
	if a.clientConfig.SigningRegion == "" {
		return "", errors.New("Region is not specified")
	}
	cred, err := a.clientConfig.Config.Credentials.Get()
	if err != nil {
		return "", ioterr.New(err, "getting credentials")
	}
	sessionToken := cred.SessionToken

	signer := v4.NewSigner(
		credentials.NewStaticCredentials(cred.AccessKeyID, cred.SecretAccessKey, ""),
	)

	body := bytes.NewReader([]byte{})

	originalURL, err := url.Parse(fmt.Sprintf("wss://%s/mqtt", endpoint))
	if err != nil {
		return "", ioterr.New(err, "parsing server URL")
	}

	req := &http.Request{
		Method: "GET",
		URL:    originalURL,
	}
	_, err = signer.Presign(
		req, body,
		a.clientConfig.SigningName, a.clientConfig.SigningRegion,
		expire, from,
	)
	if err != nil {
		return "", ioterr.New(err, "presigning URL")
	}

	ret := req.URL.String()
	if sessionToken != "" {
		ret = ret + "&X-Amz-Security-Token=" + url.QueryEscape(sessionToken)
	}
	return ret, nil
}
