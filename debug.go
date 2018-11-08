package awsiotdev

import (
	"log"
)

var (
	// Backend function of debug print output. This can be replaced by custom logger.
	DebugPrintBackend func(...interface{}) = log.Print
	// Backend function of debug printf output. This can be replaced by custom logger.
	DebugPrintfBackend func(string, ...interface{}) = log.Printf
	// Backend function of debug println output. This can be replaced by custom logger.
	DebugPrintlnBackend func(...interface{}) = log.Println
)

type debugOut struct {
	enable bool
}

func (s *debugOut) print(a ...interface{}) {
	if s.enable {
		DebugPrintBackend(a...)
	}
}
func (s *debugOut) printf(format string, a ...interface{}) {
	if s.enable {
		DebugPrintfBackend(format, a...)
	}
}
func (s *debugOut) println(a ...interface{}) {
	if s.enable {
		DebugPrintlnBackend(a...)
	}
}
