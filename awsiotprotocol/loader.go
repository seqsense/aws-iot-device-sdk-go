package awsiotprotocol

import (
	"fmt"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var (
	protocols = make(map[string]Protocol)
)

type Protocol interface {
	Name() string
	NewClientOptions(opt *Config) (*mqtt.ClientOptions, error)
}

func init() {
	registerProtocol(Mqtts{})
}

func ByName(name string) (Protocol, error) {
	p, ok := protocols[name]
	if !ok {
		return nil, fmt.Errorf("Protocol \"%s\" is not supported", name)
	}
	return p, nil
}

func registerProtocol(p Protocol) {
	n := p.Name()
	protocols[n] = p
}
