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
	"bytes"
	"context"
	"crypto/x509"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/at-wat/mqtt-go"
	mockmqtt "github.com/at-wat/mqtt-go/mock"

	"github.com/seqsense/aws-iot-device-sdk-go/v6/internal/ioterr"
)

type mockClient interface {
	mqtt.Client
	mqtt.Handler
}

type mockDevice struct {
	mockClient
	mqtt.Retryer
}

func (d *mockDevice) ThingName() string {
	return "test"
}

func TestNew(t *testing.T) {
	errDummy := errors.New("dummy error")

	t.Run("SubscribeError", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		cli := &mockDevice{mockClient: &mockmqtt.Client{
			SubscribeFn: func(ctx context.Context, subs ...mqtt.Subscription) ([]mqtt.Subscription, error) {
				return nil, errDummy
			},
		}}
		_, err := New(ctx, cli, "template")
		var ie *ioterr.Error
		if !errors.As(err, &ie) {
			t.Errorf("Expected error type: %T, got: %T", ie, err)
		}
		if !errors.Is(err, errDummy) {
			t.Errorf("Expected error: %v, got: %v", errDummy, err)
		}
	})
}

const (
	TEST_KEY               = `-----BEGIN PRIVATE KEY-----\nMIIBVAIBADANBgkqhkiG9w0BAQEFAASCAT4wggE6AgEAAkEA+bT+piDo8Um/LUAe\nDrNBLx7qihAsThuvCn//SPXsoICofAgjAxtu2n1nVEb5ZnxKh8P72KXC4wE7G97u\n0u1tvQIDAQABAkAgiusJAY76Ky9EGXARYGElX/UXCyaLA2abirTdcFdnTzs+19nX\n4OI/jBbiEd76yjfW6RkdAN6aPezDkRnlspSBAiEA/oqwVbNHJc8D5uzARMENt34j\nYVGimAl+I3VwjaxBkd0CIQD7IzccW+L35RGpdnXMooTD+tKRaZv7XnTx5jfI3Sm9\nYQIhAK8idZlBtN5KxYCJvPCRdAKgg29eX+UEAwoar8qKjsLxAiBdv/KlyoN7CO9D\n9K3a+1xWkL6ke+k3uDYty0RN3onjYQIgbGhXwTcEzLoh3OCuuF2KJVJ5gmBANKrP\n+6AHJFrGjHY=\n-----END PRIVATE KEY-----`
	TEST_PUBKEY            = `-----BEGIN PUBLIC KEY-----\nMFwwDQYJKoZIhvcNAQEBBQADSwAwSAJBAPm0/qYg6PFJvy1AHg6zQS8e6ooQLE4b\nrwp//0j17KCAqHwIIwMbbtp9Z1RG+WZ8SofD+9ilwuMBOxve7tLtbb0CAwEAAQ==\n-----END PUBLIC KEY-----`
	TEST_CSR               = `-----BEGIN CERTIFICATE REQUEST-----\nMIIBDjCBuQIBADBUMQswCQYDVQQGEwJBVTETMBEGA1UECAwKU29tZS1TdGF0ZTEh\nMB8GA1UECgwYSW50ZXJuZXQgV2lkZ2l0cyBQdHkgTHRkMQ0wCwYDVQQDDAR0ZXN0\nMFwwDQYJKoZIhvcNAQEBBQADSwAwSAJBAPm0/qYg6PFJvy1AHg6zQS8e6ooQLE4b\nrwp//0j17KCAqHwIIwMbbtp9Z1RG+WZ8SofD+9ilwuMBOxve7tLtbb0CAwEAAaAA\nMA0GCSqGSIb3DQEBCwUAA0EAwM1MsaPCJIadGpsnIbqtNSSvg+F331yvja3kMr0R\nCdtdQ2uymQ5hxv/Qtg30WLdyXQ3XRWfRh1Fb/mkJMG1DGQ==\n-----END CERTIFICATE REQUEST-----`
	TEST_CERT              = `-----BEGIN CERTIFICATE-----\nMIIBpzCCAVECFBcYnnOAmVrDMGVQQNTbHDiDuNinMA0GCSqGSIb3DQEBCwUAMFQx\nCzAJBgNVBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRl\ncm5ldCBXaWRnaXRzIFB0eSBMdGQxDTALBgNVBAMMBHRlc3QwIBcNMjQwNjI0MTQz\nOTEyWhgPMjA1MTExMDkxNDM5MTJaMFQxCzAJBgNVBAYTAkFVMRMwEQYDVQQIDApT\nb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQxDTAL\nBgNVBAMMBHRlc3QwXDANBgkqhkiG9w0BAQEFAANLADBIAkEA+bT+piDo8Um/LUAe\nDrNBLx7qihAsThuvCn//SPXsoICofAgjAxtu2n1nVEb5ZnxKh8P72KXC4wE7G97u\n0u1tvQIDAQABMA0GCSqGSIb3DQEBCwUAA0EAZ+GS/x322f/APuFi1WQHrp5Ebe+S\nTwVHzOhhLLF5xYOp/KlXAk2ObJEov88McOYJG7A3Oc+qI739EX+oGmM3MQ==\n-----END CERTIFICATE-----`
	TEST_CERT_ACCEPTED     = `{"certificateId":"test","certificatePem":"` + TEST_CERT + `","privateKey":"` + TEST_KEY + `","certificateOwnershipToken":"token"}`
	TEST_CSR_ACCEPTED      = `{"certificateId":"test","certificatePem":"` + TEST_CERT + `","certificateOwnershipToken":"token"}`
	TEST_THING_NAME        = "test"
	TEST_REGISTER_ACCEPTED = `{"deviceConfiguration":{},"thingName":"` + TEST_THING_NAME + `"}`
)

func TestHandlers(t *testing.T) {
	t.Run("CreateCertificateAccepted", func(t *testing.T) {

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		var f FleetProvisioning
		cli := &mockDevice{mockClient: &mockmqtt.Client{PublishFn: func(ctx context.Context, msg *mqtt.Message) error {
			f.Serve(&mqtt.Message{
				Topic:   "$aws/certificates/create/json/accepted",
				Payload: []byte(TEST_CERT_ACCEPTED),
			})
			return nil
		}}}
		f, err := New(ctx, cli, "test")
		if err != nil {
			t.Fatal(err)
		}
		cli.Handle(f)
		_, _, _, _, err = f.CreateKeysAndCertificate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		certificate := bytes.NewBuffer(nil)
		f.WriteCertificate(certificate)
		cert := strings.ReplaceAll(certificate.String(), "\n", "\\n")
		if cert != TEST_CERT {
			t.Errorf("Expected certificate: %s, got: %s", TEST_CERT, certificate.String())
		}

	})
	t.Run("CreateCertificateFromCSRAccepted", func(t *testing.T) {

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		var f FleetProvisioning
		cli := &mockDevice{mockClient: &mockmqtt.Client{PublishFn: func(ctx context.Context, msg *mqtt.Message) error {
			f.Serve(&mqtt.Message{
				Topic:   "$aws/certificates/create-from-csr/json/accepted",
				Payload: []byte(TEST_CSR_ACCEPTED),
			})
			return nil
		}}}
		f, err := New(ctx, cli, "test")
		if err != nil {
			t.Fatal(err)
		}
		cli.Handle(f)

		_, _, _, err = f.CreateCertificateFromCSR(ctx, &x509.CertificateRequest{})
		if err != nil {
			t.Fatal(err)
		}
		certificate := bytes.NewBuffer(nil)
		f.WriteCertificate(certificate)
		cert := strings.ReplaceAll(certificate.String(), "\n", "\\n")
		if cert != TEST_CERT {
			t.Errorf("Expected certificate: %s, got: %s", TEST_CERT, certificate.String())
		}
	})
	t.Run("RegisterThingAccepted", func(t *testing.T) {

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		var f FleetProvisioning
		cli := &mockDevice{mockClient: &mockmqtt.Client{PublishFn: func(ctx context.Context, msg *mqtt.Message) error {
			f.Serve(&mqtt.Message{
				Topic:   "$aws/provisioning-templates/test/provision/json/accepted",
				Payload: []byte(TEST_REGISTER_ACCEPTED),
			})
			return nil
		}}}
		f, err := New(ctx, cli, "test")
		if err != nil {
			t.Fatal(err)
		}
		cli.Handle(f)
		thingName, _, err := f.RegisterThing(ctx, "", map[string]string{})
		if err != nil {
			t.Fatal(err)
		}
		if thingName != TEST_THING_NAME {
			t.Errorf("Expected thingName: %s, got: %s", TEST_THING_NAME, thingName)
		}

	})
}

func TestHandlers_InvalidResponse(t *testing.T) {
	for _, topic := range []string{
		"$aws/certificates/create/json/accepted",
		"$aws/certificates/create/json/rejected",
		"$aws/certificates/create-from-csr/json/accepted",
		"$aws/certificates/create-from-csr/json/rejected",
		"$aws/provisioning-templates/test/provision/json/accepted",
		"$aws/provisioning-templates/test/provision/json/rejected",
	} {
		topic := topic
		t.Run(topic, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			var cli *mockDevice
			cli = &mockDevice{mockClient: &mockmqtt.Client{}}

			f, err := New(ctx, cli, "test")
			if err != nil {
				t.Fatal(err)
			}
			chErr := make(chan error, 1)
			f.OnError(func(err error) { chErr <- err })
			cli.Handle(f)

			cli.Serve(&mqtt.Message{
				Topic:   topic,
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
		})
	}
}
