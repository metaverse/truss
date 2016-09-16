// Package gendoc is a truss plugin to generate markdown documentation for a
// protobuf definition file.
package gendoc

import (
	"github.com/TuneLab/go-truss/deftree"
	"github.com/TuneLab/go-truss/truss/truss"
)

// GenerateDocs accepts a deftree that represents an ast of a group of
// protofiles and returns a []truss.SimpleFile that represents a relative
// filestructure of generated docs
func GenerateDocs(dt deftree.Deftree) []truss.NamedReadWriter {
	response := ""

	microDef, ok := dt.(*deftree.MicroserviceDefinition)
	if ok {
		response = MdMicroserviceDefinition(microDef, 1)
	} else {
		response = "Error, could not cast Deftree to MicroserviceDefinition"
	}

	var file truss.SimpleFile

	file.Path = dt.GetName() + "-service/docs/docs.md"
	file.Write([]byte(response))

	var files []truss.NamedReadWriter
	files = append(files, &file)

	return files
}
