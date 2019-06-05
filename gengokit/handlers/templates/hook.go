package templates

type HookDef struct {
	Name    string
	Imports []string
	Code    string
}

var Hooks []*HookDef = []*HookDef{
	&HookDef{
		Name: "InterruptHandler",
		Imports: []string{"fmt", "os", "os/signal", "syscall"},
		Code: HookInt,
	},
	&HookDef{
		Name: "Report",
		Imports: []string{"encoding/json", "fmt", "os"},
		Code: HookReport,
	},
}

const HookHead = `
package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)
`

const HookInt = `

func InterruptHandler(errc chan<- error) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	terminateError := fmt.Errorf("%s", <-c)

	// Place whatever shutdown handling you want here

	errc <- terminateError
}
`

const HookReport = `

func Report(response, request interface{}, method string) {
	/* Closer to original client output:
	fmt.Println("Client Requested with:")
	fmt.Println(request)
	fmt.Println("Server Responded with:")
	fmt.Println(response)
	*/

	// Output response in JSON:
	buf, err := json.MarshalIndent(response, "", "  ")
	if nil != err {
		fmt.Fprintf(os.Stderr, "Failed to convert response to JSON: %v\n%s\n",
			err, buf)
	} else {
		fmt.Printf("%s\n", buf)
	}
}
`
