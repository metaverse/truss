package templates

const ServiceBase = `
package middlewares

import (
	pb "{{.PBImportPath -}}"
)

func InjectServiceMiddlewares(in pb.{{.Service.Name}}Server) pb.{{.Service.Name}}Server {
	return in
}
`
