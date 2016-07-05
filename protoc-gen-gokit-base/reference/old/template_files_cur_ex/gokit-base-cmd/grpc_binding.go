package main

import (
	"{{.AbsoluteRelativeImportPath}}pb"
	"{{.AbsoluteRelativeImportPath}}server"
	"golang.org/x/net/context"
)

type grpcBinding struct {
	server.{{.Service.GetName}}
}

{{range $i := .Service.Methods}}
func (b grpcBinding) {{$i.GetName}}(ctx context.Context, in *pb.{{$i.RequestType.GetName}}) (*pb.{{$i.ResponseType.GetName}}, error) {
	ctx = context.WithValue(ctx, "transport", "grpc")
	ctx = context.WithValue(ctx, "request-method", "{{$i.RequestType.GetName}}")
	return b.{{.Service.GetName}}.{{$i.GetName}}(in)
}
{{end}}
