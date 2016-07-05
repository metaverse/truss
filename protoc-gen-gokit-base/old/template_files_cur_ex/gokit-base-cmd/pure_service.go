
package main

import (
	"{{.AbsoluteRelativeImportPath}}controller"
	"{{.AbsoluteRelativeImportPath}}pb"
)


type pure{{.Service.GetName}} struct {
	*controller.Controller
}

{{range $i := .Service.Methods}}
func (p pure{{.Service.GetName}}) {{$i.GetName}}(req *pb.{{$i.RequestType.GetName}}) (*pb.{{$i.ResponseType.GetName}}, error) {
	res, err := p.Controller.{{$i.GetName}}(req)
	if res == nil {
		res = &pb.{{$i.ResponseType.GetName}}{}
	}
	return res, err
}
{{end}}
