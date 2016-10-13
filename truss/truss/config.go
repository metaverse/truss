package truss

import (
	"path/filepath"
)

// Config defines the inputs to a truss service generation
type Config struct {
	// The first path in $GOPATH
	GOPATH string

	// The go packge where .pb.go files protoc-gen-go creates will be written
	PBPackage string
	// The go package where the service code will be written
	ServicePackage string

	// The paths to each of the .proto files truss is being run against
	DefPaths []string
	// The files of a previously generated service, may be nil
	PrevGen []NamedReadWriter
}

// ServicePath returns the full path to Config.ServicePackage
func (c *Config) ServicePath() string {
	goSvcPath := filepath.Join(c.GOPATH, "src", c.ServicePackage)

	return goSvcPath
}

// PBPath returns the full paht to Config.PBPackage
func (c *Config) PBPath() string {
	pbPath := filepath.Join(c.GOPATH, "src", c.PBPackage)

	return pbPath
}
