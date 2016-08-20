//go:generate ./generate-test-data.sh
//go:generate cp -r ./definitions ./data
//go:generate go-bindata -pkg testproto -o testprotodata.go -prefix data/ -ignore swp data/...

/*
	This file is here to hold the `go generate` command above.

	The command uses go-bindata to generate binary data from the test data
	stored in ./data/definitions/. This binary date is stored in testdata.go
	which is then used in various truss unit tests
*/
package testprotodata
