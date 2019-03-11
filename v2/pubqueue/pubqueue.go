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

// Package pubqueue is internal implementation of awsiotdev package.
// It implements offline queueing of published messages.
package pubqueue

// Data stores topic name string and payload data of a message.
type Data struct {
	Topic   string
	Payload interface{}
}

// Queue manages queued messages.
type Queue struct {
	buf          []*Data
	maxSize      int
	dropBehavior DropBehavior
}

// New returns a message queue with specified maximum size and drop behavior setting.
func New(maxSize int, dropBehavior DropBehavior) *Queue {
	return &Queue{
		maxSize:      maxSize,
		dropBehavior: dropBehavior,
	}
}

// Enqueue pushes a message to the queue.
func (s *Queue) Enqueue(d *Data) {
	if s.maxSize > 0 && len(s.buf) >= s.maxSize {
		switch s.dropBehavior {
		case Newest:
			s.buf = s.buf[:len(s.buf)-1]
		case Oldest:
			s.buf = s.buf[1:]
		}
	}
	s.buf = append(s.buf, d)
}

// Pop gets the oldest message in the queue and drops it.
func (s *Queue) Pop() *Data {
	if len(s.buf) == 0 {
		return nil
	}
	d := s.buf[0]
	s.buf = s.buf[1:]

	return d
}

// Len returns the number of messages in the queue.
func (s *Queue) Len() int {
	return len(s.buf)
}
