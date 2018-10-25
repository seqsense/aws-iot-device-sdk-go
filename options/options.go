package options

type Options struct {
	KeyPath             string
	CertPath            string
	CaPath              string
	ClientId            string
	Region              string
	BaseReconnectTimeMs int
	Keepalive           int
	Protocol            string
	Host                string
	Debug               bool
	Will                struct {
		Topic   string
		Payload string
	}
	Qos    int
	Retain bool
}
