package cli

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

const definitionDirectory = "test-service-definitions"

func init() {
	clean := flag.Bool("clean", false, "Remove all generated test files and do nothing else")
	flag.Parse()
	if *clean {
		wd, err := os.Getwd()
		if err != nil {
			os.Exit(1)
		}
		cleanTests(filepath.Join(wd, definitionDirectory))
		os.Exit(0)
	}
}

func TestBasicTypes(t *testing.T) {
	testEndToEnd("1-basic", t)
}

func TestBasicTypesWithPBOutFlag(t *testing.T) {
	testEndToEnd("1-basic", t,
		"-pbout",
		"github.com/TuneLab/go-truss/cmd/_integration-tests/cli/test-service-definitions/1-basic/pbout")
}

func TestMultipleFiles(t *testing.T) {
	testEndToEnd("1-multifile", t)
}

// Disabled until repeated types are implemented for cliclient
func TestRepeatedTypes(t *testing.T) {
	testEndToEnd("2-repeated", t)
}

// Disabled until nested types are implemented for cliclient
func _TestNestedTypes(t *testing.T) {
	testEndToEnd("3-nested", t)
}

// Disabled until map types are implemented for cliclient
func _TestMapTypes(t *testing.T) {
	testEndToEnd("4-maps", t)
}

// runReference stores data about a client-server interaction
// This data is then used to display output
type runReference struct {
	name             string
	clientErr        bool
	clientHTTPErr    bool
	serverErr        bool
	clientOutput     string
	clientHTTPOutput string
	serverOutput     string
}

func testEndToEnd(defDir string, t *testing.T, trussOptions ...string) {
	port := 45360
	wd, _ := os.Getwd()

	fullpath := filepath.Join(wd, definitionDirectory, defDir)

	// Remove tests if they exists
	removeTestFiles(fullpath)

	trussOut, err := truss(fullpath, trussOptions...)

	// If truss fails, test error and skip communication
	if err != nil {
		t.Fatalf("Truss generation FAILED - %v\nTruss Output:\n%v", defDir, trussOut)
	}

	// Build the service to be tested
	err = buildTestService(fullpath)
	if err != nil {
		t.Fatalf("Could not build service. Error: %v", err)
	}

	// Run them save a reference to each run
	ref := runServerAndClient(fullpath, port, port+1000)
	if ref.clientErr || ref.clientHTTPErr || ref.serverErr {
		t.Logf("Communication test FAILED - %v", ref.name)
		t.Logf("Client Output\n%v", ref.clientOutput)
		t.Logf("Client HTTP Output\n%v", ref.clientHTTPOutput)
		t.Logf("Server Output\n%v", ref.serverOutput)
		t.FailNow()
	}

	// If nothing failed, delete the generated files
	removeTestFiles(fullpath)
}

// truss calls truss on *.proto in path
// Truss logs to Stdout when generation passes or fails
func truss(path string, options ...string) (string, error) {
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

	args := append(options, protofiles...)

	trussExec := exec.Command(
		"truss",
		args...,
	)
	trussExec.Dir = path

	out, err := trussExec.CombinedOutput()

	return string(out), err
}

// buildTestService builds a truss service with the package TEST
// into the `serviceDir`/bin directory
func buildTestService(serviceDir string) (err error) {

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	relDir, err := filepath.Rel(wd, serviceDir)
	if err != nil {
		return err
	}

	binDir := serviceDir + "/bin"

	err = os.MkdirAll(binDir, 0777)
	if err != nil {
		return err
	}

	const serverPath = "/TEST-service/TEST-server"
	const clientPath = "/TEST-service/TEST-cli-client"

	// Build server and client
	errChan := make(chan error)

	go goBuild("TEST-server", binDir, filepath.Join(relDir, serverPath), errChan)
	go goBuild("TEST-cli-client", binDir, filepath.Join(relDir, clientPath), errChan)

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
func goBuild(name, outputPath, relCodePath string, errChan chan error) {

	// $ go get

	goGetExec := exec.Command(
		"go",
		"get",
		"-d",
		"-v",
		"./"+relCodePath,
	)

	err := goGetExec.Run()

	if err != nil {
		errChan <- errors.Wrapf(err, "could not $ go get %v", relCodePath)
		return
	}

	// $ go build

	goBuildExec := exec.Command(
		"go",
		"build",
		"-o",
		outputPath+"/"+name,
		"./"+relCodePath,
	)

	outBytes, err := goBuildExec.CombinedOutput()
	if err != nil {
		errChan <- errors.Wrapf(err, "could not $ go build %v. \nOutput: \n%s\n", relCodePath, string(outBytes))
		return
	}

	errChan <- nil
}

// runServerAndClient execs a TEST-server and TEST-client and puts a
// runReference to their interaction on the runRefs channel
func runServerAndClient(path string, port int, debugPort int) runReference {
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

	cOut, cErr := runClient(path, "grpc", port)
	cHTTPOut, cHTTPErr := runClient(path, "http", port-70)

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
		name:             filepath.Base(path),
		clientErr:        cErr,
		clientHTTPErr:    cHTTPErr,
		serverErr:        sErr,
		clientOutput:     string(cOut),
		clientHTTPOutput: string(cHTTPOut),
		serverOutput:     serverOut.String(),
	}

	return ref
}

func runClient(path string, trans string, port int) ([]byte, bool) {
	const relativeClientPath = "/bin/TEST-cli-client"

	var client *exec.Cmd
	switch trans {
	case "http":
		client = exec.Command(
			path+relativeClientPath,
			"-http.addr",
			":"+strconv.Itoa(port),
		)
	case "grpc":
		client = exec.Command(
			path+relativeClientPath,
			"-grpc.addr",
			":"+strconv.Itoa(port),
		)
	}

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
		removeTestFiles(filepath.Join(servicesDir, d.Name()))
	}
}

// removeTestFiles removes all files created by running truss and building the
// service from a single definition directory
func removeTestFiles(defDir string) {
	os.RemoveAll(filepath.Join(defDir, "TEST-service"))
	os.RemoveAll(filepath.Join(defDir, "bin"))
	os.RemoveAll(filepath.Join(defDir, "pbout"))
	os.MkdirAll(filepath.Join(defDir, "pbout"), 0777)
}
