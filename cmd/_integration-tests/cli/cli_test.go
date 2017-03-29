package cli

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
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

var basePath string

func TestMain(m *testing.M) {
	wd, err := os.Getwd()
	if err != nil {
		os.Exit(1)
	}

	basePath = filepath.Join(wd, definitionDirectory)

	clean := flag.Bool("clean", false, "Remove all generated test files and do nothing else")
	flag.Parse()
	if *clean {
		cleanTests(basePath)
		os.Exit(0)
	}

	// Create a standalone copy of the 'basic' service binary that persists
	// through all the tests. This binary to be used to test things like flags
	// and flag-groups.
	exitCode := 0

	// Copy "1-basic" into a special "0-basic" which is removed
	copy := exec.Command(
		"cp",
		"-r",
		filepath.Join(basePath, "1-basic"),
		filepath.Join(basePath, "0-basic"),
	)
	copy.Stdout = os.Stdout
	copy.Stderr = os.Stderr
	err = copy.Run()
	if err != nil {
		fmt.Printf("cannot copy '0-basic' service: %v", err)
		os.Exit(1)
	}

	err = createTrussService(filepath.Join(basePath, "0-basic"))
	if err != nil {
		fmt.Printf("cannot create truss service: %v", err)
		os.Exit(1)
	}

	defer func() {
		os.RemoveAll(filepath.Join(basePath, "0-basic"))
		os.Exit(exitCode)
	}()

	exitCode = m.Run()
}

func TestBasicTypes(t *testing.T) {
	testEndToEnd("1-basic", "getbasic", t)
}

func TestBasicTypesWithPBOutFlag(t *testing.T) {
	testEndToEnd("1-basic", "getbasic", t,
		"--pbout",
		"github.com/TuneLab/go-truss/cmd/_integration-tests/cli/test-service-definitions/1-basic/pbout")
}

func TestBasicTypesWithRelPBOutFlag(t *testing.T) {
	testEndToEnd("1-basic", "getbasic", t,
		"--pbout",
		"./pbout")
}

func TestBasicTypesWithRelSVCOutFlag(t *testing.T) {
	testEndToEnd("1-basic", "getbasic", t,
		"--svcout",
		"./1-basic")
}

func TestMultipleFiles(t *testing.T) {
	testEndToEnd("1-multifile", "getbasic", t)
}

// Disabled until repeated types are implemented for cliclient
func TestRepeatedTypes(t *testing.T) {
	testEndToEnd("2-repeated", "getrepeated", t)
}

// Disabled until nested types are implemented for cliclient
func _TestNestedTypes(t *testing.T) {
	testEndToEnd("3-nested", "getnested", t)
}

// Disabled until map types are implemented for cliclient
func _TestMapTypes(t *testing.T) {
	testEndToEnd("4-maps", "getmap", t)
}

// Ensure that environment variables are used
func TestPortVariable(t *testing.T) {
	path := filepath.Join(basePath, "0-basic")
	grpcPort := strconv.Itoa(FindFreePort())
	httpPort := strconv.Itoa(FindFreePort())
	debugPort := strconv.Itoa(FindFreePort())

	// Set environment variables
	defer os.Unsetenv("PORT")
	if err := os.Setenv("PORT", httpPort); err != nil {
		t.Fatal(err)
	}

	// launch long running server
	server, srvrOut, errc := runServer(path,
		"-grpc.addr", ":"+grpcPort,
		"-debug.addr", ":"+debugPort)
	// run client with http transport
	clientHTTP, errHTTP := runClient(path, "-http.addr", ":"+httpPort, "getbasic")
	if errHTTP != nil {
		t.Error(string(clientHTTP))
		t.Error(errHTTP)
	}

	err := reapServer(server, errc)
	if err != nil {
		t.Error(srvrOut.String())
		t.Fatalf("cannot reap server: %v", err)
	}

}

func testEndToEnd(defDir string, subcmd string, t *testing.T, trussOptions ...string) {
	path := filepath.Join(basePath, defDir)
	createTrussService(path)

	grpcPort := strconv.Itoa(FindFreePort())
	httpPort := strconv.Itoa(FindFreePort())
	debugPort := strconv.Itoa(FindFreePort())

	// launch long running server
	server, srvrOut, errc := runServer(path,
		"-grpc.addr", ":"+grpcPort,
		"-http.addr", ":"+httpPort,
		"-debug.addr", ":"+debugPort)

	// run client with grpc transport
	clientGRPC, errGRPC := runClient(path, "-grpc.addr", ":"+grpcPort, subcmd)
	// run client with http transport
	clientHTTP, errHTTP := runClient(path, "-http.addr", ":"+httpPort, subcmd)

	// check server for errors and kill if needed
	errSRVR := reapServer(server, errc)

	if errGRPC != nil || errHTTP != nil || errSRVR != nil {
		t.Logf("Communication test FAILED - %v", filepath.Base(path))
		t.Logf("Client gRPC Output\n%v", string(clientGRPC))
		t.Logf("Client HTTP Output\n%v", string(clientHTTP))
		t.Logf("Server Output\n%v", srvrOut.String())
		t.FailNow()
	}

	// If nothing failed, delete the generated files
	removeTestFiles(path)
}

func createTrussService(path string, trussFlags ...string) error {
	// Remove tests if they exists
	removeTestFiles(path)

	trussOut, err := truss(path, trussFlags...)

	// If truss fails, test error and skip communication
	if err != nil {
		return errors.Errorf("Truss generation FAILED - %v\nTruss Output:\n%v", path, trussOut)
	}

	// Build the service to be tested
	err = buildTestService(path)
	if err != nil {
		return errors.Errorf("Could not build service. Error: %v", err)
	}
	return nil
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

// buildTestService builds a truss service with the package "test"
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

	const serverPath = "/test-service/test-server"
	const clientPath = "/test-service/test-cli-client"

	// Build server and client
	errChan := make(chan error)

	go goBuild("test-server", binDir, filepath.Join(relDir, serverPath), errChan)
	go goBuild("test-cli-client", binDir, filepath.Join(relDir, clientPath), errChan)

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

func runServer(path string, flags ...string) (*exec.Cmd, *bytes.Buffer, chan error) {
	// From within a folder with a truss `service`
	// These are the paths to the compiled binaries
	const relativeServerPath = "/bin/test-server"

	// Output buffer for the server Stdout and Stderr
	srvrOut := bytes.NewBuffer(nil)
	// Get the server command ready with the port
	server := exec.Command(
		path+relativeServerPath,
		flags...,
	)

	// Put srvrOut to be the writer of data from Stdout and Stderr
	server.Stdout = srvrOut
	server.Stderr = srvrOut

	// Start the server
	errc := make(chan error)
	go func() {
		err := server.Run()
		errc <- err
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

	return server, srvrOut, errc
}

func reapServer(server *exec.Cmd, errc chan error) error {
	select {
	// Case server errored and exited
	case err := <-errc:
		return err
	default:
		break
	}

	// If the server ever stopped then it errored
	// If it did not stop, kill it and see if that errors
	if server.Process == nil {
		// This likely means the server never started
		return errors.New("server cannot be reaped; server not running")
	}
	// If the Process is not nil, kill it, clean up our mess
	err := server.Process.Kill()
	if err != nil {
		return err
	}

	return nil
}

func runClient(path string, flags ...string) ([]byte, error) {
	const relativeClientPath = "/bin/test-cli-client"

	client := exec.Command(
		path+relativeClientPath,
		flags...,
	)

	return client.CombinedOutput()
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
	os.RemoveAll(filepath.Join(defDir, "test-service"))
	os.RemoveAll(filepath.Join(defDir, "bin"))
	os.RemoveAll(filepath.Join(defDir, "pbout"))
	os.MkdirAll(filepath.Join(defDir, "pbout"), 0777)
}

// FindFreePort returns an open TCP port. That port could be taken in the time
// between this function returning and you opening it again.
func FindFreePort() int {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		panic(err)
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		panic(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}
