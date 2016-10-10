package pbinfo

// consider renaming to census, pbregistry, or just to "essence"

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
)

type Catalog struct {
	PkgName  string
	origin   *ast.File
	Messages []*Message
	Enums    []*Enum
	// Service contains the sole service for this Catalog
	Service *Service
}

type Message struct {
	Name   string
	origin *ast.TypeSpec
	Fields []*Field
}

type Enum struct {
	Name   string
	origin *ast.TypeSpec
}

type Map struct {
	Name   string
	Key    *FieldType
	Value  *FieldType
	origin *ast.Expr
}

type Service struct {
	Name    string
	Methods []*ServiceMethod
}

type ServiceMethod struct {
	Name         string
	RequestType  *FieldType
	ResponseType *FieldType
	origin       *ast.TypeSpec
}

type Field struct {
	Name   string
	Type   *FieldType
	origin *ast.Field
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
	// May be one of four types: *ast.MapType, *ast.Ident, *ast.StarExpr, or *ast.ArrayType
	origin ast.Expr
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

func New(goFiles []io.Reader, protoFile io.Reader) (*Catalog, error) {
	fset := token.NewFileSet()
	fileAst, err := parser.ParseFile(fset, "", goFiles[0], parser.ParseComments)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't parse go file to create catalog")
	}

	rv := Catalog{
		PkgName: fileAst.Name.Name,
	}
	sp := spew.ConfigState{
		Indent: "   ",
	}
	//for _, d := range fileAst.Decls {
	//sp.Dump(d)
	//}

	typespecs, _ := retrieveTypeSpecs(fileAst)
	for _, t := range typespecs {
		sp.Dump(t)
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
	resolveTypes(&rv)
	sp.Dump(rv)

	return &rv, nil
}

func NewEnum(e *ast.TypeSpec) (*Enum, error) {
	return &Enum{
		Name: e.Name.Name,
		//Origin: e,
	}, nil
}

func NewMessage(m *ast.TypeSpec) (*Message, error) {
	rv := &Message{
		Name: m.Name.Name,
		//Origin: m,
	}
	sp := spew.ConfigState{Indent: "   "}

	_ = sp
	//sp.Dump(m)
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

func NewMap(m ast.Expr) (*Map, error) {
	rv := &Map{
		Key:   &FieldType{},
		Value: &FieldType{},
		//origin: m,
	}
	mp := m.(*ast.MapType)
	// Key will always be an ast.Ident, Value may be an ast.Ident or an
	// ast.StarExpr->ast.Ident
	key := mp.Key.(*ast.Ident)
	rv.Key.Name = key.Name
	var keyFollower func(ast.Expr)
	keyFollower = func(e ast.Expr) {
		switch ex := e.(type) {
		case *ast.Ident:
			rv.Value.Name = ex.Name
			rv.Value.origin = e
		case *ast.StarExpr:
			rv.Value.StarExpr = true
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
			origin:   in.Type,
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
// *ast.Field. The following is an informational table of how the proto-to-go
// concepts map to the Types of an ast.Field. An arrow indicates "nested
// within".
//
//     | Type Genres | Repeated               | Naked         |
//     |-------------|------------------------|---------------|
//     | Enum        | Array -> Ident         | Ident         |
//     | Message     | Array -> Star -> Ident | Star -> Ident |
//     | BaseType    | Array -> Ident         | Ident         |
//
// Map types will always have a Key which is ident, and a value that is one of
// the Type Genres specified in the table above.
func NewField(f *ast.Field) (*Field, error) {
	rv := &Field{
		Name: f.Names[0].Name,
		Type: &FieldType{},
	}

	// TypeFollower 'follows' the type of the provided ast.Field, determining
	// the name of this fields type and if it's a StarExpr, an ArrayType, or
	// both.
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
