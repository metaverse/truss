package main

import (
	"bytes"

	"os"
	"os/exec"
	"strconv"

	log "github.com/Sirupsen/logrus"
)

const RELATIVESERVERPATH = "/service/bin/server"
const RELATIVECLIENTPATH = "/service/bin/cliclient"

type BinReference struct {
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

	serverPath := workingDirectory + RELATIVESERVERPATH
	clientPath := workingDirectory + RELATIVECLIENTPATH

	refs := make(chan BinReference)
	log.Info("About to launch to goroutine to start the server")
	log.WithField("Server path", serverPath).Info("Server")
	log.WithField("Server exists", fileExists(serverPath)).Info("Server")
	if fileExists(serverPath) && fileExists(clientPath) {
		log.Info("LAUNCH")
		go runServerAndClient(workingDirectory, 8082, refs)
	}
	ref := <-refs
	log.WithField("ref", ref).Info("We Made it!")
}

func runServerAndClient(path string, port int, refChan chan BinReference) {
	// Output buffer for the server Stdout and Stderr
	serverOut := bytes.NewBuffer(nil)
	// Get the server command ready with the port
	server := exec.Command(
		path+RELATIVESERVERPATH,
		"-grpc.addr",
		":"+strconv.Itoa(port),
	)

	// Put serverOut to be the writer of data from Stdout and Stderr
	// TODO: Seperate out Stdout and Stderr
	// Assume if we hear from Stderr at any time that we have failed, consult with Leland about this
	server.Stdout = serverOut
	server.Stderr = serverOut

	log.Info("Starting the server!")
	// Start the server
	err := server.Start()
	if err != nil {
		log.WithError(err).Fatal("Fatal Error")
	}

	// Wait until we hear from the server
	// TODO: Stderr checking
	for serverOut.Len() == 0 {
	}

	client := exec.Command(
		path+RELATIVECLIENTPATH,
		"-grpc.addr",
		":"+strconv.Itoa(port),
	)

	log.Info("Starting the client!")
	//client.Stderr = os.Stdout
	//client.Stdout = os.Stdout

	cOut, err := client.CombinedOutput()
	var cErr bool
	if err != nil {
		cErr = true
	} else {
		cErr = false
	}

	log.WithField("Client Output", cOut).Info("Client returned")
	log.WithField("Client Output", string(cOut)).Info("Client returned")

	log.Info("About to check server Process State")
	// If the server already exited then it errored out
	err = server.Process.Kill()

	var sErr bool
	if err != nil {
		sErr = true
	} else {
		sErr = false
	}

	// Construct a reference to what happened here
	ref := BinReference{
		path:         path,
		clientErr:    cErr,
		serverErr:    sErr,
		clientOutput: string(cOut),
		serverOutput: serverOut.String(),
	}
	refChan <- ref
}

// fileExists checks if a file at the given path exists. Returns true if the
// file exists, and false if the file does not exist.
func fileExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}
