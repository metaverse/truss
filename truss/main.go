package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/TuneLab/go-truss/truss/protostuff"
	"github.com/TuneLab/go-truss/truss/truss"

	"github.com/TuneLab/go-truss/deftree"
	"github.com/TuneLab/go-truss/gendoc"
	"github.com/TuneLab/go-truss/gengokit"
)

var (
	pbPathFlag = flag.String("pbout", "", "The go package path where the protoc-gen-go .pb.go structs will be written.")
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTION]... [*.proto]...\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	if len(flag.Args()) == 0 {
		fmt.Fprintf(os.Stderr, "%s: missing .proto file(s)\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Try '%s --help' for more information.\n", os.Args[0])
		os.Exit(1)
	}
}

func main() {
	cfg, err := parseInput()
	exitIfError(errors.Wrap(err, "could not parse input"))

	dt, err := parseServiceDefinition(cfg.DefPaths)
	exitIfError(errors.Wrap(err, "unable to parse input definition proto files"))

	err = updateConfigWithService(cfg, dt)
	exitIfError(errors.Wrap(err, "unable to read system based on service definition"))

	genFiles, err := generateCode(cfg, dt)
	exitIfError(errors.Wrap(err, "unable to generate service"))

	for _, f := range genFiles {
		err := writeGenFile(f, cfg.ServicePath)
		if err != nil {
			exitIfError(errors.Wrap(err, "unable to write output"))
		}
	}
}

// parseInput constructs a *truss.Config with all values needed to parse
// service definition files.
func parseInput() (*truss.Config, error) {
	var cfg truss.Config

	// GOPATH
	goPaths := filepath.SplitList(os.Getenv("GOPATH"))
	if goPaths == nil {
		return nil, errors.New("$GOPATH not set")
	}
	cfg.GOPATH = goPaths[0]

	// DefPaths
	var err error
	rawDefinitionPaths := flag.Args()
	cfg.DefPaths, err = cleanProtofilePath(rawDefinitionPaths)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse input arguments")
	}

	// PBGoPath
	if *pbPathFlag != "" {
		cfg.PBGoPath = filepath.Join(cfg.GOPATH, "src", *pbPathFlag)
		if !fileExists(cfg.PBGoPath) {
			return nil, errors.Errorf(".pb.go output package directory does not exist: %q", cfg.PBGoPath)
		}
	}

	return &cfg, nil
}

// parseServiceDefinition returns a deftree which contains all needed for all
// generating a truss service and documentation
func parseServiceDefinition(definitionPaths []string) (deftree.Deftree, error) {
	protocOut, err := protostuff.CodeGeneratorRequest(definitionPaths)
	if err != nil {
		return nil, errors.Wrap(err, "unable to use parse input files with protoc")
	}

	svcFile, err := protostuff.ServiceFile(protocOut, filepath.Dir(definitionPaths[0]))
	if err != nil {
		return nil, errors.Wrap(err, "unable to find service definition file")
	}

	dt, err := deftree.New(protocOut, svcFile)
	if err != nil {
		return nil, errors.Wrap(err, "unable to construct service defintion")
	}

	return dt, nil
}

// updateConfigWithService updates the config with all information needed to
// generate a truss service using the parsedServiceDefinition deftree
func updateConfigWithService(cfg *truss.Config, dt deftree.Deftree) error {
	// Service Path
	svcName := dt.GetName() + "-service"
	cfg.ServicePath = filepath.Join(filepath.Dir(cfg.DefPaths[0]), svcName)

	// PrevGen
	var err error
	cfg.PrevGen, err = readPreviousGeneration(cfg.ServicePath)
	if err != nil {
		return errors.Wrap(err, "unable to read previously generated files")
	}

	// PBGoPath
	if cfg.PBGoPath == "" {
		cfg.PBGoPath = cfg.ServicePath
	}
	fmt.Println(cfg.PBGoPath)

	return nil
}

// generateCode returns a []truss.NamedReadWriter that represents a gokit
// service with documentation
func generateCode(cfg *truss.Config, dt deftree.Deftree) ([]truss.NamedReadWriter, error) {
	if cfg.PrevGen == nil {
		err := mkdir(cfg.ServicePath)
		if err != nil {
			return nil, errors.Wrap(err, "unable to create service directory")
		}
	}

	err := protostuff.GeneratePBDotGo(cfg.DefPaths, cfg.ServicePath, cfg.PBGoPath)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create .pb.go files")
	}

	genGokitFiles, err := gengokit.GenerateGokit(dt, cfg.PrevGen, cfg.GoSvcImportPath(), cfg.GoPBImportPath())
	if err != nil {
		return nil, errors.Wrap(err, "unable to generate gokit service")
	}

	genDocFiles := gendoc.GenerateDocs(dt)

	genFiles := append(genGokitFiles, genDocFiles...)

	return genFiles, nil
}

// writeGenFile writes a truss.NamedReadWriter to the filesystem
// to be contained within serviceDir
func writeGenFile(f truss.NamedReadWriter, serviceDir string) error {
	// the serviceDir contains /NAME-service so we want to write to the
	// directory above
	outDir := filepath.Dir(serviceDir)

	// i.e. NAME-service/generated/endpoint.go
	name := f.Name()

	fullPath := filepath.Join(outDir, name)
	err := mkdir(fullPath)
	if err != nil {
		return err
	}

	file, err := os.Create(fullPath)
	if err != nil {
		return errors.Wrapf(err, "could create file %v", fullPath)
	}

	_, err = io.Copy(file, f)
	if err != nil {
		return errors.Wrapf(err, "could not write to %v", fullPath)
	}
	return nil
}

// cleanProtofilePath returns the absolute filepath of a group of files
// of the files, or an error if the files are not in the same directory
func cleanProtofilePath(rawPaths []string) ([]string, error) {
	var fullPaths []string

	// Parsed passed file paths
	for _, def := range rawPaths {
		full, err := filepath.Abs(def)
		if err != nil {
			return nil, errors.Wrap(err, "could not get working directoru of truss")
		}

		fullPaths = append(fullPaths, full)

		if filepath.Dir(fullPaths[0]) != filepath.Dir(full) {
			return nil, errors.Errorf("passed .proto files in different directories")
		}
	}

	return fullPaths, nil
}

// mkdir acts like $ mkdir -p path
func mkdir(path string) error {
	dir := filepath.Dir(path)

	// 0775 is the file mode that $ mkdir uses when creating a directoru
	err := os.MkdirAll(dir, 0775)

	return err
}

// exitIfError will print the error message and exit 1 if the passed error is
// non-nil
func exitIfError(err error) {
	if errors.Cause(err) != nil {
		defer os.Exit(1)
		fmt.Printf("%v\n", err)
	}
}

// readPreviousGeneration returns a []truss.NamedReadWriter for all files serviceDir
func readPreviousGeneration(serviceDir string) ([]truss.NamedReadWriter, error) {
	if fileExists(serviceDir) != true {
		return nil, nil
	}

	var files []truss.NamedReadWriter
	sfs := simpleFileConstructor{
		dirPath: filepath.Dir(serviceDir),
		files:   files,
	}
	err := filepath.Walk(serviceDir, sfs.makeSimpleFile)
	if err != nil {
		return nil, errors.Wrapf(err, "could not fully walk directory %v", sfs.dirPath)
	}

	return sfs.files, nil
}

// simpleFileConstructor has function makeSimpleFile of type filepath.WalkFunc
// This allows for filepath.Walk to be called with makeSimpleFile and build a truss.SimpleFile
// for all files in a direcotry
type simpleFileConstructor struct {
	dirPath string
	files   []truss.NamedReadWriter
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
	name := strings.TrimPrefix(path, sfs.dirPath+"/")
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
