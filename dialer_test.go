package awsiotdev

import (
	"reflect"
	"testing"

	"github.com/at-wat/mqtt-go"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"

	"github.com/seqsense/aws-iot-device-sdk-go/v4/presigner"
)

type dummyConfigProvider struct{}

func (*dummyConfigProvider) ClientConfig(serviceName string, cfgs ...*aws.Config) client.Config {
	return client.Config{}
}

func TestNewDialer(t *testing.T) {
	cases := map[string]struct {
		url    string
		dialer interface{}
		err    error
	}{
		"MQTTs": {
			url:    "mqtts://hoge.foo:1234",
			dialer: &mqtt.URLDialer{URL: "mqtts://hoge.foo:1234"},
		},
		"WebSockets": {
			url: "wss://hoge.foo:1234/ep",
			dialer: &presignDialer{
				signer:   &presigner.Presigner{},
				endpoint: "hoge.foo:1234",
			},
		},
		"UnknownProtocol": {
			url: "unknown://hoge.foo:1234",
			err: mqtt.ErrUnsupportedProtocol,
		},
	}
	for name, c := range cases {
		c := c
		t.Run(name, func(t *testing.T) {
			d, err := NewDialer(&dummyConfigProvider{}, c.url)
			if err != c.err {
				t.Fatalf("Expected error: %v, got: %v", c.err, err)
			}
			if !reflect.DeepEqual(c.dialer, d) {
				t.Errorf("Expected dialer: %v, got: %v", c.dialer, d)
			}
		})
	}
}
