package handler

// This file contains the Service definition, and a basic service
// implementation. It also includes service middlewares.

import (
	_ "errors"
	_ "time"

	"golang.org/x/net/context"

	_ "github.com/go-kit/kit/log"
	_ "github.com/go-kit/kit/metrics"

	"{{.GeneratedImport -}} /pb"
)



// NewBasicService returns a naïve, stateless implementation of Service.
func NewBasicService() Service {
	return basicService{}
}

type basicService struct{}

{{range $i := .Service.Methods}}
// {{$i.GetName}} implements Service.
func (s basicService) {{$i.GetName}}(ctx context.Context, in pb.{{$i.RequestType.GetName}}) (pb.{{$i.ResponseType.GetName}}, error){
	_ = ctx
	_ = in
	return pb.{{$i.ResponseType.GetName}}{}, nil
}
{{end}}

type Service interface {
{{range $i := .Service.Methods}}
	{{$i.GetName}}(ctx context.Context, in pb.{{$i.RequestType.GetName}}) (pb.{{$i.ResponseType.GetName}}, error)
{{- end}}
}
