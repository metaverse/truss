package protostuff

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/golang/protobuf/proto"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
	"github.com/pkg/errors"

	assets "github.com/TuneLab/go-truss/truss/template"
)

// GeneratePBDataStructures expects to be passed
// datastructures represened in the .proto files
func GeneratePBataStructures(protoFiles []string, svcDir string) error {

	err := outputGoogleImport(svcDir)
	if err != nil {
		return err
	}
	importPath, err := filepath.Rel(filepath.Join(os.Getenv("GOPATH"), "src"), svcDir)
	if err != nil {
		return err
	}

	genGoCode := "--go_out=Mgoogle/api/annotations.proto=" +
		importPath + "/third_party/googleapis/google/api," +
		"plugins=grpc:" +
		svcDir

	_, err = exec.LookPath("protoc-gen-go")
	if err != nil {
		return errors.Wrap(err, "protoc-gen-go not exist in $PATH")
	}

	protoDir := filepath.Dir(svcDir)
	err = protoc(protoFiles, protoDir, svcDir, svcDir, genGoCode)
	if err != nil {
		return errors.Wrap(err, "could not generate go code from .proto files")
	}

	return nil
}

// CodeGeneatorRequest gets the parsed output from protoc, marshals that output to a
// CodeGeneratorRequest.
func CodeGeneratorRequest(protoFiles []string, protoDir string) (*plugin.CodeGeneratorRequest, error) {

	protocOut, err := getProtocOutput(protoFiles, protoDir)
	if err != nil {
		return nil, errors.Wrap(err, "could not get output from protoc")
	}

	req := new(plugin.CodeGeneratorRequest)
	if err = proto.Unmarshal(protocOut, req); err != nil {
		return nil, errors.Wrap(err, "could not marshal protoc ouput to code generator request")
	}

	return req, nil
}

// ServiceFile searches through the files in the request and returns the
// path to the first one which contains a service declaration. If no file in
// the request contains a service, returns an empty string.
func ServiceFile(req *plugin.CodeGeneratorRequest, protoFileDir string) (*os.File, error) {
	var svcFileName string
	for _, file := range req.GetProtoFile() {
		if len(file.GetService()) > 0 {
			svcFileName = file.GetName()
		}
	}

	if svcFileName == "" {
		return nil, errors.New("passed protofiles contain no service")
	}

	svc, err := os.Open(filepath.Join(protoFileDir, svcFileName))

	if err != nil {
		return nil, errors.Wrapf(err, "could not open service file: %v\n in path: %v",
			protoFileDir, svcFileName)
	}

	return svc, nil
}

// getProtocOutput calls exec's $ protoc with the passed protofiles and the
// protoc-gen-truss-protocast plugin and returns the output of protoc
func getProtocOutput(protoFiles []string, protoFileDir string) ([]byte, error) {
	_, err := exec.LookPath("protoc-gen-truss-protocast")
	if err != nil {
		return nil, errors.Wrap(err, "protoc-gen-truss-protocast does not exist in $PATH")
	}

	protocOutDir, err := ioutil.TempDir("", "truss-")
	if err != nil {
		return nil, errors.Wrap(err, "could not create temp directory")
	}
	defer os.RemoveAll(protocOutDir)

	err = outputGoogleImport(protocOutDir)
	if err != nil {
		return nil, errors.Wrapf(err, "could not write protoc imports to dir: %s", protocOutDir)
	}

	const plugin = "--truss-protocast_out=."
	err = protoc(protoFiles, protoFileDir, protocOutDir, protocOutDir, plugin)
	if err != nil {
		return nil, errors.Wrap(err, "protoc failed")
	}

	fileInfo, err := ioutil.ReadDir(protocOutDir)
	if err != nil {
		return nil, errors.Wrapf(err, "could not read directory: %v", protocOutDir)
	}

	for _, f := range fileInfo {
		if !f.IsDir() {
			fPath := filepath.Join(protocOutDir, f.Name())
			protocOut, err := ioutil.ReadFile(fPath)
			if err != nil {
				return nil, errors.Wrapf(err, "cannot read file: %v", fPath)
			}
			return protocOut, nil
		}
	}

	return nil, errors.Errorf("no protoc output file found in: %v", protocOutDir)
}

// protoc exec's $ protoc on protoFiles, on their full path which is created with protoDir
func protoc(protoFiles []string, protoDir, outDir, importDir, plugin string) error {
	const googleAPIImportPath = "/third_party/googleapis"

	var fullPaths []string
	for _, f := range protoFiles {
		fullPaths = append(fullPaths, filepath.Join(protoDir, f))
	}

	cmdArgs := []string{
		"-I" + filepath.Join(importDir, googleAPIImportPath),
		"--proto_path=" + protoDir,
		plugin,
	}
	// Append each definition file path to the end of that command args
	cmdArgs = append(cmdArgs, fullPaths...)

	protocExec := exec.Command(
		"protoc",
		cmdArgs...,
	)

	protocExec.Dir = outDir

	outBytes, err := protocExec.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err,
			"protoc exec failed.\nprotoc output:\n\n%v\nprotoc arguments:\n\n%v\n\n",
			string(outBytes), protocExec.Args)
	}

	return nil
}

// mkdir acts like $ mkdir -p path
func mkdir(path string) error {
	// 0775 is the file mode that $ mkdir uses when creating a directory
	err := os.MkdirAll(path, 0775)

	return err
}

// outputGoogleImport places imported and required google.api.http protobuf option files
func outputGoogleImport(dir string) error {
	// Output files that are stored in template package
	for _, assetPath := range assets.AssetNames() {
		fileBytes, _ := assets.Asset(assetPath)
		fullPath := filepath.Join(dir, assetPath)

		// Rename .gotemplate to .go
		if strings.HasSuffix(fullPath, ".gotemplate") {
			fullPath = strings.TrimSuffix(fullPath, "template")
		}

		err := mkdir(filepath.Dir(fullPath))
		if err != nil {
			return errors.Wrapf(err, "unable to create directory for %v", filepath.Dir(fullPath))
		}

		err = ioutil.WriteFile(fullPath, fileBytes, 0666)
		if err != nil {
			return errors.Wrapf(err, "cannot create template file at path %v", fullPath)
		}
	}

	return nil
}
