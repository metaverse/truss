package clientarggen

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/pkg/errors"
)

// GenerateCarveFunc returns the string form of go code for a function which
// will parse a string representing a typed slice. The type of the slice is
// based on the "GoType" of the input ClientArg struct. For example, if GoType
// field of the ClientArg is "uint32", then the source code returned would be
// for a function which could accept a string and marshal that string into a
// []uint32.
//
// Supported types are:
//
//     uint32
//     uint64
//     int32
//     int64
//     float32
//     float64
//     bool
//     string
func GenerateCarveFunc(m *ClientArg) string {
	parsefunc := ParseFunction(m.GoType)
	typeconv := TypeConversion(m.GoType)

	executor := struct {
		ParseFunc string
		TypeConv  string
		GoArg     string
		GoType    string
		FlagType  string
	}{
		parsefunc,
		typeconv,
		m.GoArg,
		m.GoType,
		m.FlagType,
	}

	code, err := ApplyTemplate("CarveTemplate", CarveTemplate, executor, nil)
	if err != nil {
		panic(fmt.Sprintf("Couldn't apply template: %v", err))
	}

	return code
}

func GenerateCarveInvocation(m *ClientArg) string {
	code, err := ApplyTemplate("CarveInvocationTempl", invocTemplate, m, nil)
	if err != nil {
		panic(fmt.Sprintf("Couldn't apply template: %v", err))
	}
	return code
}

var invocTemplate = `{{.GoArg}} := Carve{{.GoArg}}({{.FlagArg}})`

var CarveTemplate = `
func Carve{{.GoArg}}(inpt {{.FlagType}}) []{{.GoType}} {
	inpt = strings.Trim(inpt, "[] ")
	slc := strings.Split(inpt, ",")
	var rv []{{.GoType}}

	for _, item := range slc {
		item = strings.Trim(item, " ")
		item = strings.Replace(item, "'", "\"", -1)
		if len(item) == 0 {
			continue
		}
		{{.ParseFunc}}
		if err != nil {
			panic(fmt.Sprintf("couldn't parse '%v' of '%v'", item, inpt))
		}
		rv = append(rv, {{.TypeConv}})
	}
	return rv
}

`

func ParseFunction(gotype string) string {
	invoc := ""
	switch {
	case strings.Contains(gotype, "uint32"):
		invoc = "%s, err := strconv.ParseUint(%s, 10, 32)"
	case strings.Contains(gotype, "uint64"):
		invoc = "%s, err := strconv.ParseUint(%s, 10, 64)"
	case strings.Contains(gotype, "int32"):
		invoc = "%s, err := strconv.ParseInt(%s, 10, 32)"
	case strings.Contains(gotype, "int64"):
		invoc = "%s, err := strconv.ParseInt(%s, 10, 64)"
	case strings.Contains(gotype, "bool"):
		invoc = "%s, err := strconv.ParseBool(%s)"
	case strings.Contains(gotype, "float32"):
		invoc = "%s, err := strconv.ParseFloat(%s, 32)"
	case strings.Contains(gotype, "float64"):
		invoc = "%s, err := strconv.ParseFloat(%s, 64)"
	case strings.Contains(gotype, "string"):
		invoc = "%s, err := strconv.Unquote(%s)"
	}
	return fmt.Sprintf(invoc, "tmp", "item")
}

func TypeConversion(gotype string) string {
	conv := ""
	switch {
	case strings.Contains(gotype, "uint32"):
		conv = "uint32(%s)"
	case strings.Contains(gotype, "int32"):
		conv = "int32(%s)"
	case strings.Contains(gotype, "float32"):
		conv = "float32(%s)"
	default:
		conv = "%s"
	}
	return fmt.Sprintf(conv, "tmp")
}

// ApplyTemplate applies a template with a given name, executor context, and
// function map. Returns the output of the template on success, returns an
// error if template failed to execute.
func ApplyTemplate(name string, tmpl string, executor interface{}, fncs template.FuncMap) (string, error) {
	codeTemplate := template.Must(template.New(name).Funcs(fncs).Parse(tmpl))

	code := bytes.NewBuffer(nil)
	err := codeTemplate.Execute(code, executor)
	if err != nil {
		return "", errors.Wrapf(err, "attempting to execute template %q", name)
	}
	return code.String(), nil
}
