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

package awsiotdev

import (
	"sync"

	"github.com/seqsense/aws-iot-device-sdk-go/v2/subqueue"
)

type pubSubQueues struct {
	cli            *DeviceClient
	subQueue       *subqueue.Queue
	activeSubs     map[string]*subqueue.Subscription
	activeSubsLock sync.RWMutex
}

func connectionHandler(c *DeviceClient) {
	state := inactive
	psq := &pubSubQueues{
		c,
		subqueue.New(),
		make(map[string]*subqueue.Subscription),
		sync.RWMutex{},
	}

	for {
		select {
		case d := <-c.subscribeCh:
			if state.isActive() {
				switch d.Type {
				case subqueue.Subscribe:
					psq.subscribeOrEnqueue(d)
				case subqueue.Unsubscribe:
					psq.unsubscribeOrEnqueue(d)
				}
			} else {
				if c.opt.OfflineQueueing {
					psq.subQueue.Enqueue(d)
				}
			}

		case state = <-c.stateUpdateCh:
			c.dbg.printf("State updated to %s\n", state.String())

			switch state {
			case inactive:
				// do nothing
			case established:
				c.dbg.print("Processing queued operations\n")
				psq.processQueuedOps()
			case terminating:
				c.dbg.print("Terminating connection\n")
				c.cli.Disconnect(250)
				return
			default:
				panic("Invalid internal state\n")
			}
		}
	}
}

func (s *pubSubQueues) subscribeOrEnqueue(d *subqueue.Subscription) {
	token := s.cli.cli.Subscribe(d.Topic, s.cli.opt.Qos, d.Cb)
	go func() {
		token.Wait()
		if token.Error() != nil {
			s.cli.dbg.printf("Failed to subscribe (%s)\n", token.Error())
			s.cli.subscribeCh <- d
		}
	}()
}
func (s *pubSubQueues) unsubscribeOrEnqueue(d *subqueue.Subscription) {
	token := s.cli.cli.Unsubscribe(d.Topic)
	go func() {
		token.Wait()
		if token.Error() != nil {
			s.cli.dbg.printf("Failed to unsubscribe (%s)\n", token.Error())
			s.cli.subscribeCh <- d
		}
	}()
}

func (s *pubSubQueues) processQueuedOps() {
	for s.subQueue.Len() > 0 {
		d := s.subQueue.Pop()

		switch d.Type {
		case subqueue.Subscribe:
			s.subscribeOrEnqueue(d)
		case subqueue.Unsubscribe:
			s.unsubscribeOrEnqueue(d)
		}
	}
}
