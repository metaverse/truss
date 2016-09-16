package genswagger

import (
	"github.com/TuneLab/go-truss/gendoc/doctree"

	swagger "github.com/go-openapi/spec"
	"github.com/pkg/errors"
	//spew "github.com/davecgh/go-spew/spew"

	"github.com/go-openapi/loads"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/validate"

	"github.com/TuneLab/go-truss/truss/truss"
)

type ParamsField struct {
	*doctree.HttpParameter
	MethodField *doctree.MessageField
}

func GenerateSwaggerFile(dt doctree.Doctree) ([]truss.NamedReadWriter, error) {
	svc, err := getProtoService(dt)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get service from doctree")
	}

	spec := genSwagObject(svc)

	swaggerBytes, err := spec.MarshalJSON()
	if err != nil {
		return nil, errors.Wrap(err, "unable to marshal swagger to json")
	}

	var file truss.SimpleFile
	file.Path = "service/swagger/swagger.json"
	file.Write(swaggerBytes)

	var files []truss.NamedReadWriter
	files = append(files, &file)

	return files, nil
}

func validateSwaggerBytes(spec []byte) error {
	doc, err := loads.Analyzed(spec, "2.0")
	if err != nil {
		return errors.Wrap(err, "swagger file can not be loaded into swagger doc")
	}

	err = validate.Spec(doc, strfmt.Default)
	if err != nil {
		return errors.Wrap(err, "swagger file cannot be validated")
	}

	return nil
}

// Spec: https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#swagger-object
func genSwagObject(s *doctree.ProtoService) swagger.Swagger {
	var spec swagger.Swagger

	spec.Swagger = "2.0"
	spec.Info = genSwagInfo(s)
	spec.Paths = genSwagPaths(s)
	spec.Definitions = genSwagDefinitions(s)

	return spec
}

// Spec: https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#infoObject
func genSwagInfo(s *doctree.ProtoService) *swagger.Info {
	var info swagger.Info

	info.Title = s.GetName()
	info.Version = "0.0.1"

	return &info
}

// Spec: https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#pathsObject
func genSwagPaths(s *doctree.ProtoService) *swagger.Paths {
	var paths swagger.Paths
	paths.Paths = make(map[string]swagger.PathItem)

	for _, meth := range s.Methods {
		for _, http := range meth.HttpBindings {
			paths.Paths[http.Path] = genSwagPathItem(meth, http)
		}
	}

	return &paths
}

// Spec: https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#pathItemObject
// Each PathItem must have an Operation
func genSwagPathItem(meth *doctree.ServiceMethod, httpBind *doctree.MethodHttpBinding) swagger.PathItem {
	var pathItem swagger.PathItem

	op := genSwagOperation(meth, httpBind)

	switch httpBind.Verb {
	case "get":
		pathItem.Get = op
	case "post":
		pathItem.Post = op
	case "put":
		pathItem.Put = op
	case "patch":
		pathItem.Patch = op
	case "delete":
		pathItem.Delete = op
	default:
		pathItem.Get = op
	}

	return pathItem
}

// Spec: https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#operationObject
func genSwagOperation(meth *doctree.ServiceMethod, httpBind *doctree.MethodHttpBinding) *swagger.Operation {
	var op swagger.Operation

	// Generate parameters from RequestType
	op.Parameters = genSwagParameters(meth, httpBind)

	resp := genSwagResponse(meth)

	// 200
	op.RespondsWith(200, resp)

	return &op
}

func genSwagResponse(meth *doctree.ServiceMethod) *swagger.Response {
	var resp swagger.Response

	// Response description is the comments on the http option
	if meth.RequestType.GetDescription() == "" {
		resp.Description = "ADD COMMENTS TO MESSAGE: " + meth.ResponseType.GetName()
	} else {
		resp.Description = meth.ResponseType.GetDescription()
	}

	ref, _ := swagger.NewRef("#/definitions/" + meth.ResponseType.GetName())
	var sch swagger.Schema
	sch.Ref = ref
	resp.Schema = &sch

	//resp.Schema = genSwagPrimitiveSchema(meth)

	return &resp
}

// Spec: https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#parameterObject
func genSwagParameters(meth *doctree.ServiceMethod, httpBind *doctree.MethodHttpBinding) []swagger.Parameter {
	var params []swagger.Parameter
	_ = params

	for _, h := range httpBind.Params {
		var p swagger.Parameter
		p.Name = h.GetName()
		p.In = h.Location

		field := meth.RequestType.GetByName(h.GetName()).(*doctree.MessageField)
		switch h.Location {

		// param is require if it is in the path
		// if it is in the path or query it requires
		// a type, and we add a format as well
		case "path":
			p.Required = true
			fallthrough
		case "query":
			t, f, _ := msgFieldToSwagType(field)
			p.Type = t
			p.Format = f
		// If it is in the body, it must have a schema
		case "body":
			sch := genSwagPrimitiveSchema(field)
			p.Schema = &sch
		}
		p.Description = h.GetDescription()

		params = append(params, p)
	}

	return params
}

// genSwagDefinitions generates swagger definitions for each rpc's responseType
// Spec: https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#definitionsObject
func genSwagDefinitions(s *doctree.ProtoService) swagger.Definitions {
	var def swagger.Definitions

	def = make(map[string]swagger.Schema)
	for _, meth := range s.Methods {
		var outerSchema swagger.Schema
		outerSchema.Properties = genSwagSchemaProperties(meth)
		def[meth.ResponseType.GetName()] = outerSchema
		//	spew.Fdump(os.Stderr, outerSchema)
	}

	return def
}

func genSwagSchemaProperties(meth *doctree.ServiceMethod) map[string]swagger.Schema {
	prop := make(map[string]swagger.Schema)

	for _, f := range meth.ResponseType.Fields {
		prop[f.GetName()] = genSwagPrimitiveSchema(f)
	}

	return prop
}

// Spec: https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#schemaObject
func genSwagPrimitiveSchema(msgF *doctree.MessageField) swagger.Schema {
	var sch swagger.Schema

	t, f, _ := msgFieldToSwagType(msgF)
	sch.AddType(t, f)

	return sch
}

func msgFieldToSwagType(t *doctree.MessageField) (ftype, format string, ok bool) {
	switch t.Type.GetName() {
	case "TYPE_DOUBLE":
		return "number", "double", true
	case "TYPE_FLOAT":
		return "number", "float", true
	case "TYPE_INT64":
		return "string", "int64", true
	case "TYPE_UINT64":
		return "string", "uint64", true
	case "TYPE_INT32":
		return "integer", "int32", true
	case "TYPE_FIXED64":
		return "string", "uint64", true
	case "TYPE_FIXED32":
		return "integer", "int64", true
	case "TYPE_BOOL":
		return "boolean", "boolean", true
	case "TYPE_STRING":
		return "string", "string", true
	// case "TYPE_GROUP":
	// case "TYPE_MESSAGE":
	case "TYPE_BYTES":
		return "string", "byte", true
	case "TYPE_UINT32":
		return "integer", "int64", true
	// case "TYPE_ENUM":
	case "TYPE_SFIXED32":
		return "integer", "int32", true
	case "TYPE_SFIXED64":
		return "string", "int64", true
	case "TYPE_SINT32":
		return "integer", "int32", true
	case "TYPE_SINT64":
		return "string", "int64", true
	default:
		return "", "", false
	}
}

// getProtoService finds returns the service within a doctree.Doctree
func getProtoService(dt doctree.Doctree) (*doctree.ProtoService, error) {
	md := dt.(*doctree.MicroserviceDefinition)
	files := md.Files
	var service *doctree.ProtoService

	for _, file := range files {
		if len(file.Services) > 0 {
			service = file.Services[0]
		}
	}

	if service == nil {
		return nil, errors.New("no service found")
	}

	return service, nil
}
