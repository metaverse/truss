// +build integration

package integration

import (
	"bytes"
	"flag"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"testing"

	"github.com/pkg/errors"
)

func init() {
	clean := flag.Bool("clean", false, "Remove all generated test files and do nothing else")
	flag.Parse()
	if *clean {
		wd, _ := os.Getwd()
		servicesDir := wd + "/test_service_definitions"
		cleanTests(servicesDir)
		os.Exit(0)
	}
}

// runReference stores data about a client-server interaction
// This data is then used to display output
type runReference struct {
	name         string
	clientErr    bool
	serverErr    bool
	clientOutput string
	serverOutput string
}

func TestTruss(t *testing.T) {
	wd, _ := os.Getwd()
	servicesDir := wd + "/test_service_definitions"

	runRefs := make(chan runReference)
	runCount := 0

	dirs, _ := ioutil.ReadDir(servicesDir)
	for i, d := range dirs {
		// If this item is not a directory skip it
		if !d.IsDir() {
			continue
		}

		// tests will be run on the fullpath to service directoru
		sDir := servicesDir + "/" + d.Name()

		// Clean up the service directories in each test
		removeTestFiles(sDir)

		// On port relative to 8082
		port := 8082 + i

		// Running the integration tests one at a time because truss running on all files at once
		// seems to slow the system more than distribute the work
		t.Logf("Running integration test for %v...", d.Name())
		out, err := truss(sDir)

		// If truss fails, test error and skip communication
		if err != nil {
			t.Errorf("Truss generation FAILED - %v\nTruss Output:\n%v", d.Name(), out)
			continue
		}

		// Build the service to be tested
		err = buildTestService(sDir)
		if err != nil {
			t.Errorf("Could not buld service. Error:%v", err)
		}

		// Run them save a reference to each run
		go runServerAndClient(sDir, port, port+1000, runRefs)
		runCount++
	}

	// Check all the runs, the test failed if there were any errors
	for i := 0; i < runCount; i++ {
		ref := <-runRefs
		if ref.clientErr || ref.serverErr {
			t.Errorf("Communication test FAILED - %v", ref.name)
			t.Logf("Client Output\n%v", ref.clientOutput)
			t.Logf("Server Output\n%v", ref.serverOutput)
		}
	}

	// If nothing failed, delete the generated files
	if !t.Failed() {
		cleanTests(servicesDir)
	}
}

// truss calls truss on *.proto in path
// Truss logs to Stdout when generation passes or fails
func truss(path string) (string, error) {
	var protofiles []string
	files, err := ioutil.ReadDir(path)
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		if strings.HasSuffix(f.Name(), ".proto") {
			protofiles = append(protofiles, f.Name())
		}
	}

	trussExec := exec.Command(
		"truss",
		protofiles...,
	)
	trussExec.Dir = path

	out, err := trussExec.CombinedOutput()

	return string(out), err
}

// buildTestService builds a truss service with the package TEST
// into the `serviceDir`/bin directory
func buildTestService(serviceDir string) (err error) {

	goPath := os.Getenv("GOPATH")
	goImportPath := strings.TrimPrefix(serviceDir, goPath+"/src/")

	binDir := serviceDir + "/bin"

	err = mkdir(binDir)
	if err != nil {
		return err
	}

	const serverPath = "/TEST-service/TEST-server/..."
	const clientPath = "/TEST-service/TEST-client/..."

	// Build server and client
	errChan := make(chan error)

	go goBuild("TEST-server", binDir, goImportPath+serverPath, errChan)
	go goBuild("TEST-client", binDir, goImportPath+clientPath, errChan)

	err = <-errChan
	if err != nil {
		return err
	}

	err = <-errChan
	if err != nil {
		return err
	}

	return nil
}

// goBuild calls the `$ go get ` to install dependenices
// and then calls `$ go build ` to build the service
func goBuild(name, outputPath, codePath string, errChan chan error) {

	// $ go get

	goGetExec := exec.Command(
		"go",
		"get",
		"-d",
		"-v",
		codePath,
	)

	goGetExec.Stderr = os.Stderr

	err := goGetExec.Run()

	if err != nil {
		errChan <- errors.Wrapf(err, "could not $ go get %v", codePath)
		return
	}

	// $ go build

	goBuildExec := exec.Command(
		"go",
		"build",
		"-o",
		outputPath+"/"+name,
		codePath,
	)

	goBuildExec.Stderr = os.Stderr

	err = goBuildExec.Run()
	if err != nil {
		errChan <- errors.Wrapf(err, "could not $ go build %v", codePath)
		return
	}

	errChan <- nil
}

// mkdir acts like $ mkdir -p path
func mkdir(path string) error {
	dir := filepath.Dir(path)

	// 0775 is the file mode that $ mkdir uses when creating a directoru
	err := os.MkdirAll(dir, 0775)

	return err
}

// runServerAndClient execs a TEST-server and TEST-client and puts a
// runReference to their interaction on the runRefs channel
func runServerAndClient(path string, port int, debugPort int, runRefs chan runReference) {
	// From within a folder with a truss `service`
	// These are the paths to the compiled binaries
	const relativeServerPath = "/bin/TEST-server"

	// Output buffer for the server Stdout and Stderr
	serverOut := bytes.NewBuffer(nil)
	// Get the server command ready with the port
	server := exec.Command(
		path+relativeServerPath,
		"-grpc.addr",
		":"+strconv.Itoa(port),
		"-http.addr",
		":"+strconv.Itoa(port-70),
		"-debug.addr",
		":"+strconv.Itoa(debugPort),
	)

	// Put serverOut to be the writer of data from Stdout and Stderr
	server.Stdout = serverOut
	server.Stderr = serverOut

	// Start the server
	serverErrChan := make(chan error)
	go func() {
		err := server.Run()
		serverErrChan <- err
		defer server.Process.Kill()
	}()

	// We may need to wait a few miliseconds for the server to startup
	retryTime := time.Millisecond * 100
	t := time.NewTimer(retryTime)
	for server.Process == nil {
		<-t.C
		t.Reset(retryTime)
	}
	<-t.C

	cOut, cErr := runClient(path, port)

	var sErr bool

	// If the server ever stopped then it errored
	// If it did not stop, kill it and see if that errors
	select {
	case <-serverErrChan:
		sErr = true
	default:
		if server.Process == nil {
			// This likely means the server never started
			sErr = true
		} else {
			// If the Process is not nil, kill it, clean up our mess
			err := server.Process.Kill()
			if err != nil {
				sErr = true
			} else {
				sErr = false
			}
		}
	}

	// Construct a reference to what happened here
	ref := runReference{
		name:         filepath.Base(path),
		clientErr:    cErr,
		serverErr:    sErr,
		clientOutput: string(cOut),
		serverOutput: serverOut.String(),
	}

	runRefs <- ref
}

func runClient(path string, port int) ([]byte, bool) {
	const relativeClientPath = "/bin/TEST-client"

	client := exec.Command(
		path+relativeClientPath,
		"-grpc.addr",
		":"+strconv.Itoa(port),
	)

	cOut, err := client.CombinedOutput()

	var cErr bool
	if err != nil {
		cErr = true
	} else {
		cErr = false
	}

	return cOut, cErr
}

// fileExists checks if a file at the given path exists. Returns true if the
// file exists, and false if the file does not exist.
func fileExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

// cleanTests removes all test files from all directories in servicesDir
func cleanTests(servicesDir string) {
	// Clean up the service directories in each test
	dirs, _ := ioutil.ReadDir(servicesDir)
	for _, d := range dirs {
		// If this item is not a directory skip it
		if !d.IsDir() {
			continue
		}
		removeTestFiles(servicesDir + "/" + d.Name())
	}
}

// removeTestFiles removes all files created by running truss and building the
// service from a single definition directory
func removeTestFiles(defDir string) {
	if fileExists(defDir + "/TEST-service") {
		os.RemoveAll(defDir + "/TEST-service")
		os.RemoveAll(defDir + "/third_party")
		os.RemoveAll(defDir + "/bin")
	}
}
