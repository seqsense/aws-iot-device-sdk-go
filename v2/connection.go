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
	"time"

	"github.com/seqsense/aws-iot-device-sdk-go/v2/pubqueue"
	"github.com/seqsense/aws-iot-device-sdk-go/v2/subqueue"
)

type pubSubQueues struct {
	cli            *DeviceClient
	pubQueue       *pubqueue.Queue
	subQueue       *subqueue.Queue
	activeSubs     map[string]*subqueue.Subscription
	activeSubsLock sync.RWMutex
}

func connectionHandler(c *DeviceClient) {
	state := inactive
	drop, err := pubqueue.NewDropBehavior(c.opt.OfflineQueueDropBehavior)
	if err != nil {
		panic(err)
	}
	psq := &pubSubQueues{
		c,
		pubqueue.New(c.opt.OfflineQueueMaxSize, drop),
		subqueue.New(),
		make(map[string]*subqueue.Subscription),
		sync.RWMutex{},
	}

	for {
		statePrev := state

		select {
		case d := <-c.publishCh:
			if state.isActive() {
				psq.publishOrEnqueue(d)
			} else {
				if c.opt.OfflineQueueing {
					psq.pubQueue.Enqueue(d)
				}
			}

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

		case <-c.stableTimerCh:
			c.dbg.print("Stable timer reached\n")
			c.stateUpdateCh <- stable

		case state = <-c.stateUpdateCh:
			c.dbg.printf("State updated to %s\n", state.String())

			switch state {
			case inactive:
				c.stableTimerCh = make(chan bool)
				c.reconnectPeriod = c.reconnectPeriod * 2
				if c.reconnectPeriod > c.opt.MaximumReconnectTime {
					c.reconnectPeriod = c.opt.MaximumReconnectTime
				}
				c.dbg.printf("Trying to reconnect (%d ms)\n", c.reconnectPeriod/time.Millisecond)
				go func() {
					time.Sleep(c.reconnectPeriod)
					c.stateUpdateCh <- reconnecting
				}()

			case reconnecting:
				c.connect()
				psq.resubscribe()

			case established:
				c.dbg.print("Processing queued operations\n")
				go func() {
					time.Sleep(c.opt.MinimumConnectionTime)
					c.stableTimerCh <- true
				}()
				psq.processQueuedOps()

			case stable:
				if statePrev == established {
					c.dbg.print("Connection is stable\n")
					c.reconnectPeriod = c.opt.BaseReconnectTime
				}

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

func (s *pubSubQueues) publishOrEnqueue(d *pubqueue.Data) {
	token := s.cli.cli.Publish(d.Topic, s.cli.opt.Qos, s.cli.opt.Retain, d.Payload)
	go func() {
		token.Wait()
		if token.Error() != nil {
			s.cli.dbg.printf("Failed to publish (%s)\n", token.Error())
			// MQTT doesn't guarantee receive order; just append to the last
			s.cli.publishCh <- d
		}
	}()
}
func (s *pubSubQueues) subscribeOrEnqueue(d *subqueue.Subscription) {
	token := s.cli.cli.Subscribe(d.Topic, s.cli.opt.Qos, d.Cb)
	go func() {
		token.Wait()
		if token.Error() != nil {
			s.cli.dbg.printf("Failed to subscribe (%s)\n", token.Error())
			s.cli.subscribeCh <- d
		} else {
			s.setSubscription(d)
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
		} else {
			s.clearSubscription(d)
		}
	}()
}

func (s *pubSubQueues) processQueuedOps() {
	for s.pubQueue.Len() > 0 {
		d := s.pubQueue.Pop()
		s.publishOrEnqueue(d)
	}
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
func (s *pubSubQueues) resubscribe() {
	if !s.cli.opt.AutoResubscribe {
		return
	}
	for _, d := range s.getActiveSubs() {
		s.clearSubscription(d)

		token := s.cli.cli.Subscribe(d.Topic, s.cli.opt.Qos, d.Cb)
		go func(d *subqueue.Subscription) {
			token.Wait()
			if token.Error() != nil {
				s.cli.dbg.printf("Failed to subscribe (%s)\n", token.Error())
				s.cli.subscribeCh <- d
			} else {
				s.setSubscription(d)
			}
		}(d)
	}
}

func (s *pubSubQueues) getActiveSubs() []*subqueue.Subscription {
	s.activeSubsLock.RLock()
	defer s.activeSubsLock.RUnlock()

	subs := make([]*subqueue.Subscription, 0, len(s.activeSubs))
	for _, d := range s.activeSubs {
		subs = append(subs, d)
	}
	return subs
}

func (s *pubSubQueues) setSubscription(d *subqueue.Subscription) {
	s.activeSubsLock.Lock()
	defer s.activeSubsLock.Unlock()

	s.activeSubs[d.Topic] = d
}

func (s *pubSubQueues) clearSubscription(d *subqueue.Subscription) {
	s.activeSubsLock.Lock()
	defer s.activeSubsLock.Unlock()

	delete(s.activeSubs, d.Topic)
}
