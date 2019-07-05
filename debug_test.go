package awsiotdev

import (
	"errors"
	"fmt"
	"log"
)

func Example_debugOut_print() {
	DebugPrintBackend = func(a ...interface{}) {
		fmt.Print(a...)
	}

	s := &debugOut{true}
	s.print("Test1:", true)
	s = &debugOut{false}
	s.print("Test2:", false)

	DebugPrintBackend = log.Print

	// Output:
	// Test1:true
}

func Example_debugOut_printf() {
	DebugPrintfBackend = func(format string, a ...interface{}) {
		fmt.Printf(format, a...)
	}

	s := &debugOut{true}
	s.printf("Formatted value 1 (%d)", 10)
	s = &debugOut{false}
	s.printf("Formatted value 2 (%d)", 11)

	DebugPrintfBackend = log.Printf

	// Output:
	// Formatted value 1 (10)
}

func Example_debugOut_println() {
	DebugPrintlnBackend = func(a ...interface{}) {
		fmt.Println(a...)
	}

	s := &debugOut{true}
	s.println(errors.New("Error string 1"))
	s = &debugOut{false}
	s.println(errors.New("Error string 2"))

	DebugPrintlnBackend = log.Println

	// Output:
	// Error string 1
}
