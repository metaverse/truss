// Package execprotoc provides an interface for interacting with proto
// requiring only paths to files on disk
package execprotoc

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/golang/protobuf/proto"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
	"github.com/pkg/errors"
)

// GeneratePBDotGo creates .pb.go files from the passed protoPaths and writes
// them to outDir.
func GeneratePBDotGo(protoPaths, gopath []string, outDir string) error {
	genGoCode := "--go_out=" +
		"plugins=grpc:" +
		outDir

	_, err := exec.LookPath("protoc-gen-go")
	if err != nil {
		return errors.Wrap(err, "cannot find protoc-gen-go in PATH")
	}

	err = protoc(protoPaths, gopath, genGoCode)
	if err != nil {
		return errors.Wrap(err, "cannot exec protoc with protoc-gen-go")
	}

	return nil
}

// CodeGeneratorRequest returns a protoc CodeGeneratorRequest from running
// protoc on protoPaths
func CodeGeneratorRequest(protoPaths, gopath []string) (*plugin.CodeGeneratorRequest, error) {
	protocOut, err := getProtocOutput(protoPaths, gopath)
	if err != nil {
		return nil, errors.Wrap(err, "cannot get output from protoc")
	}

	req := new(plugin.CodeGeneratorRequest)
	if err = proto.Unmarshal(protocOut, req); err != nil {
		return nil, errors.Wrap(err, "cannot marshal protoc ouput to code generator request")
	}

	return req, nil
}

// ServiceFile returns the file in req that contains a service declaration.
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
		return nil, errors.Wrapf(err, "cannot open service file: %v\n in path: %v",
			protoFileDir, svcFileName)
	}

	return svc, nil
}

// getProtocOutput executes protoc with the passed protofiles and the
// protoc-gen-truss-protocast plugin and returns the output of protoc
func getProtocOutput(protoPaths, gopath []string) ([]byte, error) {
	_, err := exec.LookPath("protoc-gen-truss-protocast")
	if err != nil {
		return nil, errors.Wrap(err, "protoc-gen-truss-protocast does not exist in $PATH")
	}

	protocOutDir, err := ioutil.TempDir("", "truss-")
	if err != nil {
		return nil, errors.Wrap(err, "cannot create temp directory")
	}
	//defer os.RemoveAll(protocOutDir)

	pluginCall := "--truss-protocast_out=" + protocOutDir

	err = protoc(protoPaths, gopath, pluginCall)
	if err != nil {
		return nil, errors.Wrap(err, "protoc failed")
	}

	fileInfo, err := ioutil.ReadDir(protocOutDir)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot read directory: %v", protocOutDir)
	}

	for _, f := range fileInfo {
		if f.IsDir() {
			continue
		}
		fPath := filepath.Join(protocOutDir, f.Name())
		protocOut, err := ioutil.ReadFile(fPath)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot read file: %v", fPath)
		}
		return protocOut, nil
	}

	return nil, errors.Errorf("no protoc output file found in: %v", protocOutDir)
}

// protoc executes protoc on protoPaths
func protoc(protoPaths, gopath []string, plugin string) error {
	var cmdArgs []string

	cmdArgs = append(cmdArgs, "--proto_path="+filepath.Dir(protoPaths[0]))

	for _, gp := range gopath {
		cmdArgs = append(cmdArgs, "-I"+filepath.Join(gp, "src"))
	}

	cmdArgs = append(cmdArgs, plugin)
	// Append each definition file path to the end of that command args
	cmdArgs = append(cmdArgs, protoPaths...)

	protocExec := exec.Command(
		"protoc",
		cmdArgs...,
	)

	log.Println(protocExec.Args)
	outBytes, err := protocExec.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err,
			"protoc exec failed.\nprotoc output:\n\n%v\nprotoc arguments:\n\n%v\n\n",
			string(outBytes), protocExec.Args)
	}
	log.Println(string(outBytes))

	return nil
}
