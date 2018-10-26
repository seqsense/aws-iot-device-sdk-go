package devicecli

import (
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/seqsense/aws-iot-device-sdk-go/options"
	"github.com/seqsense/aws-iot-device-sdk-go/protocols"
	"github.com/seqsense/aws-iot-device-sdk-go/pubqueue"
)

const (
	stateUpdaterQueue = 100
	publishQueue      = 100
)

type DeviceClient struct {
	opt             *options.Options
	mqttOpt         *mqtt.ClientOptions
	cli             mqtt.Client
	reconnectPeriod time.Duration
	stateUpdater    chan deviceState
	stableTimer     chan bool
	publish         chan *pubqueue.Data
	pubQueue        *pubqueue.Queue
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

	d := &DeviceClient{
		opt:             opt,
		mqttOpt:         mqttOpt,
		cli:             nil,
		reconnectPeriod: opt.BaseReconnectTime,
		stateUpdater:    make(chan deviceState, stateUpdaterQueue),
		stableTimer:     make(chan bool),
		publish:         make(chan *pubqueue.Data, publishQueue),
		pubQueue:        &pubqueue.Queue{},
	}

	connectionLost := func(client mqtt.Client, err error) {
		log.Printf("Connection lost (%s)\n", err.Error())
		d.stateUpdater <- inactive
	}
	onConnect := func(client mqtt.Client) {
		log.Printf("Connection established\n")
		d.stateUpdater <- established
	}

	d.mqttOpt.OnConnectionLost = connectionLost
	d.mqttOpt.OnConnect = onConnect
	if opt.Will != nil {
		d.mqttOpt.SetWill(opt.Will.Topic, opt.Will.Payload, opt.Qos, opt.Retain)
	}
	d.mqttOpt.SetKeepAlive(opt.Keepalive)
	d.mqttOpt.SetAutoReconnect(false) // MQTT AutoReconnect doesn't work well for mqtts
	d.mqttOpt.SetConnectTimeout(time.Second * 5)

	d.connect()
	go connectionStateHandler(d)

	return d
}

func (s *DeviceClient) connect() {
	s.cli = mqtt.NewClient(s.mqttOpt)

	token := s.cli.Connect()
	go func() {
		token.Wait()
		if token.Error() != nil {
			log.Printf("Failed to connect (%s)\n", token.Error())
			s.stateUpdater <- inactive
		}
	}()
}

func (s *DeviceClient) Disconnect() {
	s.stateUpdater <- terminating
}

func (s *DeviceClient) Publish(topic string, payload interface{}) {
	s.publish <- &pubqueue.Data{topic, payload}
}

func (s *DeviceClient) Subscribe(topic string, cb mqtt.MessageHandler) {
	// TODO: keep subscription and re-subscribe on connect
	token := s.cli.Subscribe(topic, s.opt.Qos, cb)
	go func() {
		token.Wait()
		if token.Error() != nil {
			log.Printf("Failed to subscribe (%s)\n", token.Error())
			s.stateUpdater <- inactive
		}
	}()
}
