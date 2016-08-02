package main

import (
	"flag"
	//"fmt"
	"os"
	"path"
	"strings"

	"github.com/TuneLab/gob/truss/generator"

	log "github.com/Sirupsen/logrus"
)

func init() {
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{
		ForceColors: true,
	})
}

// Stages are documented in README.md
func main() {
	flag.Parse()

	if len(flag.Args()) == 0 {
		log.Fatal("No proto files passed")
		os.Exit(1)
	}

	rawDefinitionPaths := flag.Args()

	execWd, err := os.Getwd()
	if err != nil {
		log.WithError(err).Fatal("Cannot get working directory")
	}

	var workingDirectory string
	var definitionFiles []string

	// Parsed passed file paths
	for _, def := range rawDefinitionPaths {
		// If the definition file path is not absolute, then make it absolute using trusses working directory
		if !path.IsAbs(def) {
			def = path.Clean(def)
			def = path.Join(execWd, def)
		}

		// The working direcotry for this definition file
		wd := path.Dir(def)
		// Add the base name of definition file to the slice
		definitionFiles = append(definitionFiles, path.Base(def))

		// If the working directory has not beenset before set it
		if workingDirectory == "" {
			workingDirectory = wd
		} else {
			// If the working directory for this definition file is different than the previous
			if wd != workingDirectory {
				log.Fatal("Passed protofiles reside in different directories")
			}
		}
	}

	goPath := os.Getenv("GOPATH")

	if !strings.HasPrefix(workingDirectory, goPath) {
		log.Fatal("truss envoked from outside of $GOPATH")
	}

	// From `$GOPATH/src/org/user/thing` get `org/user/thing` for importing in golang
	genImportPath := strings.TrimPrefix(workingDirectory, goPath+"/src/")

	generator.GenerateMicroservice(genImportPath, workingDirectory, definitionFiles)
}
