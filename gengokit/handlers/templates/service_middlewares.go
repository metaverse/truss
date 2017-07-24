package templates

const ServiceBase = `
package handlers

import (
	pb "{{.PBImportPath -}}"
)

func WrapService(in pb.{{.Service.Name}}Server) pb.{{.Service.Name}}Server {
	return in
}
`
