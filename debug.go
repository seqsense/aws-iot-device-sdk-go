package awsiotdev

import (
	"log"
)

var (
	// DebugPrintBackend stores backend function of debug print output.
	// This can be replaced by custom logger.
	DebugPrintBackend func(...interface{}) = log.Print
	// DebugPrintfBackend stores backend function of debug printf output.
	// This can be replaced by custom logger.
	DebugPrintfBackend func(string, ...interface{}) = log.Printf
	// DebugPrintlnBackend stores backend function of debug println output.
	// This can be replaced by custom logger.
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
