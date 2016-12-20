// Package gendoc is a truss plugin to generate markdown documentation for a
// protobuf definition file.
package gendoc

import (
	"io"
	"strings"

	"github.com/TuneLab/go-truss/deftree"
)

func findServiceName(md *deftree.MicroserviceDefinition) string {
	rv := "default"
	for _, f := range md.Files {
		for _, svc := range f.Services {
			rv = svc.GetName()
		}
	}
	return rv

}

// GenerateDocs accepts a deftree that represents an ast of a group of
// protofiles and returns map[string]io.Reader that represents a relative
// filestructure of generated docs
func GenerateDocs(dt deftree.Deftree) map[string]io.Reader {
	response := ""

	microDef, ok := dt.(*deftree.MicroserviceDefinition)
	if ok {
		response = MdMicroserviceDefinition(microDef, 1)
	} else {
		response = "Error, could not cast Deftree to MicroserviceDefinition"
	}

	files := make(map[string]io.Reader)
	md := dt.(*deftree.MicroserviceDefinition)
	svcname := strings.ToLower(findServiceName(md))
	files[svcname+"-service/docs/docs.md"] = strings.NewReader(response)

	return files
}
