package awsiotdev

import (
	"log"
	"time"

	"github.com/seqsense/aws-iot-device-sdk-go/pubqueue"
	"github.com/seqsense/aws-iot-device-sdk-go/subqueue"
)

type pubSubQueues struct {
	cli        *DeviceClient
	pubQueue   *pubqueue.Queue
	subQueue   *subqueue.Queue
	activeSubs map[string]*subqueue.Subscription
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
			log.Print("Stable timer reached\n")
			c.stateUpdateCh <- stable

		case state = <-c.stateUpdateCh:
			log.Printf("State updated to %s\n", state.String())

			switch state {
			case inactive:
				c.stableTimerCh = make(chan bool)
				c.reconnectPeriod = c.reconnectPeriod * 2
				if c.reconnectPeriod > c.opt.MaximumReconnectTime {
					c.reconnectPeriod = c.opt.MaximumReconnectTime
				}
				log.Printf("Trying to reconnect (%d ms)\n", c.reconnectPeriod/time.Millisecond)

				go func() {
					time.Sleep(c.reconnectPeriod)
					c.stateUpdateCh <- reconnecting
				}()

			case reconnecting:
				c.connect()
				psq.resubscribe()

			case established:
				log.Print("Processing queued operations\n")
				go func() {
					time.Sleep(c.opt.MinimumConnectionTime)
					c.stableTimerCh <- true
				}()
				psq.processQueuedOps()

			case stable:
				if statePrev == established {
					log.Print("Connection is stable\n")
					c.reconnectPeriod = c.opt.BaseReconnectTime
				}

			case terminating:
				log.Print("Terminating connection\n")
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
			log.Printf("Failed to publish (%s)\n", token.Error())
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
			log.Printf("Failed to subscribe (%s)\n", token.Error())
			s.cli.subscribeCh <- d
		} else {
			s.activeSubs[d.Topic] = d
		}
	}()
}
func (s *pubSubQueues) unsubscribeOrEnqueue(d *subqueue.Subscription) {
	token := s.cli.cli.Unsubscribe(d.Topic)
	go func() {
		token.Wait()
		if token.Error() != nil {
			log.Printf("Failed to unsubscribe (%s)\n", token.Error())
			s.cli.subscribeCh <- d
		} else {
			delete(s.activeSubs, d.Topic)
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
	for _, d := range s.activeSubs {
		delete(s.activeSubs, d.Topic)

		token := s.cli.cli.Subscribe(d.Topic, s.cli.opt.Qos, d.Cb)
		go func(d *subqueue.Subscription) {
			token.Wait()
			if token.Error() != nil {
				log.Printf("Failed to subscribe (%s)\n", token.Error())
				s.cli.subscribeCh <- d
			} else {
				s.activeSubs[d.Topic] = d
			}
		}(d)
	}
}
