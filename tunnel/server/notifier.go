package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/at-wat/mqtt-go"

	"github.com/seqsense/aws-iot-device-sdk-go/v4/tunnel"
)

const (
	defaultTopicFormat = "aws/things/%s/tunnels/notify"
)

// Notifier is a token notifier.
type Notifier struct {
	cli         mqtt.Client
	topicFormat string
}

// NewNotifier creates token notifier via MQTT.
func NewNotifier(cli mqtt.Client, opts ...NotifierOption) *Notifier {
	n := &Notifier{
		cli:         cli,
		topicFormat: defaultTopicFormat,
	}
	for _, o := range opts {
		o(n)
	}
	return n
}

func (n *Notifier) notify(ctx context.Context, thingName string, notify *tunnel.Notification) error {
	b, err := json.Marshal(notify)
	if err != nil {
		return err
	}
	return n.cli.Publish(ctx,
		&mqtt.Message{
			Topic:   fmt.Sprintf(n.topicFormat, thingName),
			QoS:     mqtt.QoS1,
			Payload: b,
		},
	)
}

// NotifierOption is a functional option of Notifier.
type NotifierOption func(*Notifier)

// WithNotifyTopicFormat sets notify topic format.
// %s will be replaced by thing name.
func WithNotifyTopicFormat(format string) NotifierOption {
	return func(n *Notifier) {
		n.topicFormat = format
	}
}
