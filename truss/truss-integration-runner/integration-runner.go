package main

import (
	"bytes"
	"io/ioutil"
	"net"
	"path/filepath"
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
	// refCount is increased for every server/client call
	// and decreased every time one is display, for exiting
	refCount := 0

	// Loop through all directories in the running path
	dirs, err := ioutil.ReadDir(workingDirectory)
	for _, d := range dirs {
		// If this item is not a directory skip it
		if !d.IsDir() {
			continue
		}
		// Build the full path to this directory and the path to the client and server
		// binaries within it
		testPath := workingDirectory + "/" + d.Name()
		log.WithField("Test path", testPath).Debug()

		serverPath := testPath + RELATIVESERVERPATH
		clientPath := testPath + RELATIVECLIENTPATH

		// If the server and client binary exist then run them against eachother
		if fileExists(serverPath) && fileExists(clientPath) {
			port := 8082 + refCount
			debugPort := 9082 + refCount
			checkPort(port)
			log.WithFields(log.Fields{
				"testPath":  testPath,
				"port":      port,
				"debugPort": debugPort,
			}).
				Debug("LAUNCH")
			go runServerAndClient(testPath, port, debugPort, runRefs)
			refCount = refCount + 1
		}
	}

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
		refCount = refCount - 1
		if refCount == 0 {
			break
		}

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
