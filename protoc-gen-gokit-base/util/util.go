package util

import (
	"fmt"
	"os"
)

var (
	response = string("")
)

func Log(args ...interface{}) {
	fmt.Fprintln(os.Stderr, args...)
}

// Leland Batey's log to os.Stderr
func Logf(format string, args ...interface{}) {
	response += fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, format, args...)
}
