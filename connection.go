package devicecli

import (
	"log"
	"time"
)

func connectionHandler(c *DeviceClient) {
	state := inactive
	for {
		statePrev := state

		select {
		case d := <-c.publishCh:
			if state.isActive() {
				token := c.cli.Publish(d.Topic, c.opt.Qos, c.opt.Retain, d.Payload)
				go func() {
					token.Wait()
					if token.Error() != nil {
						log.Printf("Failed to publish (%s)\n", token.Error())
						if c.opt.OfflineQueueing {
							c.pubQueue.Enqueue(d)
						}
					}
				}()
			} else {
				if c.opt.OfflineQueueing {
					c.pubQueue.Enqueue(d)
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

				time.Sleep(c.reconnectPeriod)
				c.connect()

			case established:
				log.Print("Processing queued operations\n")
				go func() {
					time.Sleep(c.opt.MinimumConnectionTime)
					c.stableTimerCh <- true
				}()
				for c.pubQueue.Len() > 0 {
					d := c.pubQueue.Pop()

					token := c.cli.Publish(d.Topic, c.opt.Qos, c.opt.Retain, d.Payload)
					go func() {
						token.Wait()
						if token.Error() != nil {
							log.Printf("Failed to publish (%s)\n", token.Error())
							// MQTT doesn't guarantee receive order; just append to the last
							c.pubQueue.Enqueue(d)
						}
					}()
				}

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
