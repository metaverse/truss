package pbinfo

// consider renaming to census, pbregistry, or just to "essence"

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"strings"

	"github.com/pkg/errors"
)

type Catalog struct {
	// PkgName will be the pacakge name of the go file(s) analyzed. So if a
	// Go file contained "package authz", then PkgName will be "authz". If
	// multiple Go files are analyzed, it will be the package name of the last
	// go file analyzed.
	PkgName  string
	Messages []*Message
	Enums    []*Enum
	// Service contains the sole service for this Catalog
	Service *Service
}

type Message struct {
	Name   string
	Fields []*Field
}

type Enum struct {
	Name string
}

type Map struct {
	Name      string
	KeyType   *FieldType
	ValueType *FieldType
}

type Service struct {
	Name    string
	Methods []*ServiceMethod
}

type ServiceMethod struct {
	Name         string
	RequestType  *FieldType
	ResponseType *FieldType
	// Bindings contains information for mapping http paths and paramters onto
	// the fields of this ServiceMethods RequestType.
	Bindings []*HTTPBinding
}

type Field struct {
	Name string
	Type *FieldType
}

type FieldType struct {
	// Name will contain the name of the type, for example "string" or "bool"
	Name string
	// Enum contains a pointer to the Enum type this fieldtype represents, if
	// this FieldType represents an Enum. If not, Enum is nil.
	Enum *Enum
	// Message contains a pointer to the Message type this FieldType
	// represents, if this FieldType represents a Message. If not, Message is
	// nil.
	Message *Message
	// Map contains a pointer to the Map type this FieldType represents, if
	// this FieldType represents a Map. If not, Map is nil.
	Map *Map
	// StarExpr is True if this FieldType represents a pointer to a type.
	StarExpr bool
	// ArrayType is True if this FieldType represents a slice of a type.
	ArrayType bool
}

// HTTPBinding represents one of potentially several bindings from a gRPC
// service method to a particuar HTTP path/verb.
type HTTPBinding struct {
	Verb string
	Path string
	// There is one HTTPParamter for each of the parent service methods Fields.
	Params []*HTTPParameter
}

// HTTPParameter represents the location of one field for a given HTTPBinding.
// Each HTTPParameter corresponds to one Field of the parent
// ServiceMethod.RequestType.Fields
type HTTPParameter struct {
	// Field points to a Field on the Parent service methods "RequestType".
	Field *Field
	// Location will be either "body", "path", or "query"
	Location string
}

func retrieveTypeSpecs(f *ast.File) ([]*ast.TypeSpec, error) {
	var rv []*ast.TypeSpec
	for _, dec := range f.Decls {
		switch gendec := dec.(type) {
		case *ast.GenDecl:
			for _, spec := range gendec.Specs {
				switch ts := spec.(type) {
				case *ast.TypeSpec:
					rv = append(rv, ts)
				}
			}
		}
	}
	return rv, nil
}

func New(goFiles []io.Reader, protoFiles []io.Reader) (*Catalog, error) {
	rv := Catalog{}

	for _, gofile := range goFiles {
		fset := token.NewFileSet()
		fileAst, err := parser.ParseFile(fset, "", gofile, parser.ParseComments)
		if err != nil {
			return nil, errors.Wrap(err, "couldn't parse go file to create catalog")
		}

		typespecs, _ := retrieveTypeSpecs(fileAst)
		for _, t := range typespecs {
			//sp.Dump(t)
			switch typdf := t.Type.(type) {
			case *ast.Ident:
				if typdf.Name == "int32" {
					nenm, err := NewEnum(t)
					if err != nil {
						return nil, errors.Wrapf(err, "error parsing enum %q", t.Name.Name)
					}
					rv.Enums = append(rv.Enums, nenm)
				}
			case *ast.StructType:
				nmsg, err := NewMessage(t)
				if err != nil {
					return nil, errors.Wrapf(err, "error parsing message %q", t.Name.Name)
				}
				rv.Messages = append(rv.Messages, nmsg)
			case *ast.InterfaceType:
				// Each service will have two interfaces ("{SVCNAME}Server" and
				// "{SVCNAME}Client") each containing the same information that we
				// care about, but structured a bit differently. For simplicity,
				// skip the "Client" interface.
				if strings.HasSuffix("Client", t.Name.Name) {
					break
				}
				nsvc, err := NewService(t)
				if err != nil {
					return nil, errors.Wrapf(err, "error parsing service %q", t.Name.Name)
				}
				rv.Service = nsvc
			}
		}
	}
	resolveTypes(&rv)
	err := ConsolidateHTTP(&rv, protoFiles)
	if err != nil {
		return nil, errors.Wrap(err, "failed to consolidate HTTP")
	}

	return &rv, nil
}

// NewEnum returns a new Enum struct derived from an *ast.TypeSpec
func NewEnum(e *ast.TypeSpec) (*Enum, error) {
	return &Enum{
		Name: e.Name.Name,
	}, nil
}

// NewMessage returns a new Message struct derived from an *ast.TypeSpec with a
// Type of *ast.StructType.
func NewMessage(m *ast.TypeSpec) (*Message, error) {
	rv := &Message{
		Name: m.Name.Name,
	}

	strct := m.Type.(*ast.StructType)
	for _, f := range strct.Fields.List {
		nfield, err := NewField(f)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't create field %q while creating message %q", f.Names[0].Name, rv.Name)
		}
		rv.Fields = append(rv.Fields, nfield)
	}

	return rv, nil
}

// NewMap returns a new Map struct derived from an ast.Expr interface
// implemented by an *ast.MapType struct. This code cannot accept an arbitrary
// MapType, only one which follows the conventions of Go code generated by
// protoc-gen-go. Those conventions are:
//
//     1. The KeyType of the *ast.MapType will always be an ast.Ident
//     2. The ValueType may be an ast.Ident OR an ast.StarExpr -> ast.Ident
//
// These rules are a result of the rules for map fields of Protobuf messages,
// namely that a key may only be represented by a non-float basetype (e.g.
// int64, string, etc.), and that a value may be either a basetype or a Message
// type or an Enum type. In the resulting Go code, a basetype will be
// represented as an ast.Ident, while a key that is a Message or Enum type will
// be represented as an *ast.StarExpr which references an ast.Ident.
func NewMap(m ast.Expr) (*Map, error) {
	rv := &Map{
		KeyType:   &FieldType{},
		ValueType: &FieldType{},
	}
	mp := m.(*ast.MapType)
	// KeyType will always be an ast.Ident, ValueType may be an ast.Ident or an
	// ast.StarExpr->ast.Ident
	key := mp.Key.(*ast.Ident)
	rv.KeyType.Name = key.Name
	var keyFollower func(ast.Expr)
	keyFollower = func(e ast.Expr) {
		switch ex := e.(type) {
		case *ast.Ident:
			rv.ValueType.Name = ex.Name
		case *ast.StarExpr:
			rv.ValueType.StarExpr = true
			keyFollower(ex.X)
		}
	}
	keyFollower(mp.Value)

	return rv, nil
}

// NewService returns a new Service struct derived from an *ast.TypeSpec with a
// Type of *ast.InterfaceType.
func NewService(s *ast.TypeSpec) (*Service, error) {
	rv := &Service{
		Name: s.Name.Name,
	}
	asvc := s.Type.(*ast.InterfaceType)
	for _, m := range asvc.Methods.List {
		nmeth, err := NewServiceMethod(m)
		if err != nil {
			return nil, errors.Wrapf(err, "Couldn't create service method %q of service %q", m.Names[0].Name, rv.Name)
		}
		rv.Methods = append(rv.Methods, nmeth)
	}
	return rv, nil
}

// NewServiceMethod returns a new ServiceMethod derived from the provided
// *ast.Field. The given *ast.Field is intended to have a Type of *ast.FuncType
// from an *ast.InterfaceType's Methods.List attribute. Providing an *ast.Field
// with a different structure may return an error.
func NewServiceMethod(m *ast.Field) (*ServiceMethod, error) {
	rv := &ServiceMethod{
		Name: m.Names[0].Name,
	}
	ft, ok := m.Type.(*ast.FuncType)
	if !ok {
		return nil, errors.New("Provided *ast.Field.Type is not of type *ast.FuncType; cannot proceed")
	}

	input := ft.Params.List
	output := ft.Results.List

	// Zero'th param of a serverMethod is Context.context, while first param is
	// this methods RequestType
	rq := input[1]
	rs := output[0]

	makeFieldType := func(in *ast.Field) (*FieldType, error) {
		star, ok := in.Type.(*ast.StarExpr)
		if !ok {
			return nil, errors.New("could not create FieldType, in.Type is not *ast.StarExpr")
		}
		ident, ok := star.X.(*ast.Ident)
		if !ok {
			return nil, errors.New("could not create FieldType, star.Type is not *ast.Ident")
		}
		return &FieldType{
			Name:     ident.Name,
			StarExpr: true,
		}, nil
	}

	var err error
	rv.RequestType, err = makeFieldType(rq)
	if err != nil {
		return nil, errors.Wrapf(err, "RequestType creation of service method %q failed", rv.Name)
	}
	rv.ResponseType, err = makeFieldType(rs)
	if err != nil {
		return nil, errors.Wrapf(err, "ResponseType creation of service method %q failed", rv.Name)
	}

	return rv, nil
}

// NewField returns a Field struct with information distilled from an
// *ast.Field.
func NewField(f *ast.Field) (*Field, error) {
	// The following is an informational table of how the proto-to-go
	// concepts map to the Types of an ast.Field. An arrow indicates "nested
	// within". This is here as an implementors aid.
	//
	//     | Type Genres | Repeated               | Naked         |
	//     |-------------|------------------------|---------------|
	//     | Enum        | Array -> Ident         | Ident         |
	//     | Message     | Array -> Star -> Ident | Star -> Ident |
	//     | BaseType    | Array -> Ident         | Ident         |
	//
	// Map types will always have a KeyType which is ident, and a value that is one of
	// the Type Genres specified in the table above.
	rv := &Field{
		Name: f.Names[0].Name,
		Type: &FieldType{},
	}

	// TypeFollower 'follows' the type of the provided ast.Field, determining
	// the name of this fields type and if it's a StarExpr, an ArrayType, or
	// both, and modifying the return value accordingly.
	var typeFollower func(ast.Expr) error
	typeFollower = func(e ast.Expr) error {
		switch ex := e.(type) {
		case *ast.Ident:
			rv.Type.Name = ex.Name
		case *ast.StarExpr:
			rv.Type.StarExpr = true
			typeFollower(ex.X)
		case *ast.ArrayType:
			rv.Type.ArrayType = true
			typeFollower(ex.Elt)
		case *ast.MapType:
			mp, err := NewMap(ex)
			if err != nil {
				return errors.Wrapf(err, "failed to create map for field %q", rv.Name)
			}
			rv.Type.Map = mp
		}
		return nil
	}
	err := typeFollower(f.Type)
	if err != nil {
		return nil, err
	}
	return rv, nil
}