package tunnel

// ClientMode is a mode of the client.
type ClientMode string

func (m ClientMode) String() string {
	return string(m)
}

// List of ClientModes.
const (
	Source      ClientMode = "source"
	Destination ClientMode = "destination"
)

type notification struct {
	ClientAccessToken string     `json:"clientAccessToken"`
	ClientMode        ClientMode `json:"clientMode"`
	Region            string     `json:"region"`
	Services          []string   `json:"services"`
}
