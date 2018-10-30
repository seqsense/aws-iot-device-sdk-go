package subqueue

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type OperationType int32

const (
	Subscribe OperationType = iota
	Unsubscribe
)

type Subscription struct {
	Type  OperationType
	Topic string
	Cb    mqtt.MessageHandler
}

type Queue struct {
	queue []*Subscription
}

func New() *Queue {
	return &Queue{}
}

func (s *Queue) Enqueue(d *Subscription) {
	s.queue = append(s.queue, d)
}

func (s *Queue) Pop() *Subscription {
	if len(s.queue) == 0 {
		return nil
	}
	d := s.queue[0]
	s.queue = s.queue[1:]

	return d
}

func (s *Queue) Len() int {
	return len(s.queue)
}
