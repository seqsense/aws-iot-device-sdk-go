package awsiotprotocol

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Mqtts struct {
}

func (s Mqtts) Name() string {
	return "mqtts"
}

func (s Mqtts) NewClientOptions(opt *Config) (*mqtt.ClientOptions, error) {

	tlsconfig, err := newTLSConfig(opt.Host, opt.CaPath, opt.CertPath, opt.KeyPath)
	if err != nil {
		return nil, err
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(
		fmt.Sprintf("ssl://%s:8883", opt.Host))
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
