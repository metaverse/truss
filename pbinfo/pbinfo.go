package pbinfo

// consider renaming to census, pbregistry, or just to "essence"

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io"

	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
)

type Catalog struct {
	PkgName        string
	Origin         *ast.File
	Messages       []*Message
	Enums          []*Enum
	ServiceMethods []*ServiceMethod
}

type Message struct {
	Name   string
	Origin *ast.TypeSpec
	Fields []*Field
}

type Enum struct {
	Name   string
	Origin *ast.TypeSpec
}

type Map struct {
	Name  string
	Key   *FieldType
	Value *FieldType
}

type ServiceMethod struct {
	Name   string
	Origin *ast.TypeSpec
}

type Field struct {
	Name   string
	Type   FieldType
	Origin *ast.Field
}

type FieldType struct {
	Name      string
	Enum      *Enum
	Message   *Message
	Map       *Map
	StarExpr  bool
	ArrayType bool
	// May be one of four types: *ast.MapType, *ast.Ident, *ast.StarExpr, or *ast.ArrayType
	Origin ast.Expr
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
			nmeth, err := NewServiceMethod(t)
			if err != nil {
				return nil, errors.Wrapf(err, "error parsing service method %q", t.Name.Name)
			}
			rv.ServiceMethods = append(rv.ServiceMethods, nmeth)
		}
	}
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

	sp.Dump(m)
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

func NewMap(m *ast.TypeSpec) (*Map, error) {
	return &Map{
		Name: m.Name.Name,
		//Origin: m,
	}, nil
}

func NewServiceMethod(s *ast.TypeSpec) (*ServiceMethod, error) {
	return &ServiceMethod{
		Name: s.Name.Name,
		//Origin: s,
	}, nil
}

func NewField(f *ast.Field) (*Field, error) {
	rv := &Field{
		Name: f.Names[0].Name,
	}

	typeFollower := func(e ast.Expr, typeFollower func(ast.Expr)) {
		switch etyp := e.(type) {
		case *ast.Ident:
			rv.Type.Name = etyp.Name
		case *ast.StarExpr:
			rv.Type.StarExpr = true
			typeFollower(etyp.X)
		case *ast.ArrayType:
			rv.Type.ArrayType = true
			typeFollower(etyp.Elt)
		case *ast.MapType:
			// TODO call NewMap here
		}
	}
	typeFollower(f.Type, typeFollower)
	return rv, nil
}
