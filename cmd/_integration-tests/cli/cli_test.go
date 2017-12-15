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
	// Create a standalone copy of the 'basic' service binary that persists
	// through all the tests. This binary to be used to test things like flags
	// and flag-groups.

	exitCode := 1
	defer func() {
		if exitCode == 0 {
			cleanTests(basePath)
		}
		os.Exit(exitCode)
	}()

	clean := flag.Bool("clean", false, "Remove all generated test files and do nothing else")
	flag.Parse()
	if *clean {
		exitCode = 0
		return
	}

	// Cleanup so that cp works as expected
	cleanTests(basePath)

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
		return
	}

	path := filepath.Join(basePath, "0-basic")

	err = createTrussService(path)
	if err != nil {
		fmt.Printf("cannot create truss service: %v", err)
		return
	}

	err = buildTestService(filepath.Join(path, "test-service"))
	if err != nil {
		fmt.Printf("cannot build truss service: %v", err)
		return
	}

	exitCode = m.Run()
}

// Ensure that environment variables are used
func TestPortVariable(t *testing.T) {
	path := filepath.Join(basePath, "0-basic", "test-service")
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

func TestBasicTypes(t *testing.T) {
	testEndToEnd("1-basic", "getbasic", t)
}

func TestBasicTypesWithRelSVCOutFlag(t *testing.T) {
	svcOut := "./tunelab"
	path := filepath.Join(basePath, "1-basic")
	err := createTrussService(path, "--svcout", svcOut)
	if err != nil {
		t.Fatal(err)
	}
	err = buildTestService(filepath.Join(path, svcOut))
	if err != nil {
		t.Fatal(err)
	}
}

func TestBasicTypesWithTrailingSlashSVCOutFlag(t *testing.T) {
	svcOut := "./tunelab/"
	path := filepath.Join(basePath, "1-basic")
	err := createTrussService(path, "--svcout", svcOut)
	if err != nil {
		t.Fatal(err)
	}
	err = buildTestService(filepath.Join(path, svcOut, "test-service"))
	if err != nil {
		t.Fatal(err)
	}
}

func TestMultipleFiles(t *testing.T) {
	testEndToEnd("1-multifile", "getbasic", t)
}

func TestRepeatedTypes(t *testing.T) {
	testEndToEnd("2-repeated", "getrepeated", t)
}

func TestNestedTypes(t *testing.T) {
	testEndToEnd("3-nested", "getnested", t)
}

// Disabled until map types are implemented for cliclient
func _TestMapTypes(t *testing.T) {
	testEndToEnd("4-maps", "getmap", t)
}

func TestGRPCOnly(t *testing.T) {
	path := filepath.Join(basePath, "5-grpconly")
	err := createTrussService(path)
	if err != nil {
		t.Fatal(err)
	}
	err = buildTestService(filepath.Join(path, "test-service"))
	if err != nil {
		t.Fatal(err)
	}
}

func TestAdditionalBindings(t *testing.T) {
	testEndToEnd("6-additional_bindings", "getadditional", t)
}

func TestCustomHTTPVerbs(t *testing.T) {
	testEndToEnd("7-custom_http_verb", "getadditional", t)
	testEndToEnd("7-custom_http_verb", "postadditional", t)
}

func TestMessageOnly(t *testing.T) {
	path := filepath.Join(basePath, "8-message_only")
	err := createTrussService(path)
	if err != nil {
		t.Fatal(err)
	}
}

func testEndToEnd(defDir string, subcmd string, t *testing.T, trussOptions ...string) {
	path := filepath.Join(basePath, defDir)
	err := createTrussService(path, trussOptions...)
	if err != nil {
		t.Fatal(err)
	}
	path = filepath.Join(path, "test-service")
	err = buildTestService(path)
	if err != nil {
		t.Fatal(err)
	}

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
}

func createTrussService(path string, trussFlags ...string) error {
	trussOut, err := truss(path, trussFlags...)

	// If truss fails, test error and skip communication
	if err != nil {
		return errors.Errorf("Truss generation FAILED - %v\nTruss Output:\n%v Error:\n%v", path, trussOut, err)
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
	args = append(args, "-v")
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

	const serverPath = "cmd/test-server"
	const clientPath = "cmd/test"

	// Build server and client
	errChan := make(chan error)

	go goBuild("test-server", binDir, filepath.Join(relDir, serverPath), errChan)
	go goBuild("test", binDir, filepath.Join(relDir, clientPath), errChan)

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
	const relativeClientPath = "/bin/test"

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
	// Remove the 0-basic used for non building tests
	os.RemoveAll(filepath.Join(servicesDir, "0-basic"))
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
	// svcout dir
	os.RemoveAll(filepath.Join(defDir, "tunelab"))
	// service dir
	os.RemoveAll(filepath.Join(defDir, "test-service"))
	// where the binaries are compiled to
	os.RemoveAll(filepath.Join(defDir, "bin"))
	// Remove all the .pb.go files which may remain
	dirs, _ := ioutil.ReadDir(defDir)
	for _, d := range dirs {
		if strings.HasSuffix(d.Name(), ".pb.go") {
			os.RemoveAll(filepath.Join(defDir, d.Name()))
		}
	}
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
