package options

import (
	"time"

	"github.com/seqsense/aws-iot-device-sdk-go/pubqueue"
)

type TopicPayload struct {
	Topic   string
	Payload string
}

type Options struct {
	KeyPath                  string
	CertPath                 string
	CaPath                   string
	ClientId                 string
	Region                   string
	BaseReconnectTime        time.Duration
	MaximumReconnectTime     time.Duration
	MinimumConnectionTime    time.Duration
	Keepalive                time.Duration
	Protocol                 string
	Host                     string
	Debug                    bool
	Qos                      byte
	Retain                   bool
	Will                     *TopicPayload
	OfflineQueueing          bool
	OfflineQueueMaxSize      int
	OfflineQueueDropBehavior pubqueue.QueueDropBehavior
	AutoResubscribe          bool
}
