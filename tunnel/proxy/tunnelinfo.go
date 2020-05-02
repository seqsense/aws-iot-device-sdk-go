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
}
