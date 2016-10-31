package svcdef

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/TuneLab/go-truss/truss/execprotoc"
	"github.com/pkg/errors"
)

func NewFromString(def string) (*Svcdef, error) {
	const defFileName = "definition.proto"
	const goFileName = "definition.pb.go"

	// Write our proto file to a directory
	protoDir, err := ioutil.TempDir("./", "truss-svcdef-")
	if err != nil {
		return nil, errors.Wrap(err, "could not create temp directory to store proto definition")
	}
	defer os.RemoveAll(protoDir)

	defPath := filepath.Join(protoDir, defFileName)

	err = ioutil.WriteFile(defPath, []byte(def), 0666)
	if err != nil {
		return nil, errors.Wrap(err, "could not write proto definition to file")
	}

	cur, err := filepath.Abs("./")
	if err != nil {
		return nil, errors.Wrap(err, "could not get absolute path for ./")
	}

	// Create our pb.go file
	err = execprotoc.GeneratePBDotGo([]string{defPath}, cur, protoDir)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create a pb.go file")
	}
	gPath := filepath.Join(protoDir, goFileName)

	buf, err := ioutil.ReadFile(gPath)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to read pb.go file %q", gPath)
	}
	pbgo := string(buf)

	sd, err := New([]io.Reader{strings.NewReader(pbgo)}, []io.Reader{strings.NewReader(def)})
	if err != nil {
		return nil, errors.Wrap(err, "could not create new svcdef from pb.go and definition")
	}

	return sd, nil
}
