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

	generatePbGoCmd = "--go_out=Mgoogle/api/annotations.proto=" + genImportPath + GOOGLE_API_HTTP_IMPORT_PATH + "/google/api,plugins=grpc:./service/DONOTEDIT/compiledpb"
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
	fmt.Println(definitionPath)

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

	err := os.MkdirAll("service/DONOTEDIT/compiledpb", 0777)
	if err != nil {
		log.WithField("DirPath", "service/DONOTEDIT/compiledpb").WithError(err).Fatal("Cannot create directories")
	}

	protoc(definitionPath, generatePbGoCmd)
	protoc(definitionPath, generateDocsCmd)
	protoc(definitionPath, generateGoKitCmd)
}

func protoc(definitionPath string, command string) error {

	protocExec := exec.Command(
		"protoc",
		"-I/usr/local/include",
		"-I.",
		"-I"+workingDirectory+GOOGLE_API_HTTP_IMPORT_PATH,
		command,
		definitionPath,
	)

	val, err := protocExec.Output()
	fmt.Println(strings.Join(protocExec.Args, " "))

	if err != nil {
		log.WithFields(log.Fields{
			"output": string(val),
			"input":  protocExec.Args,
		}).WithError(err).Fatal("Protoc call failed")
	}

	return nil

}

func check(err error) {
	if err != nil {
		log.WithError(err).Warn("Error")
		os.Exit(1)
	}

}
