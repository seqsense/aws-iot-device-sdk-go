package devicecli

type deviceState int

const (
	inactive deviceState = iota
	established
	stable
	terminating
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
	default:
		return "unknown"
	}
}
