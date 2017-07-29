package templates

const bbb2306ServiceMiddlewares = `
package handlers

import (
	pb "{{.PBImportPath -}}"
)

func WrapService(in pb.{{.Service.Name}}Server) pb.{{.Service.Name}}Server {
	return in
}
`

var ServiceMiddlewares = map[string]map[string]string{
	"bbb2306": {
		"ServiceMiddlewares": bbb2306ServiceMiddlewares,
	},
}
