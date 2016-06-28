package util

import (
	"fmt"
	"os"
)

var (
	response = string("")
)

// Leland Batey's log to os.Stderr
func Logf(format string, args ...interface{}) {
	response += fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, format, args...)
}
