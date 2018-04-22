// Package parsesvcname will parse the service name of a protobuf package. The
// name returned will always be camelcased according to the conventions
// outlined in github.com/golang/protobuf/protoc-gen-go/generator.CamelCase.
package parsesvcname

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/tuneinc/truss/svcdef"
	"github.com/tuneinc/truss/truss/execprotoc"
	"github.com/pkg/errors"
)

// FromPaths accepts the paths of protobuf definition files and returns the
// name of the service in that protobuf definition file.
func FromPaths(gopath []string, protoDefPaths []string) (string, error) {
	td, err := ioutil.TempDir("", "parsesvcname")
	defer os.RemoveAll(td)
	if err != nil {
		return "", errors.Wrap(err, "failed to create temporary directory for .pb.go files")
	}
	err = execprotoc.GeneratePBDotGo(protoDefPaths, gopath, td)
	if err != nil {
		return "", errors.Wrap(err, "failed to generate .pb.go files from proto definition files")
	}

	// Get path names of .pb.go files
	pbgoPaths := []string{}
	for _, p := range protoDefPaths {
		base := filepath.Base(p)
		barename := strings.TrimSuffix(base, filepath.Ext(p))
		pbgp := filepath.Join(td, barename+".pb.go")
		pbgoPaths = append(pbgoPaths, pbgp)
	}

	// Open all .pb.go files and store in map to be passed to svcdef.New()
	openFiles := func(paths []string) (map[string]io.Reader, error) {
		rv := map[string]io.Reader{}
		for _, p := range paths {
			reader, err := os.Open(p)
			if err != nil {
				return nil, errors.Wrapf(err, "cannot open file %q", p)
			}
			rv[p] = reader
		}
		return rv, nil
	}
	pbgoFiles, err := openFiles(pbgoPaths)
	if err != nil {
		return "", errors.Wrap(err, "cannot open all .pb.go files")
	}
	pbFiles, err := openFiles(protoDefPaths)
	if err != nil {
		return "", errors.Wrap(err, "cannot open all .proto files")
	}

	sd, err := svcdef.New(pbgoFiles, pbFiles)
	if err != nil {
		return "", errors.Wrapf(err, "failed to create service definition; did you pass ALL the protobuf files to truss?")
	}

	if sd.Service == nil {
		return "", errors.New("no service defined")
	}

	return sd.Service.Name, nil
}

func FromReaders(gopath []string, protoDefReaders []io.Reader) (string, error) {
	protoDir, err := ioutil.TempDir("", "parsesvcname-fromreaders")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temporary directory for protobuf files")
	}
	// Ensures all the temporary files are removed
	defer os.RemoveAll(protoDir)

	protoDefPaths := []string{}
	for _, rdr := range protoDefReaders {
		f, err := ioutil.TempFile(protoDir, "parsesvcname-fromreader")
		_, err = io.Copy(f, rdr)
		if err != nil {
			return "", errors.Wrap(err, "couldn't copy contents of our proto file into the os.File: ")
		}
		path := f.Name()
		f.Close()
		protoDefPaths = append(protoDefPaths, path)
	}
	return FromPaths(gopath, protoDefPaths)
}
