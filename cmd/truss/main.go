package main

import (
	"flag"
	"fmt"
	"go/build"
	"io"
	"os"
	"path/filepath"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/pkg/errors"

	"github.com/TuneLab/go-truss/truss"
	"github.com/TuneLab/go-truss/truss/execprotoc"
	"github.com/TuneLab/go-truss/truss/parsepkgname"

	"github.com/TuneLab/go-truss/deftree"
	"github.com/TuneLab/go-truss/gendoc"
	ggkconf "github.com/TuneLab/go-truss/gengokit"
	gengokit "github.com/TuneLab/go-truss/gengokit/generator"
	"github.com/TuneLab/go-truss/svcdef"
)

var (
	pbPackageFlag  = flag.String("pbout", "", "The go package path where the protoc-gen-go .pb.go structs will be written.")
	svcPackageFlag = flag.String("svcout", "", "The go package path where the generated service directory will be written.")
)

func init() {
	log.SetLevel(log.InfoLevel)

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

	for path, file := range genFiles {
		err := writeGenFile(file, path, cfg.ServicePath)
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
	cfg.GoPath = filepath.SplitList(os.Getenv("GOPATH"))
	if len(cfg.GoPath) == 0 {
		return nil, errors.New("GOPATH not set")
	}
	log.WithField("GOPATH", cfg.GoPath).Debug()

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
		return nil, errors.Wrapf(err, "cannot open package file %q", cfg.DefPaths[0])
	}
	svcName, err := parsepkgname.FromReader(defFile)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot parse package name from file %q", cfg.DefPaths[0])
	}
	svcFolderName := svcName + "-service"

	if *svcPackageFlag == "" {
		svcPath := filepath.Join(filepath.Dir(cfg.DefPaths[0]), svcFolderName)
		p, err := build.Default.ImportDir(svcPath, build.FindOnly)
		if err != nil {
			return nil, err
		}
		if p.Root == "" {
			return nil, errors.New("proto files path not in GOPATH")
		}

		cfg.ServicePackage = p.ImportPath
		cfg.ServicePath = p.Dir
	} else {
		baseSVCPackage := *svcPackageFlag
		p, err := build.Default.Import(baseSVCPackage, "", build.FindOnly)
		if err != nil {
			return nil, err
		}
		if p.Root == "" {
			return nil, errors.New("svcout not in gopath GOPATH")
		}
		if !fileExists(p.Dir) {
			return nil, errors.Errorf("specified package path for service output directory does not exist: %q", p.Dir)
		}
		cfg.ServicePackage = filepath.Join(baseSVCPackage, svcFolderName)
		cfg.ServicePath = filepath.Join(p.Dir, svcFolderName)
	}
	log.WithField("Service Package", cfg.ServicePackage).Debug()
	log.WithField("Service Path", cfg.ServicePath).Debug()

	// PrevGen
	cfg.PrevGen, err = readPreviousGeneration(cfg.ServicePath)
	if err != nil {
		return nil, errors.Wrap(err, "cannot read previously generated files")
	}

	// PBGoPackage
	if *pbPackageFlag == "" {
		cfg.PBPackage = cfg.ServicePackage
		cfg.PBPath = cfg.ServicePath
	} else {
		cfg.PBPackage = *pbPackageFlag
		p, err := build.Default.Import(cfg.PBPackage, "", build.FindOnly)
		if err != nil {
			return nil, err
		}
		if !fileExists(p.Dir) {
			return nil, errors.Errorf("specified package path for .pb.go output directory does not exist: %q", p.Dir)
		}
		cfg.PBPath = p.Dir
	}
	log.WithField("PB Package", cfg.PBPackage).Debug()
	log.WithField("PB Path", cfg.PBPath).Debug()

	return &cfg, nil
}

// parseServiceDefinition returns a deftree which contains all necessary
// information for generating a truss service and its documentation.
func parseServiceDefinition(cfg *truss.Config) (deftree.Deftree, *svcdef.Svcdef, error) {
	protoDefPaths := cfg.DefPaths
	// Create the ServicePath so the .pb.go files may be place within it
	if cfg.PrevGen == nil {
		err := os.MkdirAll(cfg.ServicePath, 0777)
		if err != nil {
			return nil, nil, errors.Wrap(err, "cannot create service directory")
		}
	}

	err := execprotoc.GeneratePBDotGo(cfg.DefPaths, cfg.GoPath, cfg.PBPath)
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
				return nil, errors.Wrapf(err, "cannot open file %q", p)
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
		pbgp := filepath.Join(cfg.PBPath, barename+".pb.go")
		pbgoPaths = append(pbgoPaths, pbgp)
	}
	pbgoFiles, err := openFiles(pbgoPaths)
	if err != nil {
		return nil, nil, errors.Wrap(err, "cannot open all .pb.go files")
	}
	pbFiles, err := openFiles(protoDefPaths)
	if err != nil {
		return nil, nil, errors.Wrap(err, "cannot open all .proto files")
	}

	// Create the svcdef
	sd, err := svcdef.New(pbgoFiles, pbFiles)
	if err != nil {
		return nil, nil, errors.Wrap(err, "cannot create svcdef")
	}

	// Create the Deftree
	protocOut, err := execprotoc.CodeGeneratorRequest(protoDefPaths, cfg.GoPath)
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

// generateCode returns a map[string]io.Reader that represents a gokit
// service with documentation
func generateCode(cfg *truss.Config, dt deftree.Deftree, sd *svcdef.Svcdef) (map[string]io.Reader, error) {
	conf := ggkconf.Config{
		PBPackage:     cfg.PBPackage,
		GoPackage:     cfg.ServicePackage,
		PreviousFiles: cfg.PrevGen,
	}

	genGokitFiles, err := gengokit.GenerateGokit(sd, conf)
	if err != nil {
		return nil, errors.Wrap(err, "cannot generate gokit service")
	}

	genDocFiles := gendoc.GenerateDocs(dt)

	return combineFiles(genGokitFiles, genDocFiles), nil
}

// combineFiles takes any numbers of map[string]io.Reader and combines them into one.
func combineFiles(group ...map[string]io.Reader) map[string]io.Reader {
	final := make(map[string]io.Reader)
	for _, g := range group {
		for path, file := range g {
			if final[path] != nil {
				log.WithField("path", path).
					Warn("truss generated two files with same path, outputting final one specified")
			}
			final[path] = file
		}
	}

	return final
}

// writeGenFile writes a file at relPath relative to serviceDir to the filesystem
func writeGenFile(file io.Reader, relPath, serviceDir string) error {
	// the serviceDir contains /NAME-service so we want to write to the
	// directory above
	outDir := filepath.Dir(serviceDir)

	// i.e. NAME-service/generated/endpoint.go

	fullPath := filepath.Join(outDir, relPath)
	err := os.MkdirAll(filepath.Dir(fullPath), 0777)
	if err != nil {
		return err
	}

	outFile, err := os.Create(fullPath)
	if err != nil {
		return errors.Wrapf(err, "cannot create file %v", fullPath)
	}

	_, err = io.Copy(outFile, file)
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

// readPreviousGeneration returns a map[string]io.Reader representing the files in serviceDir
func readPreviousGeneration(serviceDir string) (map[string]io.Reader, error) {
	if fileExists(serviceDir) != true {
		return nil, nil
	}

	dir, _ := filepath.Split(serviceDir)
	sfs := simpleFileConstructor{
		dir:   dir,
		files: make(map[string]io.Reader),
	}
	err := filepath.Walk(serviceDir, sfs.makeSimpleFile)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot fully walk directory %v", sfs.dir)
	}

	return sfs.files, nil
}

// simpleFileConstructor has function makeSimpleFile of type filepath.WalkFunc
// This allows for filepath.Walk to be called with makeSimpleFile and build a
// map[string]io.Reader for all files in a directory
type simpleFileConstructor struct {
	dir   string
	files map[string]io.Reader
}

// makeSimpleFile is of type filepath.WalkFunc
// makeSimpleFile adds files as io.Readers to sfs.files by path
func (sfs *simpleFileConstructor) makeSimpleFile(path string, info os.FileInfo, err error) error {
	if info.IsDir() {
		return nil
	}

	file, ioErr := os.Open(path)

	if ioErr != nil {
		return errors.Wrapf(ioErr, "cannot read file: %v", path)
	}

	// trim the prefix of the path to the proto files from the full path to the file
	name := strings.TrimPrefix(path, sfs.dir)

	sfs.files[name] = file

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
