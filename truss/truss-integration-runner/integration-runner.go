package main

import (
	"bytes"
	"io/ioutil"
	"net"
	"path/filepath"
	"strings"
	"time"

	"os"
	"os/exec"
	"strconv"

	log "github.com/Sirupsen/logrus"
)

// From within a folder with a truss `service`
// These are the paths to the compiled binaries
const RELATIVESERVERPATH = "/service/bin/server"
const RELATIVECLIENTPATH = "/service/bin/cliclient"

// runReference stores data about a client-server interaction
// This data is then used to display output
type runReference struct {
	path         string
	clientErr    bool
	serverErr    bool
	clientOutput string
	serverOutput string
}

func init() {
	log.SetLevel(log.InfoLevel)
	log.SetFormatter(&log.TextFormatter{
		ForceColors: true,
	})
}

func main() {
	workingDirectory, err := os.Getwd()
	if err != nil {
		log.WithError(err).Fatal("Cannot get working directory")
	}

	// runRefs will be passed to all gorutines running communication tests
	// and will be read to display output
	runRefs := make(chan runReference)
	// tasksCount is increased for every server/client call
	// and decreased every time one is display, for exiting
	tasksCount := 0

	done := make(chan bool)

	// Loop through all directories in the running path
	dirs, err := ioutil.ReadDir(workingDirectory)
	for _, d := range dirs {
		// If this item is not a directory skip it
		if !d.IsDir() {
			continue
		}

		// tests will be run on the fullpath to directory
		testDir := workingDirectory + "/" + d.Name()
		// On port relative to 8082, increasing by tasksCount
		port := 8082 + tasksCount

		log.WithField("Service", d.Name()).Info("Starting integration test")

		// Running the integration tests one at a time because truss running on all files at once
		// seems to slow the system more than distribute the work
		communicationTestRan := runTest(testDir, port, runRefs)

		// If communication test ran, increase the running taskCount
		if communicationTestRan {
			tasksCount = tasksCount + 1
		}
	}

	// exitWhenFinished calls os.Exit when it has received on chan `done` `tasksCount` number of times
	go exitWhenFinished(tasksCount, done)

	// range through the runRefs channel, display info if pass
	// display warn with debug info if fail
	for ref := range runRefs {
		if ref.clientErr || ref.serverErr {
			log.WithField("Service", filepath.Base(ref.path)).Warn("Communication test FAILED")
			log.Warnf("Client Output\n%v", ref.clientOutput)
			log.Warnf("Server Output\n%v", ref.serverOutput)
		} else {
			log.WithField("Service", filepath.Base(ref.path)).Info("Communication test passed")
		}
		done <- true

	}
}

// exitWhenFinished counts down from the passed tasksCount
// every time something on chan done is received
// when zero is reached, os.Exit is called
func exitWhenFinished(tasksCount int, done chan bool) {
	for _ = range done {
		tasksCount = tasksCount - 1
		if tasksCount == 0 {
			os.Exit(0)
		}
	}
}

// runTest generates, builds, and runs truss services
// testPath is the full path to the definition files
// portRef is a reference port to launch services on
// runRefs is a channel where references to client/server communication will be passed back
// runTest returns a bool representing whether or not the client/server communication was tested
func runTest(testPath string, portRef int, runRefs chan runReference) (communicationTestRan bool) {
	// Build the full path to this directory and the path to the client and server
	// binaries within it
	log.WithField("Test path", testPath).Debug()

	// Generate and build service
	truss(testPath)

	serverPath := testPath + RELATIVESERVERPATH
	clientPath := testPath + RELATIVECLIENTPATH

	// If the server and client binary exist then run them against each other
	if fileExists(serverPath) && fileExists(clientPath) {
		port := portRef
		debugPort := portRef + 1000
		checkPort(port)
		log.WithFields(log.Fields{
			"testPath":  testPath,
			"port":      port,
			"debugPort": debugPort,
		}).
			Debug("LAUNCH")
		go runServerAndClient(testPath, port, debugPort, runRefs)
		return true
	}
	return false
}

// truss calls truss on *.proto in path
// Truss logs to Stdout when generation passes or fails
func truss(path string) {
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

	log.WithField("Path", path).Debug("Exec Truss")
	val, err := trussExec.CombinedOutput()

	if err != nil {
		log.Warn(err)
		log.Warn(trussExec.Args)
		log.Warn(path)
		log.WithField("Service", filepath.Base(path)).Warn("Truss generation FAILED")
		log.Warnf("Truss Output:\n%v", string(val))
	} else {
		log.WithField("Service", filepath.Base(path)).Info("Truss generation passed")
	}
}

// checkPort checks if port is being used
// TODO: Make work
func checkPort(port int) {
	log.Debug("Checking Port")
	ips, _ := net.LookupIP("localhost")
	listener, err := net.ListenTCP("tcp",
		&net.TCPAddr{
			IP:   ips[0],
			Port: port,
		})
	_ = listener

	log.Debug("Checking Error")
	if err != nil {
		log.WithField("port", port).Warn("PORT MAY BE TAKEN")
	}
	listener.Close()

	//net.Dial("tcp", "localhost:"+strconv.Itoa(port))

}

func runServerAndClient(path string, port int, debugPort int, refChan chan runReference) {

	// Output buffer for the server Stdout and Stderr
	serverOut := bytes.NewBuffer(nil)
	// Get the server command ready with the port
	server := exec.Command(
		path+RELATIVESERVERPATH,
		"-grpc.addr",
		":"+strconv.Itoa(port),
		"-debug.addr",
		":"+strconv.Itoa(debugPort),
	)

	// Put serverOut to be the writer of data from Stdout and Stderr
	server.Stdout = serverOut
	server.Stderr = serverOut

	log.Debug("Starting the server!")
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
		log.WithField("path", path).Debug("Timer Reset")
	}
	<-t.C
	log.WithField("path", path).Debug("Timer Reset last")

	cOut, cErr := runClient(path, port)

	log.WithField("Client Output", string(cOut)).Debug("Client returned")

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
		path:         path,
		clientErr:    cErr,
		serverErr:    sErr,
		clientOutput: string(cOut),
		serverOutput: serverOut.String(),
	}
	refChan <- ref
}

func runClient(path string, port int) ([]byte, bool) {
	client := exec.Command(
		path+RELATIVECLIENTPATH,
		"-grpc.addr",
		":"+strconv.Itoa(port),
	)

	log.Debug("Starting the client!")

	cOut, err := client.CombinedOutput()

	var cErr bool
	if err != nil {
		log.WithError(err).Warn()
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
