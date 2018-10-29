package devicecli

type deviceState int

const (
	inactive deviceState = iota
	established
	stable
	terminating
	reconnecting
)

func (s deviceState) String() string {
	switch s {
	case inactive:
		return "inactive"
	case established:
		return "established"
	case stable:
		return "stable"
	case terminating:
		return "terminating"
	case reconnecting:
		return "reconnecting"
	default:
		return "unknown"
	}
}

func (s deviceState) isActive() bool {
	return s == established || s == stable
}
