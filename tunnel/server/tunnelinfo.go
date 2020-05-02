package proxy

import (
	"time"
)

type tunnelInfo struct {
	thingName       string
	services        []string
	destAccessToken string
	srcAccessToken  string
	expireAt        time.Time
	chClosed        chan struct{}
	chDestSrc       chan []byte
	chSrcDest       chan []byte
}
