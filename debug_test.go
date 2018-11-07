package awsiotdev

import (
	"errors"
	"fmt"
	"log"
)

func Example_debugPrint() {
	DebugPrintBackend = func(a ...interface{}) {
		fmt.Print(a...)
	}

	s := &DeviceClient{opt: &Options{Debug: true}}
	s.debugPrint("Test1:", true)
	s = &DeviceClient{opt: &Options{Debug: false}}
	s.debugPrint("Test2:", false)

	DebugPrintBackend = log.Print

	// Output:
	// Test1:true
}

func Example_debugPrintf() {
	DebugPrintfBackend = func(format string, a ...interface{}) {
		fmt.Printf(format, a...)
	}

	s := &DeviceClient{opt: &Options{Debug: true}}
	s.debugPrintf("Formatted value 1 (%d)", 10)
	s = &DeviceClient{opt: &Options{Debug: false}}
	s.debugPrintf("Formatted value 2 (%d)", 11)

	DebugPrintfBackend = log.Printf

	// Output:
	// Formatted value 1 (10)
}

func Example_debugPrintln() {
	DebugPrintlnBackend = func(a ...interface{}) {
		fmt.Println(a...)
	}

	s := &DeviceClient{opt: &Options{Debug: true}}
	s.debugPrintln(errors.New("Error string 1"))
	s = &DeviceClient{opt: &Options{Debug: false}}
	s.debugPrintln(errors.New("Error string 2"))

	DebugPrintlnBackend = log.Println

	// Output:
	// Error string 1
}
