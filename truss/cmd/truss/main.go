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

	"github.com/TuneLab/gob/truss/data"
)

const GENERATED_PATH = "service"
const GOOGLE_API_HTTP_IMPORT_PATH = "/service/DONOTEDIT/third_party/googleapis"

var workingDirectory string
var genImportPath string
var GOPATH string

var generatePbGoCmd string
var generateDocsCmd string
var generateGoKitCmd string

func init() {
	log.SetLevel(log.DebugLevel)

	var err error
	workingDirectory, err = os.Getwd()
	if err != nil {
		log.WithError(err).Fatal("Cannot get working directory")
	}

	GOPATH = os.Getenv("GOPATH")

	genImportPath = strings.TrimPrefix(workingDirectory, GOPATH+"/src/")

	generatePbGoCmd = "--go_out=Mgoogle/api/annotations.proto=" + genImportPath + GOOGLE_API_HTTP_IMPORT_PATH + "/google/api,plugins=grpc:./service/DONOTEDIT/pb"
	generateDocsCmd = "--gendoc_out=."
	generateGoKitCmd = "--truss-gokit_out=."
}

func main() {
	flag.Parse()

	if len(flag.Args()) != 1 {
		fmt.Fprintf(os.Stderr, "usage: truss microservice.proto\n")
		os.Exit(1)
	}

	definitionPath := flag.Arg(0)

	for _, filePath := range data.AssetNames() {
		fileBytes, _ := data.Asset(filePath)
		fullPath := workingDirectory + "/" + filePath

		dirPath := filepath.Dir(fullPath)

		err := os.MkdirAll(dirPath, 0777)
		if err != nil {
			log.WithField("DirPath", dirPath).WithError(err).Fatal("Cannot create directories")
		}

		err = ioutil.WriteFile(fullPath, fileBytes, 0666)
		if err != nil {
			log.WithField("FilePath", fullPath).WithError(err).Fatal("Cannot create ")
		}
	}

	err := os.MkdirAll("service/DONOTEDIT/pb", 0777)
	if err != nil {
		log.WithField("DirPath", "service/DONOTEDIT/pb").WithError(err).Fatal("Cannot create directories")
	}
	protoc(definitionPath, generatePbGoCmd)

	protoc(definitionPath, generateDocsCmd)
	protoc(definitionPath, generateGoKitCmd)

	err = os.MkdirAll("service/bin", 0777)
	if err != nil {
		log.WithField("DirPath", "service/bin").WithError(err).Fatal("Cannot create directories")
	}

	goBuild("server", "./service/DONOTEDIT/cmd/svc/...")
	goBuild("cliclient", "./service/DONOTEDIT/cmd/cliclient/...")

}

func goBuild(name string, path string) {

	goBuildExec := exec.Command(
		"go",
		"build",
		"-o",
		"service/bin/"+name,
		path,
	)

	log.WithField("cmd", strings.Join(goBuildExec.Args, " ")).Info("go build")
	val, err := goBuildExec.Output()

	if err != nil {
		log.WithFields(log.Fields{
			"output": string(val),
			"input":  goBuildExec.Args,
		}).WithError(err).Fatal("Protoc call failed")
	}
}

func protoc(definitionPath string, command string) {

	protocExec := exec.Command(
		"protoc",
		"-I/usr/local/include",
		"-I.",
		"-I"+workingDirectory+GOOGLE_API_HTTP_IMPORT_PATH,
		command,
		definitionPath,
	)

	log.WithField("cmd", strings.Join(protocExec.Args, " ")).Info("protoc")
	val, err := protocExec.Output()

	if err != nil {
		log.WithFields(log.Fields{
			"output": string(val),
			"input":  protocExec.Args,
		}).WithError(err).Fatal("Protoc call failed")
	}

}

func check(err error) {
	if err != nil {
		log.WithError(err).Warn("Error")
		os.Exit(1)
	}

}
