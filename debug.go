package awsiotdev

import (
	"log"
)

var (
	// Backend function of debug print output. This can be replaced by custom logger.
	DebugPrintBackend func(...interface{})
	// Backend function of debug printf output. This can be replaced by custom logger.
	DebugPrintfBackend func(string, ...interface{})
	// Backend function of debug println output. This can be replaced by custom logger.
	DebugPrintlnBackend func(...interface{})
)

func init() {
	DebugPrintBackend = log.Print
	DebugPrintfBackend = log.Printf
	DebugPrintlnBackend = log.Println
}

func (s *DeviceClient) debugPrint(a ...interface{}) {
	if s.opt.Debug {
		DebugPrintBackend(a...)
	}
}
func (s *DeviceClient) debugPrintf(format string, a ...interface{}) {
	if s.opt.Debug {
		DebugPrintfBackend(format, a...)
	}
}
func (s *DeviceClient) debugPrintln(a ...interface{}) {
	if s.opt.Debug {
		DebugPrintlnBackend(a...)
	}
}
