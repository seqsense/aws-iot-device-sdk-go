package devicecli

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/seqsense/aws-iot-device-sdk-go/options"
	"github.com/seqsense/aws-iot-device-sdk-go/protocols"
)

type DeviceClient struct {
	opt *options.Options
	cli mqtt.Client
}

func New(opt *options.Options) *DeviceClient {
	p, err := protocols.ByName(opt.Protocol)
	if err != nil {
		panic(err)
	}
	mqttOpt, err := p.NewClientOptions(opt)
	if err != nil {
		panic(err)
	}
	cli := mqtt.NewClient(mqttOpt)

	return &DeviceClient{
		opt,
		cli,
	}
}
