package handlers

import (
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/gochipon/truss/gengokit"
	"github.com/gochipon/truss/gengokit/httptransport"
	"github.com/gochipon/truss/svcdef"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestHooksAddingImport(t *testing.T) {
	const def = `
		syntax = "proto3";
		package echo;

		service _Foo_Bar {
		  rpc Echo (EchoRequest) returns (EchoResponse) {}
		}
		message EchoRequest {
		  string In = 1;
		}
		message EchoResponse {
		  string Out = 1;
		}
	`

	const prev = `
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

	sd, err := svcdef.NewFromString(def, gopath)
	require.NoError(t, err)

	conf := gengokit.Config{
		GoPackage: "github.com/gochipon/truss/gengokit",
		PBPackage: "github.com/gochipon/truss/gengokit/echo-service",
	}

	te, err := gengokit.NewData(sd, conf)
	require.NoError(t, err)
	newHooksf, err := renderHooksFile(prev, te)
	require.NoError(t, err)

	c1 := httptransport.FormatCode(prev)
	c2 := httptransport.FormatCode(newHooksf)

	require.Greater(t, len(c2), len(c1), "new code should be longer than the previous go code")
	require.Contains(t, c2, "svc")
	require.Contains(t, c2, "SetConfig")
	require.Contains(t, c2, "InterruptHandler")
	require.NotContains(t, c2, "server")

}

// renderHooksFile takes in a previous file as a string and returns the
// generated handlers/hooks.go file as a string. This helper method exists
// because the logic for reading the io.Reader to a string is repeated.
func renderHooksFile(prev string, data *gengokit.Data) (string, error) {
	var prevFile io.Reader
	if prev != "" {
		prevFile = strings.NewReader(prev)
	}

	h := NewHook(prevFile)

	next, err := h.Render(ServerHandlerPath, data)
	if err != nil {
		return "", err
	}

	nextBytes, err := ioutil.ReadAll(next)
	if err != nil {
		return "", err
	}

	nextCode, err := testFormat(string(nextBytes))
	if err != nil {
		return "", errors.Wrap(err, "cannot format")
	}

	nextCode = strings.TrimSpace(nextCode)

	return nextCode, nil
}
