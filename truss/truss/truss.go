// Package truss contains the relative file tree data structure that represents
// the paths and contents of generated files
package truss

// SimpleFile stores a file name and that file's content
// Name is a path relative to the directory containing the .proto files
// Name should start with "service/" for all generated and read in files
type SimpleFile struct {
	Name    *string
	Content *string
}
