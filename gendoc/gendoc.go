// Package gendoc is a truss plugin
// to generate markdown documentation for a protobuf definition file.
package gendoc

import (
	"github.com/TuneLab/gob/gendoc/doctree"
	"github.com/TuneLab/gob/truss/truss"
)

// GenerateDocs accepts a doctree that represents an ast of a group of
// protofiles and returns a []truss.SimpleFile that represents a relative
// filestructure of generated docs
func GenerateDocs(dt doctree.Doctree) []truss.NamedReadWriter {
	response := dt.Markdown()

	var file truss.SimpleFile

	file.Path = "service/docs/docs.md"
	file.Write([]byte(response))

	var files []truss.NamedReadWriter
	files = append(files, &file)

	return files
}
