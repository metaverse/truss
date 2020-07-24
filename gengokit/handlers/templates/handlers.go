package templates

const HandlerMethods = `
{{ with $te := .}}
		{{range $i := .Methods}}
		func (s {{ToLower $te.ServiceName}}Service) {{.Name}}(ctx context.Context, in *pb.{{GoName .RequestType.Name}}) (*pb.{{GoName .ResponseType.Name}}, error){
			var resp pb.{{GoName .ResponseType.Name}}
			return &resp, nil
		}
		{{end}}
{{- end}}
`

const Handlers = `
package handlers

import (
	"context"

	pb "{{.PBImportPath -}}"
)

// NewService returns a naïve, stateless implementation of Service.
func NewService() pb.{{GoName .Service.Name}}Server {
	return {{ToLower .Service.Name}}Service{}
}

type {{ToLower .Service.Name}}Service struct{}

{{with $te := . }}
	{{range $i := $te.Service.Methods}}
		func (s {{ToLower $te.Service.Name}}Service) {{$i.Name}}(ctx context.Context, in *pb.{{GoName $i.RequestType.Name}}) (*pb.{{GoName $i.ResponseType.Name}}, error){
			var resp pb.{{GoName $i.ResponseType.Name}}
			return &resp, nil
		}
	{{end}}
{{- end}}
`
