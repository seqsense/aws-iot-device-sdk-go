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
