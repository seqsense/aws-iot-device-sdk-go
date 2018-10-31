package pubqueue

import (
	"errors"
	"strings"
)

type DropBehavior int

const (
	Oldest DropBehavior = iota
	Newest
	Unknown
)

func (x DropBehavior) String() string {
	switch x {
	case Oldest:
		return "oldest"
	case Newest:
		return "newest"
	default:
		return "unknown"
	}
}

func NewDropBehavior(b string) (DropBehavior, error) {
	switch strings.ToLower(b) {
	case "oldest":
		return Oldest, nil
	case "newest":
		return Newest, nil
	default:
		return Unknown, errors.New("Unknown drop behavior")
	}
}
