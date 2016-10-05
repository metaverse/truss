package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/TuneLab/go-truss/truss/truss"
	"github.com/TuneLab/go-truss/truss/protostuff"

	"github.com/TuneLab/go-truss/deftree"
	"github.com/TuneLab/go-truss/gendoc"
	"github.com/TuneLab/go-truss/gengokit"
)

func main() {
	flag.Parse()

	if len(flag.Args()) == 0 {
		exitIfError(errors.New("no arguments passed"))
	}

	rawDefinitionPaths := flag.Args()

	protoDir, definitionFiles, err := cleanProtofilePath(rawDefinitionPaths)

	// Check truss is running in $GOPATH
	goPath := os.Getenv("GOPATH")

	if !strings.HasPrefix(protoDir, goPath) {
		exitIfError(errors.New("truss envoked on files outside of $GOPATH"))
	}

	protocOut, err := protostuff.CodeGeneratorRequest(definitionFiles, protoDir)
	exitIfError(err)

	svcFile, err := protostuff.ServiceFile(protocOut, protoDir)
	exitIfError(err)

	// Make a deftree
	dt, err := deftree.New(protocOut, svcFile)
	exitIfError(err)

	// Generate the .pb.go files containing the golang data structures
	// From `$GOPATH/src/org/user/thing` get `org/user/thing` for importing in golang
	svcName := dt.GetName() + "-service"
	svcDir := filepath.Join(protoDir, svcName)
	err = mkdir(svcDir)
	exitIfError(err)

	err = protostuff.GeneratePBataStructures(definitionFiles, svcDir)
	exitIfError(err)

	prevGen, err := readPreviousGeneration(protoDir, svcDir)
	exitIfError(err)

	// generate docs
	genDocFiles := gendoc.GenerateDocs(dt)

	// generate gokit microservice
	goSvcImportPath, err := filepath.Rel(filepath.Join(goPath, "src"), svcDir)
	exitIfError(err)
	genFiles, err := gengokit.GenerateGokit(dt, prevGen, goSvcImportPath)
	exitIfError(err)

	// append files together
	genFiles = append(genFiles, genDocFiles...)

	// Write files to disk
	for _, f := range genFiles {
		name := f.Name()

		fullPath := filepath.Join(protoDir, name)
		err := mkdir(fullPath)
		exitIfError(err)

		file, err := os.Create(fullPath)
		exitIfError(errors.Wrapf(err, "could create file %v", fullPath))

		_, err = io.Copy(file, f)
		exitIfError(errors.Wrapf(err, "could not write to %v", fullPath))
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

// readPreviousGeneration accepts the path to the directory where the inputed .proto files are stored, protoDir,
// it returns a []truss.NamedReadWriter for all files in the service/ dir in protoDir
func readPreviousGeneration(protoDir, serviceDir string) ([]truss.NamedReadWriter, error) {
	if fileExists(serviceDir) != true {
		return nil, nil
	}

	var files []truss.NamedReadWriter
	sfs := simpleFileConstructor{
		protoDir: protoDir,
		files:    files,
	}
	err := filepath.Walk(serviceDir, sfs.makeSimpleFile)
	if err != nil {
		return nil, errors.Wrapf(err, "could not fully walk directory %v", protoDir)
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
