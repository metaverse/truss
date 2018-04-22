package svcdef

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/tuneinc/truss/truss/execprotoc"
	"github.com/pkg/errors"
)

// NewFromString creates a Svcdef from a string of a valid protobuf file. Very
// useful in tests.
func NewFromString(def string, gopath []string) (*Svcdef, error) {
	const defFileName = "definition.proto"
	const goFileName = "definition.pb.go"

	// Write our proto file to a directory
	protoDir, err := ioutil.TempDir("", "trusssvcdef")
	if err != nil {
		return nil, errors.Wrap(err, "cannot create temp directory to store proto definition")
	}
	defer os.RemoveAll(protoDir)

	defPath := filepath.Join(protoDir, defFileName)

	err = ioutil.WriteFile(defPath, []byte(def), 0666)
	if err != nil {
		return nil, errors.Wrap(err, "cannot write proto definition to file")
	}

	// Create our pb.go file
	err = execprotoc.GeneratePBDotGo([]string{defPath}, gopath, protoDir)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create a pb.go file")
	}
	gPath := filepath.Join(protoDir, goFileName)

	buf, err := ioutil.ReadFile(gPath)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot read pb.go file %q", gPath)
	}
	pbgo := string(buf)

	pbgoMap := map[string]io.Reader{
		"/tmp/doesntexist.pb.go": strings.NewReader(pbgo),
	}
	pFileMap := map[string]io.Reader{
		"/tmp/doesntexist.proto": strings.NewReader(def),
	}
	sd, err := New(pbgoMap, pFileMap)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create new svcdef from pb.go and definition")
	}

	return sd, nil
}
