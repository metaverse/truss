package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/TuneLab/go-truss/truss/protostage"
	"github.com/TuneLab/go-truss/truss/truss"

	"github.com/TuneLab/go-truss/gendoc"
	"github.com/TuneLab/go-truss/gendoc/doctree/makedt"
	"github.com/TuneLab/go-truss/gengokit"
)

func main() {
	noBuild := flag.Bool("nobuild", false, "Set -nobuild to generate code without building")
	flag.Parse()

	if len(flag.Args()) == 0 {
		exitIfError(errors.New("no arguments passed"))
	}

	rawDefinitionPaths := flag.Args()

	protoDir, definitionFiles, err := cleanProtofilePath(rawDefinitionPaths)

	var files []*os.File
	for _, f := range definitionFiles {
		protoF, err := os.Open(f)
		exitIfError(errors.Wrapf(err, "could not open %v", protoF))

		files = append(files, protoF)
	}

	// Check truss is running in $GOPATH
	goPath := os.Getenv("GOPATH")

	if !strings.HasPrefix(protoDir, goPath) {
		exitIfError(errors.New("truss envoked on files outside of $GOPATH"))
	}

	// Stage directories and files needed on disk
	err = protostage.Stage(protoDir)
	exitIfError(err)

	// Generate the .pb.go files containing the golang data structures
	// From `$GOPATH/src/org/user/thing` get `org/user/thing` for importing in golang
	goImportPath := strings.TrimPrefix(protoDir, goPath+"/src/")
	err = protostage.GeneratePBDataStructures(definitionFiles, protoDir, goImportPath)

	// Compose protocOut and service file to make a doctree
	protocOut, serviceFile, err := protostage.Compose(definitionFiles, protoDir)
	exitIfError(err)

	// Make a doctree
	dt, err := makedt.New(protocOut, serviceFile)
	exitIfError(err)

	prevGen, err := readPreviousGeneration(protoDir)
	exitIfError(err)

	// generate docs
	genDocFiles := gendoc.GenerateDocs(dt)

	// generate gokit microservice
	genFiles, err := gengokit.GenerateGokit(dt, prevGen, goImportPath)
	exitIfError(err)

	// append files together
	genFiles = append(genFiles, genDocFiles...)

	// Write files to disk
	for _, f := range genFiles {
		name := f.Name()

		mkdir(name)
		file, err := os.Create(name)
		exitIfError(errors.Wrapf(err, "could create file %v", name))

		_, err = io.Copy(file, f)
		exitIfError(errors.Wrapf(err, "could not write to %v", name))
	}

	if !*noBuild {
		err := buildMicroservice(goImportPath)
		exitIfError(errors.Wrap(err, "could not build microservice"))
	}
}

// cleanProtofilePath takes a slice of file paths and returns the
// absolute directory that contains the file paths, an array of the basename
// of the files, or an error if the files are not in the same directory
func cleanProtofilePath(rawPaths []string) (wd string, definitionFiles []string, err error) {
	execWd, err := os.Getwd()
	if err != nil {
		return "", nil, errors.Wrap(err, "could not get working directoru of truss")
	}

	var workingDirectory string

	// Parsed passed file paths
	for _, def := range rawPaths {
		// If the definition file path is not absolute, then make it absolute using trusses working directory
		if !path.IsAbs(def) {
			def = path.Clean(def)
			def = path.Join(execWd, def)
		}

		// The working direcotry for this definition file
		dir := path.Dir(def)
		// Add the base name of definition file to the slice
		definitionFiles = append(definitionFiles, path.Base(def))

		// If the working directory has not beenset before set it
		if workingDirectory == "" {
			workingDirectory = dir
		} else {
			// If the working directory for this definition file is different than the previous
			if workingDirectory != dir {
				return "", nil,
					errors.Errorf(
						"all .proto files must reside in the same directory\n"+
							"these two differ: \n%v\n%v",
						wd,
						workingDirectory)
			}
		}
	}

	return workingDirectory, definitionFiles, nil
}

// mkdir acts like $ mkdir -p path
func mkdir(path string) error {
	dir := filepath.Dir(path)

	// 0775 is the file mode that $ mkdir uses when creating a directoru
	err := os.MkdirAll(dir, 0775)

	return err
}

func exitIfError(err error) {
	if errors.Cause(err) != nil {
		defer os.Exit(1)
		fmt.Printf("%v\n", err)
	}
}

func buildMicroservice(goImportPath string) (err error) {
	const serverPath = "/service/DONOTEDIT/cmd/svc/..."
	const clientPath = "/service/DONOTEDIT/cmd/cliclient/..."

	// Build server and client
	errChan := make(chan error)

	go goBuild("server", goImportPath+serverPath, errChan)
	go goBuild("cliclient", goImportPath+clientPath, errChan)

	err = <-errChan
	if err != nil {
		return err
	}

	err = <-errChan
	if err != nil {
		return err
	}

	return nil
}

// goBuild calls the `$ go get ` to install dependenices
// and then calls `$ go build service/bin/$name $path`
// to put the iterating binaries in the correct place
func goBuild(name string, path string, errChan chan error) {

	// $ go get

	goGetExec := exec.Command(
		"go",
		"get",
		"-d",
		"-v",
		path,
	)

	goGetExec.Stderr = os.Stderr

	err := goGetExec.Run()

	if err != nil {
		errChan <- errors.Wrapf(err, "could not $ go get %v", path)
		return
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

	err = goBuildExec.Run()
	if err != nil {
		errChan <- errors.Wrapf(err, "could not $ go build %v", path)
		return
	}

	errChan <- nil
}

// readPreviousGeneration accepts the path to the directory where the inputed .proto files are stored, protoDir,
// it returns a []truss.NamedReadWriter for all files in the service/ dir in protoDir
func readPreviousGeneration(protoDir string) ([]truss.NamedReadWriter, error) {
	if fileExists(protoDir+"/service") != true {
		return nil, nil
	}

	var files []truss.NamedReadWriter
	sfs := simpleFileConstructor{
		protoDir: protoDir,
		files:    files,
	}
	err := filepath.Walk(protoDir+"/service", sfs.makeSimpleFile)
	if err != nil {
		return nil, errors.Wrapf(err, "could not fully walk directory %v/service", protoDir)
	}

	return sfs.files, nil
}

// simpleFileConstructor has the function makeSimpleFile which is of type filepath.WalkFunc
// This allows for filepath.Walk to be called with makeSimpleFile and build a truss.SimpleFile
// for all files in a direcotry
type simpleFileConstructor struct {
	protoDir string
	files    []truss.NamedReadWriter
}

// makeSimpleFile is of type filepath.WalkFunc
// makeSimpleFile constructs a truss.SimpleFile and stores it in SimpleFileConstructor.files
func (sfs *simpleFileConstructor) makeSimpleFile(path string, info os.FileInfo, err error) error {
	if info.IsDir() {
		return nil
	}

	byteContent, ioErr := ioutil.ReadFile(path)

	if ioErr != nil {
		return errors.Wrapf(ioErr, "could not read file: %v", path)
	}

	// name will be in the always start with "service/"
	// trim the prefix of the path to the proto files from the full path to the file
	name := strings.TrimPrefix(path, sfs.protoDir+"/")
	var file truss.SimpleFile
	file.Path = name
	file.Write(byteContent)

	sfs.files = append(sfs.files, &file)

	return nil
}

// fileExists checks if a file at the given path exists. Returns true if the
// file exists, and false if the file does not exist.
func fileExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}
