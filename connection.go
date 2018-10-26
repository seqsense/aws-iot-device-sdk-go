package devicecli

import (
	"log"
	"time"
)

func connectionStateHandler(c *DeviceClient) {
	state := inactive
	for {
		statePrev := state

		select {
		case <-c.stableTimer:
			log.Print("Stable timer reached\n")
			c.stateUpdater <- stable

		case state = <-c.stateUpdater:
			log.Printf("State updated to %s\n", state.String())

			switch state {
			case inactive:
				c.stableTimer = make(chan bool)
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
					c.stableTimer <- true
				}()

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
