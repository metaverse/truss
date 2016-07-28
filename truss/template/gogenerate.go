//go:generate go-bindata -pkg template -o template.go -ignore swp service/...

/*
	This file is here to hold the `go generate` command above.

	The command uses go-bindata to generate binary data from the template files
	stored in ./service. This binary date is stored in template.go
	which is then compiled into the truss binary
*/
package template
