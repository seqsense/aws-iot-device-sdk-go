// Copyright 2018 SEQSENSE, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package subqueue is internal implementation of awsiotdev package.
// It implements offline queueing of MQTT subscription.
package subqueue

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// OperationType represents type of the operation.
type OperationType int32

const (
	// Subscribe requests a new subscription.
	Subscribe OperationType = iota
	// Unsubscribe requests stopping the subscription.
	Unsubscribe
)

// Subscription stores a request of subscription operation.
type Subscription struct {
	Type  OperationType
	Topic string
	Cb    mqtt.MessageHandler
}

// Queue manages queued subscription operations.
type Queue struct {
	queue []*Subscription
}

// New returns a queue of subscription operations.
func New() *Queue {
	return &Queue{}
}

// Enqueue pushes a subscription operation to the queue.
func (s *Queue) Enqueue(d *Subscription) {
	s.queue = append(s.queue, d)
}

// Pop gets the oldest subscription operation in the queue and drops it.
func (s *Queue) Pop() *Subscription {
	if len(s.queue) == 0 {
		return nil
	}
	d := s.queue[0]
	s.queue = s.queue[1:]

	return d
}

// Len returns the number of messages in the queue.
func (s *Queue) Len() int {
	return len(s.queue)
}
