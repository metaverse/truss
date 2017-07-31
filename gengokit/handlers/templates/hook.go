package templates

const bbb2306Hook = `
package handlers

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func InterruptHandler(errc chan<- error) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	terminateError := fmt.Errorf("%s", <-c)

	// Place whatever shutdown handling you want here

	errc <- terminateError
}
`

var Hook = map[string]map[string]string{
	"bbb2306": {
		"Hook": bbb2306Hook,
	},
	"v0.5.0": {
		"Hook": bbb2306Hook,
	},
}
