package protostage

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/golang/protobuf/proto"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
	"github.com/pkg/errors"

	templates "github.com/TuneLab/go-truss/truss/template"
)

// Stage outputs the third_party imports that are required for protoc imports
// and go build.
func Stage(protoDir string) error {
	err := buildDirectories(protoDir)
	if err != nil {
		return err
	}

	err = outputGoogleImport(protoDir)
	if err != nil {
		return err
	}

	return nil
}

// GeneratePBDataStructures calls $ protoc with the protoc-gen-go to output
// .pb.go files in ./service/DONOTEDIT/pb which contain the golang
// datastructures represened in the .proto files
func GeneratePBDataStructures(protoFiles []string, protoDir, importPath, packageName string) error {
	pbDataStructureDir := "/" + packageName + "-service/"

	genGoCode := "--go_out=Mgoogle/api/annotations.proto=" +
		importPath +
		"/third_party/googleapis/google/api," +
		"plugins=grpc:" +
		protoDir +
		"/" + packageName + "-service"

	_, err := exec.LookPath("protoc-gen-go")
	if err != nil {
		return errors.Wrap(err, "protoc-gen-go not exist in $PATH")
	}

	err = protoc(protoFiles, protoDir, protoDir+pbDataStructureDir, genGoCode)
	if err != nil {
		return errors.Wrap(err, "could not generate go code from .proto files")
	}

	return nil
}

// Compose gets the parsed output from protoc, marshals that output to a
// CodeGeneratorRequest. Then Compose finds the .proto file containing the
// service definition and returns the CodeGeneratorRequest and the service
// definition file
func Compose(protoFiles []string, protoDir string) (*plugin.CodeGeneratorRequest, *os.File, error) {

	protocOut, err := getProtocOutput(protoFiles, protoDir)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not get output from protoc")
	}

	req := new(plugin.CodeGeneratorRequest)
	if err = proto.Unmarshal(protocOut, req); err != nil {
		return nil, nil, errors.Wrap(err, "could not marshal protoc ouput to code generator request")
	}

	svcFile, err := findServiceFile(req, protoDir)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to find which input file contains a service")
	}

	return req, svcFile, nil
}

// absoluteDir takes a path to file or directory if path is to a file return
// the absolute path to the directory containing the file if path is to a
// directory return the absolute path of that directory
func absoluteDir(path string) (string, error) {
	absP, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return "", errors.Wrapf(err, "cannot find absolute path of %v", path)
	}

	return absP, nil
}

// getProtocOutput calls exec's $ protoc with the passed protofiles and the
// protoc-gen-truss-protocast plugin and returns the output of protoc
func getProtocOutput(protoFiles []string, protoFileDir string) ([]byte, error) {
	_, err := exec.LookPath("protoc-gen-truss-protocast")
	if err != nil {
		return nil, errors.Wrap(err, "protoc-gen-truss-protocast does not exist in $PATH")
	}

	protocOutDir, err := ioutil.TempDir("", "truss-")
	defer os.RemoveAll(protocOutDir)

	log.WithField("Protoc output dir", protocOutDir).Debug("Protoc output directory created")

	const plugin = "--truss-protocast_out=."
	err = protoc(protoFiles, protoFileDir, protocOutDir, plugin)
	if err != nil {
		return nil, errors.Wrap(err, "protoc failed")
	}

	fileInfo, err := ioutil.ReadDir(protocOutDir)
	if err != nil {
		return nil, errors.Wrapf(err, "could not read directory: %v", protocOutDir)
	}

	var protocOut []byte
	if len(fileInfo) > 0 {
		fileName := fileInfo[0].Name()
		filePath := protocOutDir + "/" + fileName
		protocOut, err = ioutil.ReadFile(filePath)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot read file: %v", filePath)
		}
	} else {
		return nil, errors.Errorf("no protoc output file found in: %v", protocOutDir)
	}

	return protocOut, nil
}

// protoc exec's $ protoc on protoFiles, on their full path which is created with protoDir
func protoc(protoFiles []string, protoDir, outDir, plugin string) error {
	const googleAPIHTTPImportPath = "/third_party/googleapis"

	var fullPaths []string
	for _, f := range protoFiles {
		fullPaths = append(fullPaths, protoDir+"/"+f)
	}

	cmdArgs := []string{
		//"-I.",
		"-I" + protoDir + googleAPIHTTPImportPath,
		"--proto_path=" + protoDir,
		plugin,
	}
	// Append each definition file path to the end of that command args
	cmdArgs = append(cmdArgs, fullPaths...)

	protocExec := exec.Command(
		"protoc",
		cmdArgs...,
	)

	log.Debug(protocExec.Args)

	protocExec.Dir = outDir

	outBytes, err := protocExec.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err,
			"protoc exec failed.\nprotoc output:\n\n%v\nprotoc arguments:\n\n%v\n\n",
			string(outBytes), protocExec.Args)
	}

	return nil
}

// findServiceFile Searches through the files in the request and returns the
// path to the first one which contains a service declaration. If no file in
// the request contains a service, returns an empty string.
func findServiceFile(req *plugin.CodeGeneratorRequest, protoFileDir string) (*os.File, error) {
	var svcFileName string
	for _, file := range req.GetProtoFile() {
		if len(file.GetService()) > 0 {
			svcFileName = file.GetName()
		}
	}

	if svcFileName == "" {
		return nil, errors.New("passed protofiles contain no service")
	}

	svc, err := os.Open(protoFileDir + "/" + svcFileName)

	if err != nil {
		return nil, errors.Wrapf(err, "could not open service file: %v\n in path: %v",
			protoFileDir, svcFileName)
	}

	return svc, nil
}

// buildDirectories outputs the directories of the third_party imports
func buildDirectories(protoDir string) error {
	// third_party created by going through assets in template
	// and creating directoires that are not there
	for _, fp := range templates.AssetNames() {
		dir := filepath.Dir(protoDir + "/" + fp)
		err := mkdir(dir)
		if err != nil {
			return errors.Wrapf(err, "unable to create directory for %v", dir)
		}
	}

	return nil
}

// mkdir acts like $ mkdir -p path
func mkdir(path string) error {
	// 0775 is the file mode that $ mkdir uses when creating a directoru
	err := os.MkdirAll(path, 0775)

	return err
}

// outputGoogleImport places imported and required google.api.http protobuf option files
// into their required directories as part of stage one generation
func outputGoogleImport(workingDirectory string) error {
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
			return errors.Wrapf(err, "cannot create template file at path %v", fullPath)
		}
	}

	return nil
}
