package protocols

import (
	"fmt"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/seqsense/aws-iot-device-sdk-go/options"
	"github.com/seqsense/aws-iot-device-sdk-go/protocols/mqtts"
)

var (
	protocols = make(map[string]Protocol)
)

type Protocol interface {
	Name() string
	NewClientOptions(opt *options.Options) (*mqtt.ClientOptions, error)
}

func init() {
	registerProtocol(mqtts.Mqtts{})
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
