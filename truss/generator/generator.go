package generator

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/Sirupsen/logrus"

	templates "github.com/TuneLab/gob/truss/template"
)

func init() {
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{
		ForceColors: true,
	})
}

// GenerateMicroservice takes a golang importPath, a path to .proto definition files workingDirectory, and a slice of
// definition files DefinitionFiles and outputs a ./service direcotry in workingDirectory with a generated microservice
func GenerateMicroservice(importPath string, workingDirectory string, definitionFiles []string) {

	done := make(chan bool)

	// Stage 1
	buildDirectories(workingDirectory)
	outputGoogleImport(workingDirectory)

	// Stage 2, 3, 4
	generatePbGoCmd := "--go_out=Mgoogle/api/annotations.proto=" + importPath + "/service/DONOTEDIT/third_party/googleapis/google/api,plugins=grpc:./service/DONOTEDIT/pb"
	const generateDocsCmd = "--truss-doc_out=."
	const generateGoKitCmd = "--truss-gokit_out=."

	go protoc(workingDirectory, definitionFiles, generatePbGoCmd, done)
	go protoc(workingDirectory, definitionFiles, generateDocsCmd, done)
	go protoc(workingDirectory, definitionFiles, generateGoKitCmd, done)

	<-done
	<-done
	<-done
}

// BuildMicroservice builds a microservice using `$ go build` for a microservice generated with GenerateMicroservice
func BuildMicroservice(importPath string) {

	done := make(chan bool)

	// Stage 5
	go goBuild("server", importPath+"/service/DONOTEDIT/cmd/svc/...", done)
	go goBuild("cliclient", importPath+"/service/DONOTEDIT/cmd/cliclient/...", done)

	<-done
	<-done
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
func buildDirectories(workingDirectory string) {
	// third_party created by going through assets in template
	// and creating directoires that are not there
	for _, filePath := range templates.AssetNames() {
		fullPath := workingDirectory + "/" + filePath

		dirPath := filepath.Dir(fullPath)

		err := os.MkdirAll(dirPath, 0777)
		if err != nil {
			log.WithField("DirPath", dirPath).WithError(err).Fatal("Cannot create directories")
		}
	}

	// Create the directory where protoc will store the compiled .pb.go files
	err := os.MkdirAll(workingDirectory+"/service/DONOTEDIT/pb", 0777)
	if err != nil {
		log.WithField("DirPath", "service/DONOTEDIT/pb").WithError(err).Fatal("Cannot create directories")
	}

	// Create the directory where go build will put the compiled binaries
	err = os.MkdirAll(workingDirectory+"/service/bin", 0777)
	if err != nil {
		log.WithField("DirPath", "service/bin").WithError(err).Fatal("Cannot create directories")
	}
}

// outputGoogleImport places imported and required google.api.http protobuf option files
// into their required directories as part of stage one generation
func outputGoogleImport(workingDirectory string) {
	// Output files that are stored in template package
	for _, filePath := range templates.AssetNames() {
		fileBytes, _ := templates.Asset(filePath)
		fullPath := workingDirectory + "/" + filePath

		// Rename .gotemplate to .go
		if strings.HasSuffix(fullPath, ".gotemplate") {
			fullPath = strings.TrimSuffix(fullPath, "template")
		}

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

	// $ go get

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

	// $ go build

	goBuildExec := exec.Command(
		"go",
		"build",
		"-o",
		"service/bin/"+name,
		path,
	)

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

func protoc(workingDirectory string, definitionPaths []string, command string, done chan bool) {
	const googleApiHttpImportPath = "/service/DONOTEDIT/third_party/googleapis"

	cmdArgs := []string{
		"-I.",
		"-I" + workingDirectory + googleApiHttpImportPath,
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
