//go:generate go-bindata -o template.go -pkg template -ignore swp template_files/...

/*
	This file is here to hold the `go generate` command above.

	The command uses go-bindata to generate binary data from the template files
	stored in ./template_files. This binary date is stored in template.go
	which is then compiled into the protoc-gen-truss-gokit binary.
*/
package template
