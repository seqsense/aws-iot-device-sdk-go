package awsiotdev

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"testing"
	"time"

	"golang.org/x/net/websocket"

	"github.com/at-wat/mqtt-go"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/seqsense/aws-iot-device-sdk-go/v4/internal/ioterr"
	"github.com/seqsense/aws-iot-device-sdk-go/v4/presigner"
)

type dummyConfigProvider struct{}

func (*dummyConfigProvider) ClientConfig(serviceName string, cfgs ...*aws.Config) client.Config {
	return client.Config{}
}

func TestNewDialer(t *testing.T) {
	t.Run("ValidURL", func(t *testing.T) {
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
				if !errors.Is(err, c.err) {
					var ie *ioterr.Error
					if !errors.As(err, &ie) {
						t.Errorf("Expected error type: ioterr.Error, actual: %T", err)
					}
					t.Fatalf("Expected error: %v, got: %v", c.err, err)
				}
				if !reflect.DeepEqual(c.dialer, d) {
					t.Errorf("Expected dialer: %v, got: %v", c.dialer, d)
				}
			})
		}
	})
	t.Run("InvalidURL", func(t *testing.T) {
		_, err := NewDialer(&dummyConfigProvider{}, ":aaa")
		var ie *ioterr.Error
		if !errors.As(err, &ie) {
			t.Errorf("Expected error type: %T, actual: %T", ie, err)
		}
		var ue *url.Error
		if !errors.As(err, &ue) {
			t.Errorf("Expected error type: %T, actual: %T", ue, err)
		}
	})
}

func TestPresignDialer(t *testing.T) {
	os.Clearenv()
	os.Setenv("AWS_ACCESS_KEY_ID", "AKAAAAAAAAAAAAAAAAAA")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "1111111111111111111111111111111111111111")
	os.Setenv("AWS_REGION", "world-1")

	sess := session.Must(session.NewSession())
	ps := presigner.New(sess)

	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	wsConfig, err := websocket.NewConfig("wss://", "wss://")
	if err != nil {
		t.Fatal(err)
	}
	wsSrv := websocket.Server{
		Config:  *wsConfig,
		Handler: func(c *websocket.Conn) {},
	}
	chAccept := make(chan bool, 1)
	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !<-chAccept {
				http.NotFound(w, r)
				return
			}
			wsSrv.ServeHTTP(w, r)
		}),
	}
	cert, priv, err := generateSelfSignedCert()
	if err != nil {
		t.Fatal(err)
	}
	go srv.ServeTLS(ln, cert, priv)
	defer srv.Shutdown(context.Background())

	d := &presignDialer{
		signer:   ps,
		endpoint: ln.Addr().String(),
		opts:     []mqtt.DialOption{mqtt.WithTLSConfig(&tls.Config{InsecureSkipVerify: true})},
	}

	t.Run("Success", func(t *testing.T) {
		chAccept <- true
		conn, err := d.Dial()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		_ = conn.Close()
	})
	t.Run("Error", func(t *testing.T) {
		chAccept <- false
		conn, err := d.Dial()
		if err == nil {
			_ = conn.Close()
			t.Fatalf("Dial should fail")
		}
		var ie *ioterr.Error
		if !errors.As(err, &ie) {
			t.Fatalf("Expected error type: %T, actual: %T", ie, err)
		}
	})
}

func generateSelfSignedCert() (string, string, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", err
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Foo"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return "", "", err
	}
	certFile, err := ioutil.TempFile("", "*.pub")
	if err != nil {
		return "", "", err
	}
	privFile, err := ioutil.TempFile("", "*.key")
	if err != nil {
		return "", "", err
	}
	if err := pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: der}); err != nil {
		return "", "", err
	}
	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return "", "", err
	}
	if err := pem.Encode(privFile, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		return "", "", err
	}
	return certFile.Name(), privFile.Name(), nil
}
