package devicecli

import (
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/seqsense/aws-iot-device-sdk-go/options"
	"github.com/seqsense/aws-iot-device-sdk-go/protocols"
	"github.com/seqsense/aws-iot-device-sdk-go/pubqueue"
	"github.com/seqsense/aws-iot-device-sdk-go/subqueue"
)

const (
	stateUpdaterChCap = 100
	publishChCap      = 100
	subscribeChCap    = 100
)

type DeviceClient struct {
	opt             *options.Options
	mqttOpt         *mqtt.ClientOptions
	cli             mqtt.Client
	reconnectPeriod time.Duration
	stateUpdateCh   chan deviceState
	stableTimerCh   chan bool
	publishCh       chan *pubqueue.Data
	subscribeCh     chan *subqueue.Subscription
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
		stateUpdateCh:   make(chan deviceState, stateUpdaterChCap),
		stableTimerCh:   make(chan bool),
		publishCh:       make(chan *pubqueue.Data, publishChCap),
		subscribeCh:     make(chan *subqueue.Subscription, subscribeChCap),
	}

	connectionLost := func(client mqtt.Client, err error) {
		log.Printf("Connection lost (%s)\n", err.Error())
		d.stateUpdateCh <- inactive
	}
	onConnect := func(client mqtt.Client) {
		log.Printf("Connection established\n")
		d.stateUpdateCh <- established
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
	go connectionHandler(d)

	return d
}

func (s *DeviceClient) connect() {
	s.cli = mqtt.NewClient(s.mqttOpt)

	token := s.cli.Connect()
	go func() {
		token.Wait()
		if token.Error() != nil {
			log.Printf("Failed to connect (%s)\n", token.Error())
			s.stateUpdateCh <- inactive
		}
	}()
}

func (s *DeviceClient) Disconnect() {
	s.stateUpdateCh <- terminating
}

func (s *DeviceClient) Publish(topic string, payload interface{}) {
	s.publishCh <- &pubqueue.Data{topic, payload}
}

func (s *DeviceClient) Subscribe(topic string, cb mqtt.MessageHandler) {
	s.subscribeCh <- &subqueue.Subscription{subqueue.Subscribe, topic, cb}
}
func (s *DeviceClient) Unsubscribe(topic string) {
	s.subscribeCh <- &subqueue.Subscription{subqueue.Unsubscribe, topic, nil}
}
