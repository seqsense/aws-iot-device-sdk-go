package devicecli

import (
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/seqsense/aws-iot-device-sdk-go/options"
	"github.com/seqsense/aws-iot-device-sdk-go/protocols"
)

type deviceState int

const (
	inactive deviceState = iota
	established
	stable
	terminating
)

func (s deviceState) String() string {
	switch s {
	case inactive:
		return "inactive"
	case established:
		return "established"
	case stable:
		return "stable"
	case terminating:
		return "terminating"
	default:
		return "unknown"
	}
}

type DeviceClient struct {
	opt             *options.Options
	mqttOpt         *mqtt.ClientOptions
	cli             mqtt.Client
	reconnectPeriod time.Duration
	stateUpdater    chan deviceState
	stableTimer     chan bool
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
		stateUpdater:    make(chan deviceState),
		stableTimer:     make(chan bool),
	}

	connectionLost := func(client mqtt.Client, err error) {
		fmt.Printf("Connection lost (%s)\n", err.Error())
		d.stateUpdater <- inactive
	}
	onConnect := func(client mqtt.Client) {
		fmt.Printf("Connection established\n")
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

func connectionStateHandler(c *DeviceClient) {
	state := inactive
	for {
		select {
		case <-c.stableTimer:
			if state != established {
				panic("Stable timer reached but not in established state")
			}
			fmt.Print("Stable timer reached\n")
			go func() {
				c.stateUpdater <- stable
			}()

		case state = <-c.stateUpdater:
			fmt.Printf("State updated to %s\n", state.String())

			switch state {
			case inactive:
				c.stableTimer = make(chan bool)
				c.reconnectPeriod = c.reconnectPeriod * 2
				if c.reconnectPeriod > c.opt.MaximumReconnectTime {
					c.reconnectPeriod = c.opt.MaximumReconnectTime
				}
				fmt.Printf("Trying to reconnect (%d ms)\n", c.reconnectPeriod/time.Millisecond)

				time.Sleep(c.reconnectPeriod)
				c.connect()

			case established:
				fmt.Print("Processing queued operations\n")
				go func() {
					time.Sleep(c.opt.MinimumConnectionTime)
					c.stableTimer <- true
				}()

			case stable:
				fmt.Print("Stable\n")
				c.reconnectPeriod = c.opt.BaseReconnectTime

			case terminating:
				fmt.Print("Terminating connection\n")
				c.cli.Disconnect(250)
				return

			default:
				panic("Invalid internal state\n")
			}
		}
	}
}

func (s *DeviceClient) connect() {
	s.cli = mqtt.NewClient(s.mqttOpt)

	token := s.cli.Connect()
	go func() {
		token.Wait()
		if token.Error() != nil {
			fmt.Printf("Failed to connect (%s)\n", token.Error())
			s.stateUpdater <- inactive
		}
	}()
}

func (s *DeviceClient) Disconnect() {
	s.stateUpdater <- terminating
}

func (s *DeviceClient) Publish(topic string, message string) {
	// TODO: publish with offline queue
}

func (s *DeviceClient) Subscribe(topic string, cb mqtt.MessageHandler) {
	// TODO: keep subscription and re-subscribe on connect
	token := s.cli.Subscribe(topic, s.opt.Qos, cb)
	go func() {
		token.Wait()
		if token.Error() != nil {
			fmt.Printf("Failed to subscribe (%s)\n", token.Error())
			s.stateUpdater <- inactive
		}
	}()
}
