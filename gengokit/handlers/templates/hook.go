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
	&HookDef{
		Name: "UnmarshalArg",
		Imports: []string{"bytes", "encoding/json", "github.com/pkg/errors"},
		Code: HookUnmarsh,
	},
}

const HookHead = `
package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"
)

var (
	_ = bytes.Compare
	_ = json.Compact
	_ = errors.Wrapf
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

const HookUnmarsh = `

func UnmarshalArg(dest interface{}, data, what string) {
	err := json.Unmarshal([]byte(data), dest)
	if err != nil {
		panic(errors.Wrapf(err, "unmarshalling %s from %q:", what, data))
	}
}
`
