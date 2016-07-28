package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/Sirupsen/logrus"

	templates "github.com/TuneLab/gob/truss/template"
)

const GENERATED_PATH = "service"
const GOOGLE_API_HTTP_IMPORT_PATH = "/service/DONOTEDIT/third_party/googleapis"

type globalStruct struct {
	workingDirectory string
	genImportPath    string
	GOPATH           string
	generatePbGoCmd  string
	generateDocsCmd  string
	generateGoKitCmd string
}

var global globalStruct

// We build up environment knowledge here
// 1. Get working directory
// 2. Get $GOPATH
// 3. Use 1,2 to build path for golang imports for this package
// 4. Build 3 proto commands to invoke
func init() {
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{
		ForceColors: true,
	})

	var err error
	global.workingDirectory, err = os.Getwd()
	if err != nil {
		log.WithError(err).Fatal("Cannot get working directory")
	}

	global.GOPATH = os.Getenv("GOPATH")

	// From `$GOPATH/src/org/user/thing` get `org/user/thing` from importing in golang
	global.genImportPath = strings.TrimPrefix(global.workingDirectory, global.GOPATH+"/src/")

	// Generate grpc golang code
	global.generatePbGoCmd = "--go_out=Mgoogle/api/annotations.proto=" + global.genImportPath + GOOGLE_API_HTTP_IMPORT_PATH + "/google/api,plugins=grpc:./service/DONOTEDIT/pb"
	// Generate documentation
	global.generateDocsCmd = "--truss-doc_out=."
	// Generate gokit-base service
	global.generateGoKitCmd = "--truss-gokit_out=."

}

// Stages are documented in README.md
func main() {
	flag.Parse()

	if len(flag.Args()) == 0 {
		fmt.Fprintf(os.Stderr, "usage: truss microservice.proto\n")
		os.Exit(1)
	}

	definitionPaths := flag.Args()

	Stage1()

	// Stage 2, 3, 4
	Stage234(definitionPaths)

	// Stage 5
	Stage5()

}

func Stage1() {
	// Stage 1
	global.buildDirectories()
	global.outputGoogleImport()
}

func Stage234(definitionPaths []string) {
	genPbGoDone := make(chan bool)
	genDocsDone := make(chan bool)
	genGoKitDone := make(chan bool)
	go global.protoc(definitionPaths, global.generatePbGoCmd, genPbGoDone)
	go global.protoc(definitionPaths, global.generateDocsCmd, genDocsDone)
	go global.protoc(definitionPaths, global.generateGoKitCmd, genGoKitDone)
	<-genPbGoDone
	<-genDocsDone
	<-genGoKitDone
}

func Stage5() {
	serverDone := make(chan bool)
	clientDone := make(chan bool)
	go goBuild("server", "./service/DONOTEDIT/cmd/svc/...", serverDone)
	go goBuild("cliclient", "./service/DONOTEDIT/cmd/cliclient/...", clientDone)
	<-serverDone
	<-clientDone
}

// buildDirectories puts the following directories in place
// .
// └── service
//     ├── bin
//     └── DONOTEDIT
//         ├── pb
//         └── third_party
//             └── googleapis
//                 └── google
//                     └── api
func (g globalStruct) buildDirectories() {
	// third_party created by going through assets in template
	// and creating directoires that are not there
	for _, filePath := range templates.AssetNames() {
		fullPath := g.workingDirectory + "/" + filePath

		dirPath := filepath.Dir(fullPath)

		err := os.MkdirAll(dirPath, 0777)
		if err != nil {
			log.WithField("DirPath", dirPath).WithError(err).Fatal("Cannot create directories")
		}
	}

	// Create the directory where protoc will store the compiled .pb.go files
	err := os.MkdirAll("service/DONOTEDIT/pb", 0777)
	if err != nil {
		log.WithField("DirPath", "service/DONOTEDIT/pb").WithError(err).Fatal("Cannot create directories")
	}

	// Create the directory where go build will put the compiled binaries
	err = os.MkdirAll("service/bin", 0777)
	if err != nil {
		log.WithField("DirPath", "service/bin").WithError(err).Fatal("Cannot create directories")
	}
}

// outputGoogleImport places imported and required google.api.http protobuf option files
// into their required directories as part of stage one generation
func (g globalStruct) outputGoogleImport() {
	// Output files that are stored in template package
	for _, filePath := range templates.AssetNames() {
		fileBytes, _ := templates.Asset(filePath)
		fullPath := g.workingDirectory + "/" + filePath

		err := ioutil.WriteFile(fullPath, fileBytes, 0666)
		if err != nil {
			log.WithField("FilePath", fullPath).WithError(err).Fatal("Cannot create ")
		}
	}
}

// goBuild calls the `$ go get ` to install dependenices
// and then calls `$ go build service/bin/$name $path`
// to put the iterating binaries in the correct place
func goBuild(name string, path string, done chan bool) {

	goGetExec := exec.Command(
		"go",
		"get",
		"-d",
		"-v",
		path,
	)

	goGetExec.Stderr = os.Stderr

	log.WithField("cmd", strings.Join(goGetExec.Args, " ")).Info("go get")
	val, err := goGetExec.Output()

	if err != nil {
		log.WithFields(log.Fields{
			"output": string(val),
			"input":  goGetExec.Args,
		}).WithError(err).Warn("go get failed")
	}

	goBuildExec := exec.Command(
		"go",
		"build",
		"-o",
		"service/bin/"+name,
		path,
	)
	//env := os.Environ()
	//env = append(env, "CGO_ENABLED=0")
	//goBuildExec.Env = env

	goBuildExec.Stderr = os.Stderr

	log.WithField("cmd", strings.Join(goBuildExec.Args, " ")).Info("go build")
	val, err = goBuildExec.Output()

	if err != nil {
		log.WithFields(log.Fields{
			"output": string(val),
			"input":  goBuildExec.Args,
		}).WithError(err).Fatal("go build failed")
	}
	done <- true
}

func (g globalStruct) protoc(definitionPaths []string, command string, done chan bool) {
	cmdArgs := []string{
		"-I.",
		"-I" + g.workingDirectory + GOOGLE_API_HTTP_IMPORT_PATH,
		command,
	}
	// Append each definition file path to the end of that command args
	cmdArgs = append(cmdArgs, definitionPaths...)

	protocExec := exec.Command(
		"protoc",
		cmdArgs...,
	)

	protocExec.Stderr = os.Stderr

	log.WithField("cmd", strings.Join(protocExec.Args, " ")).Info("protoc")
	val, err := protocExec.Output()

	if err != nil {
		log.WithFields(log.Fields{
			"output": string(val),
			"input":  protocExec.Args,
		}).WithError(err).Fatal("Protoc call failed")
	}
	done <- true
}
