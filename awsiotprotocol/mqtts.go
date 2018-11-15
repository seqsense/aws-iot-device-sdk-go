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

package awsiotprotocol

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"strconv"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Mqtts implements Protocol interface for MQTT over TLS using x.509 certifications.
type Mqtts struct {
}

// Name returns the protocol name.
func (s Mqtts) Name() string {
	return "mqtts"
}

// NewClientOptions returns MQTT connection options.
func (s Mqtts) NewClientOptions(opt *Config) (*mqtt.ClientOptions, error) {
	url, err := url.Parse(opt.Url)
	if err != nil {
		return nil, err
	}
	host := url.Hostname()
	if host == "" {
		return nil, errors.New("Hostname is not provided in the URL")
	}
	var port int
	if url.Port() != "" {
		port, err = strconv.Atoi(url.Port())
		if err != nil {
			return nil, err
		}
	} else {
		port = 8883
	}

	tlsconfig, err := newTLSConfig(host, opt.CaPath, opt.CertPath, opt.KeyPath)
	if err != nil {
		return nil, err
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(
		fmt.Sprintf("ssl://%s:%d", host, port))
	opts.SetClientID(opt.ClientId)
	opts.SetTLSConfig(tlsconfig)

	return opts, nil
}

func newTLSConfig(host, caFile, crtFile, keyFile string) (*tls.Config, error) {
	certpool := x509.NewCertPool()
	if cas, err := ioutil.ReadFile(caFile); err == nil {
		certpool.AppendCertsFromPEM(cas)
	}

	cert, err := tls.LoadX509KeyPair(crtFile, keyFile)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		RootCAs:            certpool,
		ClientAuth:         tls.NoClientCert,
		ClientCAs:          nil,
		InsecureSkipVerify: false,
		ServerName:         host,
		Certificates:       []tls.Certificate{cert},
	}, nil
}
