// Package truss contains the relative file tree data structure that represents
// the paths and contents of generated files
package truss

import (
	"bytes"
	"io"
)

// NamedReadWriter represents a file name and that file's content
// Name() is a path relative to the directory containing the .proto files
// Name() should start with "service/" for all generated and read in files
// os.file fulfills the NamedReadWriter interface
type NamedReadWriter interface {
	io.ReadWriter
	Name() string
}

// SimpleFile pointer implements the NamedReadWriter interface
type SimpleFile struct {
	bytes.Buffer
	Path string
}

func (s *SimpleFile) Name() string {
	return s.Path
}
