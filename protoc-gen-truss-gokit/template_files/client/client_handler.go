package clienthandler

import (
	"{{.GeneratedImport -}} /pb"
)


{{range $i := .Service.Methods}}
// {{$i.GetName}} implements Service.
func {{$i.GetName}}(in []string) (pb.{{$i.RequestType.GetName}}, error){
	_ = in
	request := pb.{{$i.RequestType.GetName}}{
		//
	}
	return request, nil
}
{{end}}
