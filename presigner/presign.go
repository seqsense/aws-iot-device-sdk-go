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
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"

	"github.com/seqsense/aws-iot-device-sdk-go/v5/internal/ioterr"
)

const (
	serviceName      = "iotdevicegateway"
	emptyPayloadHash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
)

// Presigner is an AWS v4 signer wrapper for AWS IoT.
type Presigner struct {
	cfg aws.Config
}

// New returns new AWS v4 signer wrapper for AWS IoT.
func New(c aws.Config) *Presigner {
	return &Presigner{
		cfg: c,
	}
}

// PresignWssNow generates presigned AWS IoT websocket URL for specified endpoint hostname.
// The URL is valid from now until 24 hours later which is the limit of AWS IoT Websocket connection.
func (a *Presigner) PresignWssNow(endpoint string) (string, error) {
	return a.PresignWss(endpoint, time.Hour*24, time.Now())
}

// PresignWss generates presigned AWS IoT websocket URL for specified endpoint hostname.
func (a *Presigner) PresignWss(endpoint string, expire time.Duration, from time.Time) (string, error) {
	if a.cfg.Region == "" {
		return "", errors.New("Region is not specified")
	}
	cred, err := a.cfg.Credentials.Retrieve(context.TODO())
	if err != nil {
		return "", ioterr.New(err, "getting credentials")
	}
	sessionToken := cred.SessionToken
	cred.SessionToken = ""

	originalURL, err := url.Parse(
		fmt.Sprintf("wss://%s/mqtt?X-Amz-Expires=%s", endpoint, strconv.FormatInt(int64(expire/time.Second), 10)),
	)
	if err != nil {
		return "", ioterr.New(err, "parsing server URL")
	}

	signer := v4.NewSigner()

	req := &http.Request{
		Method: "GET",
		URL:    originalURL,
	}
	presignedURL, _, err := signer.PresignHTTP(
		context.TODO(), cred, req, emptyPayloadHash, serviceName, a.cfg.Region, from,
	)
	if err != nil {
		return "", ioterr.New(err, "presigning URL")
	}

	if sessionToken != "" {
		presignedURL += "&X-Amz-Security-Token=" + url.QueryEscape(sessionToken)
	}

	return presignedURL, nil
}
