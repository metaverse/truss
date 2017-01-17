package main

import (
	"fmt"
	"go/build"
	"io"
	"os"
	"path/filepath"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"

	"github.com/TuneLab/go-truss/truss"
	"github.com/TuneLab/go-truss/truss/execprotoc"
	"github.com/TuneLab/go-truss/truss/parsesvcname"

	"github.com/TuneLab/go-truss/deftree"
	"github.com/TuneLab/go-truss/gendoc"
	ggkconf "github.com/TuneLab/go-truss/gengokit"
	gengokit "github.com/TuneLab/go-truss/gengokit/generator"
	"github.com/TuneLab/go-truss/svcdef"
)

var (
	pbPackageFlag  = flag.String("pbout", "", "Go package path where the protoc-gen-go .pb.go files will be written")
	svcPackageFlag = flag.String("svcout", "", "Go package path where the generated Go service will be written")
	verboseFlag    = flag.BoolP("verbose", "v", false, "verbose output")
)

var binName = filepath.Base(os.Args[0])

const (
	noVersion   string = "<no-version>"
	noBuildDate string = "<no-build-date>"
)

var (
	Version   string = noVersion
	BuildDate string = noBuildDate
)

func init() {
	var buildinfo string
	if Version != "" && Version != noVersion {
		buildinfo = fmt.Sprintf("version: %s", Version)
	}
	if BuildDate != "" && BuildDate != noBuildDate {
		buildinfo = fmt.Sprintf("%s built: %s", buildinfo, strings.Replace(BuildDate, "_", " ", -1))
	}

	flag.Usage = func() {
		if buildinfo != "" {
			fmt.Fprintf(os.Stderr, "%s (%s)\n", binName, strings.TrimSpace(buildinfo))
		}
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <protofile>...\n", binName)
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()

	if len(flag.Args()) == 0 {
		fmt.Fprintf(os.Stderr, "%s: missing .proto file(s)\n", binName)
		flag.Usage()
		os.Exit(1)
	}

	log.SetLevel(log.InfoLevel)
	if *verboseFlag {
		log.SetLevel(log.DebugLevel)
	}

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
	wd, err := os.Getwd()
	if err != nil {
		log.Warn(errors.Wrap(err, "cannot get working directory"))
		log.Warn("Flags will only work with non relative directories")
		wd = ""
	}

	// GOPATH
	cfg.GoPath = filepath.SplitList(os.Getenv("GOPATH"))
	if len(cfg.GoPath) == 0 {
		return nil, errors.New("GOPATH not set")
	}
	log.WithField("GOPATH", cfg.GoPath).Debug()

	// DefPaths
	rawDefinitionPaths := flag.Args()
	cfg.DefPaths, err = cleanProtofilePath(rawDefinitionPaths)
	if err != nil {
		return nil, errors.Wrap(err, "cannot parse input arguments")
	}

	// Service Path
	svcName, err := parsesvcname.FromPaths(cfg.GoPath, cfg.DefPaths)
	svcName = strings.ToLower(svcName)
	if err != nil {
		return nil, errors.Wrap(err, "cannot parse service name from the provided definition files")
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
		p, err := build.Default.Import(baseSVCPackage, wd, build.FindOnly)
		if err != nil {
			return nil, err
		}
		if p.Root == "" {
			return nil, errors.New("svcout not in GOPATH")
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
		p, err := build.Default.Import(cfg.PBPackage, wd, build.FindOnly)
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
		return nil, nil, errors.Wrapf(err, "failed to create service definition; did you pass ALL the protobuf files to truss?")
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

// combineFiles takes any number of map[string]io.Reader and combines them into one.
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
		if *verboseFlag {
			fmt.Printf("%+v\n", err)
			return
		}
		fmt.Printf("%v\n", err)
	}
}

// readPreviousGeneration returns a map[string]io.Reader representing the files in serviceDir
func readPreviousGeneration(serviceDir string) (map[string]io.Reader, error) {
	if fileExists(serviceDir) != true {
		return nil, nil
	}

	dir, _ := filepath.Split(serviceDir)
	files := make(map[string]io.Reader)

	addFileToFiles := func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		file, ioErr := os.Open(path)

		if ioErr != nil {
			return errors.Wrapf(ioErr, "cannot read file: %v", path)
		}

		// trim the prefix of the path to the proto files from the full path to the file
		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		files[relPath] = file

		return nil
	}

	err := filepath.Walk(serviceDir, addFileToFiles)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot fully walk directory %v", dir)
	}

	return files, nil
}

// fileExists checks if a file at the given path exists. Returns true if the
// file exists, and false if the file does not exist.
func fileExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}
