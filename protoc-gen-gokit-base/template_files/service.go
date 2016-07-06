package addsvc

// This file contains the Service definition, and a basic service
// implementation. It also includes service middlewares.

import (
	_ "errors"
	_ "time"

	"golang.org/x/net/context"

	_ "github.com/go-kit/kit/log"
	_ "github.com/go-kit/kit/metrics"

	"{{.AbsoluteRelativeImportPath -}} /pb"
)

// Service describes a service that adds things together.
type Service interface {
{{range $i := .Service.Methods}}
	{{$i.GetName}}(ctx context.Context, in pb.{{$i.RequestType.GetName}}) (pb.{{$i.ResponseType.GetName}}, error)
{{- end}}
}


// NewBasicService returns a naïve, stateless implementation of Service.
func NewBasicService() Service {
	return basicService{}
}

type basicService struct{}

// Sum implements Service.
{{range $i := .Service.Methods}}
func (s basicService) {{$i.GetName}}(ctx context.Context, in pb.{{$i.RequestType.GetName}}) (pb.{{$i.ResponseType.GetName}}, error){
	_ = ctx
	_ = in
	return pb.{{$i.ResponseType.GetName}}{}, nil
}
{{end}}

