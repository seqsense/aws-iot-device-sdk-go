package server

type tunnelInfo struct {
	thingName       string
	services        []string
	destAccessToken string
	srcAccessToken  string
	chDone          <-chan struct{}
	cancel          func()
	chDestSrc       chan []byte
	chSrcDest       chan []byte
}
