package truss

import (
	"path/filepath"
)

// Config defines the inputs to a truss service generation
type Config struct {
	// The first path in $GOPATH
	GOPATH string

	// The path to where .pb.go files protoc-gen-go creates will be written
	PBGoPath string
	// The path to the NAME-service directory; third_party will be written
	ServicePath string

	// The paths to each of the .proto files truss is being run against
	DefPaths []string
	// The files of a previously generated service, may be nil
	PrevGen []NamedReadWriter
}

// GoSvcImportPath returns a go package import string for the Config.ServicePath
func (c *Config) GoSvcImportPath() string {
	goSvcImportPath, err := filepath.Rel(filepath.Join(c.GOPATH, "src"), c.ServicePath)
	if err != nil {
		return ""
	}

	return goSvcImportPath
}

// GoPBImportPath returns a go package import string for the Config.PBGoPath
func (c *Config) GoPBImportPath() string {
	goPBImportPath, err := filepath.Rel(filepath.Join(c.GOPATH, "src"), c.PBGoPath)
	if err != nil {
		return ""
	}

	return goPBImportPath
}
