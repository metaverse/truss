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

	"github.com/TuneLab/go-truss/truss/execprotoc"
	"github.com/TuneLab/go-truss/truss/parsepkgname"
	"github.com/TuneLab/go-truss/truss/truss"

	"github.com/TuneLab/go-truss/deftree"
	"github.com/TuneLab/go-truss/gendoc"
	"github.com/TuneLab/go-truss/gengokit"
	ggkconf "github.com/TuneLab/go-truss/gengokit/config"
	"github.com/TuneLab/go-truss/svcdef"
)

var (
	pbPackageFlag = flag.String("pbout", "", "The go package path where the protoc-gen-go .pb.go structs will be written.")
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTION]... [*.proto]...\n", filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}

	flag.Parse()

	if len(flag.Args()) == 0 {
		fmt.Fprintf(os.Stderr, "%s: missing .proto file(s)\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "Try '%s --help' for more information.\n", filepath.Base(os.Args[0]))
		os.Exit(1)
	}
}

func main() {
	cfg, err := parseInput()
	exitIfError(errors.Wrap(err, "cannot parse input"))

	dt, sd, err := parseServiceDefinition(cfg)
	exitIfError(errors.Wrap(err, "cannot parse input definition proto files"))

	genFiles, err := generateCode(cfg, dt, sd)
	exitIfError(errors.Wrap(err, "cannot generate service"))

	for _, f := range genFiles {
		err := writeGenFile(f, cfg.ServicePath())
		if err != nil {
			exitIfError(errors.Wrap(err, "cannot to write output"))
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
		return nil, errors.New("GOPATH not set")
	}
	cfg.GOPATH = goPaths[0]

	// DefPaths
	var err error
	rawDefinitionPaths := flag.Args()
	cfg.DefPaths, err = cleanProtofilePath(rawDefinitionPaths)
	if err != nil {
		return nil, errors.Wrap(err, "cannot parse input arguments")
	}

	// Service Path
	defFile, err := os.Open(cfg.DefPaths[0])
	if err != nil {
		return nil, errors.Wrapf(err, "Could not open package file %q", cfg.DefPaths[0])
	}
	svcName, err := parsepkgname.FromReader(defFile)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot parse package name from file %q", cfg.DefPaths[0])
	}
	svcFolderName := svcName + "-service"
	svcPath := filepath.Join(filepath.Dir(cfg.DefPaths[0]), svcFolderName)
	cfg.ServicePackage, err = filepath.Rel(filepath.Join(cfg.GOPATH, "src"), svcPath)
	if err != nil {
		return nil, errors.Wrap(err, "service path is not in GOPATH")
	}

	// PrevGen
	cfg.PrevGen, err = readPreviousGeneration(cfg.ServicePath())
	if err != nil {
		return nil, errors.Wrap(err, "cannot read previously generated files")
	}

	// PBGoPackage
	if *pbPackageFlag == "" {
		cfg.PBPackage = cfg.ServicePackage
	} else {
		cfg.PBPackage = *pbPackageFlag
		if !fileExists(
			filepath.Join(cfg.GOPATH, "src", cfg.PBPackage)) {
			return nil, errors.Errorf(".pb.go output package directory does not exist: %q", cfg.PBPackage)
		}
	}

	return &cfg, nil
}

// parseServiceDefinition returns a deftree which contains all necessary
// information for generating a truss service and its documentation.
func parseServiceDefinition(cfg *truss.Config) (deftree.Deftree, *svcdef.Svcdef, error) {
	svcPath := cfg.ServicePath()
	protoDefPaths := cfg.DefPaths
	// Create the ServicePath so the .pb.go files may be place within it
	if cfg.PrevGen == nil {
		err := os.Mkdir(svcPath, 0777)
		if err != nil {
			return nil, nil, errors.Wrap(err, "cannot create service directory")
		}
	}

	err := execprotoc.GeneratePBDotGo(cfg.DefPaths, svcPath, cfg.PBPath())
	if err != nil {
		return nil, nil, errors.Wrap(err, "cannot create .pb.go files")
	}

	// Open all .pb.go files and store in slice to be passed to svcdef.New()
	//var openFiles func([]string) ([]io.Reader, error)
	openFiles := func(paths []string) ([]io.Reader, error) {
		rv := []io.Reader{}
		for _, p := range paths {
			reader, err := os.Open(p)
			if err != nil {
				return nil, errors.Wrapf(err, "couldn't open file %q", p)
			}
			rv = append(rv, reader)
		}
		return rv, nil
	}
	// Get path names of .pb.go files
	pbgoPaths := []string{}
	for _, p := range protoDefPaths {
		base := filepath.Base(p)
		barename := strings.TrimSuffix(base, filepath.Ext(p))
		pbgp := filepath.Join(cfg.PBPath(), barename+".pb.go")
		pbgoPaths = append(pbgoPaths, pbgp)
	}
	pbgoFiles, err := openFiles(pbgoPaths)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Failed to open a .pb.go file")
	}
	pbFiles, err := openFiles(protoDefPaths)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Failed to open a .proto file")
	}

	// Create the svcdef
	sd, err := svcdef.New(pbgoFiles, pbFiles)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Failed to create svcdef")
	}

	// Create the Deftree
	protocOut, err := execprotoc.CodeGeneratorRequest(protoDefPaths)
	if err != nil {
		return nil, nil, errors.Wrap(err, "cannot parse input files with protoc")
	}

	svcFile, err := execprotoc.ServiceFile(protocOut, filepath.Dir(protoDefPaths[0]))
	if err != nil {
		return nil, nil, errors.Wrap(err, "cannot find service definition file")
	}

	dt, err := deftree.New(protocOut, svcFile)
	if err != nil {
		return nil, nil, errors.Wrap(err, "cannot to construct service definition")
	}

	return dt, sd, nil
}

// generateCode returns a []truss.NamedReadWriter that represents a gokit
// service with documentation
func generateCode(cfg *truss.Config, dt deftree.Deftree, sd *svcdef.Svcdef) ([]truss.NamedReadWriter, error) {
	conf := ggkconf.Config{
		PBPackage:     cfg.PBPackage,
		GoPackage:     cfg.ServicePackage,
		PreviousFiles: cfg.PrevGen,
	}

	genGokitFiles, err := gengokit.GenerateGokit(dt, sd, conf)
	if err != nil {
		return nil, errors.Wrap(err, "cannot generate gokit service")
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
	err := os.MkdirAll(filepath.Dir(fullPath), 0777)
	if err != nil {
		return err
	}

	file, err := os.Create(fullPath)
	if err != nil {
		return errors.Wrapf(err, "cannot create file %v", fullPath)
	}

	_, err = io.Copy(file, f)
	if err != nil {
		return errors.Wrapf(err, "cannot write to %v", fullPath)
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
			return nil, errors.Wrap(err, "cannot get working directory of truss")
		}

		fullPaths = append(fullPaths, full)

		if filepath.Dir(fullPaths[0]) != filepath.Dir(full) {
			return nil, errors.Errorf("passed .proto files in different directories")
		}
	}

	return fullPaths, nil
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
	dir, _ := filepath.Split(serviceDir)
	sfs := simpleFileConstructor{
		dir:   dir,
		files: files,
	}
	err := filepath.Walk(serviceDir, sfs.makeSimpleFile)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot fully walk directory %v", sfs.dir)
	}

	return sfs.files, nil
}

// simpleFileConstructor has function makeSimpleFile of type filepath.WalkFunc
// This allows for filepath.Walk to be called with makeSimpleFile and build a truss.SimpleFile
// for all files in a direcotry
type simpleFileConstructor struct {
	dir   string
	files []truss.NamedReadWriter
}

// makeSimpleFile is of type filepath.WalkFunc
// makeSimpleFile constructs a truss.SimpleFile and stores it in SimpleFileConstructor.files
func (sfs *simpleFileConstructor) makeSimpleFile(path string, info os.FileInfo, err error) error {
	if info.IsDir() {
		return nil
	}

	byteContent, ioErr := ioutil.ReadFile(path)

	if ioErr != nil {
		return errors.Wrapf(ioErr, "cannot read file: %v", path)
	}

	// trim the prefix of the path to the proto files from the full path to the file
	name := strings.TrimPrefix(path, sfs.dir)
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
