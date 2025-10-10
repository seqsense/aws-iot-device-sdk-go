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

package fleetprovisioning

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"sync"

	"github.com/at-wat/mqtt-go"
	awsiotdev "github.com/seqsense/aws-iot-device-sdk-go/v6"
	"github.com/seqsense/aws-iot-device-sdk-go/v6/internal/ioterr"
)

type FleetProvisioning interface {
	mqtt.Handler
	// CreateKeysAndCertificate creates keys and certificate.
	CreateKeysAndCertificate(ctx context.Context) (string, *x509.Certificate, *rsa.PrivateKey, string, error)
	// CreateCertificateFromCSR creates certificate from CSR.
	CreateCertificateFromCSR(ctx context.Context, csr *x509.CertificateRequest) (string, *x509.Certificate, string, error)
	// RegisterThing registers a thing.
	RegisterThing(ctx context.Context, certToken string, parameters map[string]string) (string, map[string]string, error)
	// WriteCertificate writes PEM certificate to writer.
	WriteCertificate(writer io.Writer) (int, error)
	// WritePrivateKey writes PEM private key to writer.
	WritePrivateKey(writer io.Writer) (int, error)
	// OnError sets handler of asyncronous errors.
	OnError(func(err error))
}

type fleetProvisioning struct {
	mqtt.ServeMux
	cli          mqtt.Client
	templateName string
	pemCert      []byte
	pemKey       []byte

	mu      sync.Mutex
	onError func(err error)

	chResps map[string]chan interface{}
}

const (
	// Tokens for response handling.
	tokenCreateCertificate = "certificates/create"
	tokenCreateFromCSR     = "certificates/create-from-csr"
	tokenRegisterThing     = "provisioning-templates/provision"
)

func New(ctx context.Context, cli awsiotdev.Device, templateName string) (FleetProvisioning, error) {
	f := &fleetProvisioning{
		cli:          cli,
		chResps:      make(map[string]chan interface{}),
		templateName: templateName,
	}
	for _, sub := range []struct {
		topic string
		hand  mqtt.Handler
	}{
		{"$aws/certificates/create/json/accepted", mqtt.HandlerFunc(f.certificateCreateAccepted)},
		{"$aws/certificates/create/json/rejected", mqtt.HandlerFunc(f.rejectedCertificateCreate)},
		{"$aws/certificates/create-from-csr/json/accepted", mqtt.HandlerFunc(f.certificateCreateFromCSRAccepted)},
		{"$aws/certificates/create-from-csr/json/rejected", mqtt.HandlerFunc(f.rejectedCertificatesCreateFromCSR)},
		{"$aws/provisioning-templates/" + templateName + "/provision/json/accepted", mqtt.HandlerFunc(f.registerThingAccepted)},
		{"$aws/provisioning-templates/" + templateName + "/provision/json/rejected", mqtt.HandlerFunc(f.rejectedProvisioningTemplatesProvision)},
	} {
		if err := f.ServeMux.Handle(sub.topic, sub.hand); err != nil {
			return nil, ioterr.New(err, "registering message handlers")
		}
	}

	_, err := cli.Subscribe(ctx,
		mqtt.Subscription{Topic: "$aws/certificates/create/json/accepted", QoS: mqtt.QoS1},
		mqtt.Subscription{Topic: "$aws/certificates/create/json/rejected", QoS: mqtt.QoS1},
		mqtt.Subscription{Topic: "$aws/certificates/create-from-csr/json/accepted", QoS: mqtt.QoS1},
		mqtt.Subscription{Topic: "$aws/certificates/create-from-csr/json/rejected", QoS: mqtt.QoS1},
		mqtt.Subscription{Topic: "$aws/provisioning-templates/" + templateName + "/provision/json/accepted", QoS: mqtt.QoS1},
		mqtt.Subscription{Topic: "$aws/provisioning-templates/" + templateName + "/provision/json/rejected", QoS: mqtt.QoS1},
	)
	if err != nil {
		return nil, ioterr.New(err, "subscribing to topics")
	}
	return f, nil
}

func parseCertificate(certPem []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(certPem)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("failed to decode PEM block")
	}
	return x509.ParseCertificate(block.Bytes)
}

func parsePrivateKey(keyPem []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(keyPem)
	if block == nil || (block.Type != "PRIVATE KEY" && block.Type != "RSA PRIVATE KEY") {
		return nil, fmt.Errorf("failed to decode PEM block")
	}
	key, _ := x509.ParsePKCS8PrivateKey(block.Bytes)
	pkcs8, ok := key.(*rsa.PrivateKey)
	if ok {
		return pkcs8, nil
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

// CreateKeysAndCertificate creates keys and certificate.
func (f *fleetProvisioning) CreateKeysAndCertificate(ctx context.Context) (certID string, cert *x509.Certificate, privateKey *rsa.PrivateKey, certToken string, err error) {
	errReturn := func(err error) (string, *x509.Certificate, *rsa.PrivateKey, string, error) {
		return "", &x509.Certificate{}, &rsa.PrivateKey{}, "", err
	}
	token := tokenCreateCertificate
	ch := make(chan interface{}, 1)
	f.mu.Lock()
	f.chResps[token] = ch
	f.mu.Unlock()
	defer func() {
		f.mu.Lock()
		delete(f.chResps, token)
		f.mu.Unlock()
	}()
	if err := f.cli.Publish(ctx, &mqtt.Message{
		Topic:   "$aws/certificates/create/json",
		QoS:     mqtt.QoS1,
		Payload: json.RawMessage(fmt.Sprintf(`{}`)),
	}); err != nil {
		return errReturn(ioterr.New(err, "publishing create certificate request"))
	}

	select {
	case <-ctx.Done():
		return errReturn(ioterr.New(ctx.Err(), "waiting for response"))
	case resp := <-ch:
		switch r := resp.(type) {
		case *CertificateCreateResponse:
			cert, err := parseCertificate([]byte(r.CertificatePEM))
			if err != nil {
				return errReturn(ioterr.New(err, "parsing certificate"))
			}
			key, err := parsePrivateKey([]byte(r.PrivateKey))
			if err != nil {
				return errReturn(ioterr.New(err, "parsing private key"))
			}
			f.mu.Lock()
			f.pemCert = []byte(r.CertificatePEM)
			f.pemKey = []byte(r.PrivateKey)
			f.mu.Unlock()
			return r.CertificateID, cert, key, r.CertificateOwnershipToken, nil
		case *ErrorResponse:
			return errReturn(ioterr.New(fmt.Errorf("%d: %s", r.Code, r.Message), "creating certificate"))
		default:
			return errReturn(ioterr.New(fmt.Errorf("unexpected response: %T", r), "parsing response"))
		}
	}
}

func (f *fleetProvisioning) CreateCertificateFromCSR(ctx context.Context, csr *x509.CertificateRequest) (certID string, cert *x509.Certificate, certToken string, err error) {
	errReturn := func(err error) (string, *x509.Certificate, string, error) {
		return "", &x509.Certificate{}, "", err
	}
	token := tokenCreateFromCSR
	ch := make(chan interface{}, 1)
	f.mu.Lock()
	f.chResps[token] = ch
	f.mu.Unlock()
	defer func() {
		f.mu.Lock()
		delete(f.chResps, token)
		f.mu.Unlock()
	}()
	pemCSR := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csr.Raw})
	req := &CertificateCreateFromCSRRequest{
		CertificateSigningRequest: string(pemCSR),
	}
	jsonReq, err := json.Marshal(req)
	if err != nil {
		return errReturn(ioterr.New(err, "marshaling create certificate from CSR request"))
	}
	if err := f.cli.Publish(ctx, &mqtt.Message{
		Topic:   "$aws/certificates/create-from-csr/json",
		QoS:     mqtt.QoS1,
		Payload: jsonReq,
	}); err != nil {
		return errReturn(ioterr.New(err, "publishing create certificate request"))
	}

	select {
	case <-ctx.Done():
		return errReturn(ioterr.New(ctx.Err(), "waiting for response"))
	case resp := <-ch:
		switch r := resp.(type) {
		case *CertificateCreateFromCSRResponse:
			cert, err := parseCertificate([]byte(r.CertificatePEM))
			if err != nil {
				return errReturn(ioterr.New(err, "parsing certificate"))
			}
			f.mu.Lock()
			f.pemCert = []byte(r.CertificatePEM)
			f.mu.Unlock()
			return r.CertificateID, cert, r.CertificateOwnershipToken, nil
		case *ErrorResponse:
			return errReturn(ioterr.New(fmt.Errorf("%d: %s", r.Code, r.Message), "creating certificate"))
		default:
			return errReturn(ioterr.New(fmt.Errorf("unexpected response: %T", r), "parsing response"))
		}
	}
}

func (f *fleetProvisioning) RegisterThing(ctx context.Context, certToken string, parameters map[string]string) (thingName string, deviceConfig map[string]string, err error) {
	errReturn := func(err error) (string, map[string]string, error) {
		return "", map[string]string{}, err
	}
	token := tokenRegisterThing
	ch := make(chan interface{}, 1)
	f.mu.Lock()
	f.chResps[token] = ch
	f.mu.Unlock()
	defer func() {
		f.mu.Lock()
		delete(f.chResps, token)
		f.mu.Unlock()
	}()
	jsonReq, err := json.Marshal(&RegisterThingRequest{
		CertificateOwnershipToken: certToken,
		Parameters:                parameters,
	})
	if err != nil {
		return errReturn(ioterr.New(err, "marshaling register thing request"))
	}
	if err := f.cli.Publish(ctx, &mqtt.Message{
		Topic:   "$aws/provisioning-templates/" + f.templateName + "/provision/json",
		QoS:     mqtt.QoS1,
		Payload: jsonReq,
	}); err != nil {
		return errReturn(ioterr.New(err, "publishing register thing request"))
	}

	select {
	case <-ctx.Done():
		return errReturn(ioterr.New(ctx.Err(), "waiting for response"))
	case resp := <-ch:
		switch r := resp.(type) {
		case *RegisterThingResponse:
			return r.ThingName, r.DeviceConfiguration, nil
		case *ErrorResponse:
			return errReturn(ioterr.New(fmt.Errorf("%d: %s", r.Code, r.Message), "register thing failed"))
		default:
			return errReturn(ioterr.New(fmt.Errorf("unexpected response: %T", r), "parsing response"))
		}
	}
}

func (p *fleetProvisioning) WriteCertificate(writer io.Writer) (int, error) {
	if p.pemCert == nil {
		return 0, fmt.Errorf("certificate not created")
	}
	return writer.Write(p.pemCert)
}

func (p *fleetProvisioning) WritePrivateKey(writer io.Writer) (int, error) {
	if p.pemKey == nil {
		return 0, fmt.Errorf("private key not created")
	}
	return writer.Write(p.pemKey)
}

func (f *fleetProvisioning) certificateCreateAccepted(msg *mqtt.Message) {
	r := &CertificateCreateResponse{}
	if err := json.Unmarshal(msg.Payload, r); err != nil {
		fmt.Printf(string(msg.Payload))
		f.handleError(ioterr.New(err, "unmarshaling certificate create response"))
		return
	}
	f.handleResponse(r, tokenCreateCertificate)
}

func (f *fleetProvisioning) certificateCreateFromCSRAccepted(msg *mqtt.Message) {
	r := &CertificateCreateFromCSRResponse{}
	if err := json.Unmarshal(msg.Payload, r); err != nil {
		f.handleError(ioterr.New(err, "unmarshaling certificate create from CSR response"))
		return
	}
	f.handleResponse(r, tokenCreateFromCSR)
}

func (f *fleetProvisioning) registerThingAccepted(msg *mqtt.Message) {
	r := &RegisterThingResponse{}
	if err := json.Unmarshal(msg.Payload, r); err != nil {
		f.handleError(ioterr.New(err, "unmarshaling register thing response"))
		return
	}
	f.handleResponse(r, tokenRegisterThing)
}

func (f *fleetProvisioning) rejectedCertificateCreate(msg *mqtt.Message) {
	f.rejected(msg, tokenCreateCertificate)
}
func (f *fleetProvisioning) rejectedCertificatesCreateFromCSR(msg *mqtt.Message) {
	f.rejected(msg, tokenCreateFromCSR)
}
func (f *fleetProvisioning) rejectedProvisioningTemplatesProvision(msg *mqtt.Message) {
	f.rejected(msg, tokenRegisterThing)
}

func (f *fleetProvisioning) rejected(msg *mqtt.Message, token string) {
	e := &ErrorResponse{}
	if err := json.Unmarshal(msg.Payload, e); err != nil {
		f.handleError(ioterr.New(err, "unmarshaling error response"))
		return
	}
	f.handleResponse(e, token)
}

func (f *fleetProvisioning) handleResponse(r interface{}, token string) {
	f.mu.Lock()
	ch, ok := f.chResps[token]
	f.mu.Unlock()
	if !ok {
		return
	}
	select {
	case ch <- r:
	default:
	}
}

func (f *fleetProvisioning) OnError(cb func(err error)) {
	f.mu.Lock()
	f.onError = cb
	f.mu.Unlock()
}

func (f *fleetProvisioning) handleError(err error) {
	f.mu.Lock()
	cb := f.onError
	f.mu.Unlock()
	if cb != nil {
		cb(err)
	}
}
