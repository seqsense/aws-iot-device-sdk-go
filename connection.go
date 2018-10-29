package devicecli

import (
	"log"
	"time"

	"github.com/seqsense/aws-iot-device-sdk-go/pubqueue"
	"github.com/seqsense/aws-iot-device-sdk-go/subqueue"
)

func connectionHandler(c *DeviceClient) {
	state := inactive
	pubQueue := pubqueue.New(c.opt.OfflineQueueMaxSize, c.opt.OfflineQueueDropBehavior)
	subQueue := subqueue.New()
	activeSubs := make(map[string]*subqueue.Subscription)

	for {
		statePrev := state

		select {
		case d := <-c.publishCh:
			if state.isActive() {
				c.publishOrEnqueue(d, pubQueue)
			} else {
				if c.opt.OfflineQueueing {
					pubQueue.Enqueue(d)
				}
			}

		case d := <-c.subscribeCh:
			if state.isActive() {
				switch d.Type {
				case subqueue.Subscribe:
					c.subscribeOrEnqueue(d, subQueue, activeSubs)
				case subqueue.Unsubscribe:
					c.unsubscribeOrEnqueue(d, subQueue, activeSubs)
				}
			} else {
				if c.opt.OfflineQueueing {
					subQueue.Enqueue(d)
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
				c.resubscribe(subQueue, activeSubs)

			case established:
				log.Print("Processing queued operations\n")
				go func() {
					time.Sleep(c.opt.MinimumConnectionTime)
					c.stableTimerCh <- true
				}()
				c.processQueuedOps(pubQueue, subQueue, activeSubs)

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

func (c *DeviceClient) publishOrEnqueue(d *pubqueue.Data, pq *pubqueue.Queue) {
	token := c.cli.Publish(d.Topic, c.opt.Qos, c.opt.Retain, d.Payload)
	go func() {
		token.Wait()
		if token.Error() != nil {
			log.Printf("Failed to publish (%s)\n", token.Error())
			// MQTT doesn't guarantee receive order; just append to the last
			pq.Enqueue(d)
		}
	}()
}
func (c *DeviceClient) subscribeOrEnqueue(d *subqueue.Subscription, sq *subqueue.Queue, as map[string]*subqueue.Subscription) {
	token := c.cli.Subscribe(d.Topic, c.opt.Qos, d.Cb)
	go func() {
		token.Wait()
		if token.Error() != nil {
			log.Printf("Failed to subscribe (%s)\n", token.Error())
			sq.Enqueue(d)
		} else {
			as[d.Topic] = d
		}
	}()
}
func (c *DeviceClient) unsubscribeOrEnqueue(d *subqueue.Subscription, sq *subqueue.Queue, as map[string]*subqueue.Subscription) {
	token := c.cli.Unsubscribe(d.Topic)
	go func() {
		token.Wait()
		if token.Error() != nil {
			log.Printf("Failed to unsubscribe (%s)\n", token.Error())
			sq.Enqueue(d)
		} else {
			delete(as, d.Topic)
		}
	}()
}
func (c *DeviceClient) processQueuedOps(pq *pubqueue.Queue, sq *subqueue.Queue, as map[string]*subqueue.Subscription) {
	for pq.Len() > 0 {
		d := pq.Pop()
		c.publishOrEnqueue(d, pq)
	}
	for sq.Len() > 0 {
		d := sq.Pop()

		switch d.Type {
		case subqueue.Subscribe:
			c.subscribeOrEnqueue(d, sq, as)
		case subqueue.Unsubscribe:
			c.unsubscribeOrEnqueue(d, sq, as)
		}
	}
}
func (c *DeviceClient) resubscribe(sq *subqueue.Queue, as map[string]*subqueue.Subscription) {
	for _, d := range as {
		delete(as, d.Topic)

		token := c.cli.Subscribe(d.Topic, c.opt.Qos, d.Cb)
		go func() {
			token.Wait()
			if token.Error() != nil {
				log.Printf("Failed to subscribe (%s)\n", token.Error())
				sq.Enqueue(d)
			} else {
				as[d.Topic] = d
			}
		}()
	}
}
