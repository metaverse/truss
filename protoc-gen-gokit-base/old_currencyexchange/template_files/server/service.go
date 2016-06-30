package server

import (
	"{{.AbsoluteRelativeImportPath}}pb"
)

type {{.Service.GetName}} interface {
	{{range $i := .Service.Methods}}
	{{$i.GetName}}(req *pb.{{$i.RequestType.GetName}}) (*pb.{{$i.ResponseType.GetName}}, error)
	{{- end}}
}
