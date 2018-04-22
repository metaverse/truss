package gendoc

import (
	"fmt"
	"strings"

	"github.com/tuneinc/truss/deftree"
)

// prindent is a utility function for creating a formatted string with a given
// amount of indentation.
func prindent(depth int, format string, args ...interface{}) string {
	s := ""
	for i := 0; i < depth; i++ {
		s += "    "
	}
	return s + fmt.Sprintf(format, args...)
}

// strRepeat takes a string and an int `n` and returns a string representing
// the input repeated `n` times.
func strRepeat(in string, count int) string {
	rv := ""
	for ; count > 0; count-- {
		rv += in
	}
	return rv
}

// nameLink returns a markdown formatted link to the string provided as input.
// nameLink input is intended to be a dotted "Qualified Name" such as
// `example.footype`. Namelink will take the last string seperated by dots
// (going by the example, the string `footype`) and then create a link with
// that text to an anchor of the same name (e.g. `[footype](#footype)`). If the
// input does not contain periods, the input is returned.
//
// nameLink is primarily used to create intra-document links where one type is
// mentioned back to the definition of that type, usually where the table of
// fields for one type lists a field of some other user-defined type.
func nameLink(in string) string {
	if !strings.Contains(in, ".") {
		return in
	}
	split := strings.Split(in, ".")
	name := split[len(split)-1]
	return fmt.Sprintf("[%v](#%v)", name, name)
}

// defaultDescribeMarkdown provides a "default" way of describing the markdown
// form of an object fulfilling the "describable" interface. Returns a markdown
// string of the Name of the describable as a title of order "depth", followed
// by the description of the describable. As an example, suppose a describable
// and depth such as the following is provided:
//
//     exmplDescrb = {
//         Name: "Foo",
//         Description: "Whizbangboom",
//     }
//     depth = 2
//
// In this case, the returned string would be:
//
//     `## Foo
//
//     Whizbangboom
//
//     `
func defaultDescribeMarkdown(d deftree.Describable, depth int) string {
	rv := prindent(0, "%v %v\n\n", strRepeat("#", depth), d.GetName())
	if len(d.GetDescription()) > 1 {
		rv += prindent(0, "%v\n\n", d.GetDescription())
	}
	return rv
}

func MdMicroserviceDefinition(m *deftree.MicroserviceDefinition, depth int) string {
	rv := defaultDescribeMarkdown(m, depth)
	for _, file := range m.Files {
		rv += MdFile(file, depth+1)
	}

	rv += doc_css
	return rv
}

func MdFile(f *deftree.ProtoFile, depth int) string {
	rv := defaultDescribeMarkdown(f, depth)

	if len(f.Messages) > 0 {
		rv += fmt.Sprintf("%v %v\n\n", strRepeat("#", depth+1), "Messages")
		for _, msg := range f.Messages {
			rv += MdMessage(msg, depth+2)
		}
	}

	if len(f.Enums) > 0 {
		rv += fmt.Sprintf("%v %v\n\n", strRepeat("#", depth+1), "Enums")
		for _, enum := range f.Enums {
			rv += MdEnum(enum, depth+2)
		}
	}

	if len(f.Services) > 0 {
		rv += fmt.Sprintf("%v %v\n\n", strRepeat("#", depth+1), "Services")
		for _, svc := range f.Services {
			rv += MdService(svc, depth+2)
		}
	}
	return rv
	return ""
}

func MdMessage(m *deftree.ProtoMessage, depth int) string {
	// Embed an anchor above this title, to allow for things to link to it. The
	// 'name' of this anchor link is just the name of this ProtoMessage. This
	// may not reliably create unique 'name's in all cases, but I've not
	// encountered any problems with this aproach thus far so I'm keeping it.
	rv := `<a name="` + m.Name + `"></a>` + "\n\n"
	rv += prindent(0, "%v %v\n\n", strRepeat("#", depth), m.Name)
	if len(m.Description) > 1 {
		rv += prindent(0, "%v\n\n", m.Description)
	}

	// If there's no fields, avoid printing an empty table by short-circuiting
	if len(m.Fields) < 1 {
		rv += "\n"
		return rv
	}

	rv += "| Name | Type | Field Number | Description|\n"
	rv += "| ---- | ---- | ------------ | -----------|\n"
	for _, f := range m.Fields {
		safe_desc := f.GetDescription()
		safe_desc = strings.Replace(safe_desc, "\n", "", -1)
		rv += fmt.Sprintf("| %v | %v | %v | %v |\n", f.GetName(), nameLink(f.Type.Name), f.Number, safe_desc)
	}
	rv += "\n"
	return rv

	return ""
}

func MdEnum(e *deftree.ProtoEnum, depth int) string {
	rv := defaultDescribeMarkdown(e, depth)
	rv += "| Number | Name |\n"
	rv += "| ------ | ---- |\n"
	for _, val := range e.Values {
		rv += fmt.Sprintf("| %v | %v |\n", val.Number, val.Name)
	}
	rv += "\n\n"
	return rv
}

func MdService(s *deftree.ProtoService, depth int) string {
	rv := defaultDescribeMarkdown(s, depth)

	rv += "| Method Name | Request Type | Response Type | Description|\n"
	rv += "| ---- | ---- | ------------ | -----------|\n"
	for _, meth := range s.Methods {
		req_link := nameLink(meth.RequestType.GetName())
		res_link := nameLink(meth.ResponseType.GetName())

		rv += prindent(0, "| %v | %v | %v | %v |\n", meth.GetName(), req_link, res_link, meth.GetDescription())
	}
	rv += "\n"
	rv += fmt.Sprintf("%v %v - Http Methods\n\n", strRepeat("#", depth), s.Name)

	for _, meth := range s.Methods {
		rv += MdMethod(meth, depth+1)
	}
	return rv
}

func MdMethod(m *deftree.ServiceMethod, depth int) string {
	rv := ""

	for _, bind := range m.HttpBindings {
		rv += MdHTTPBinding(bind, depth)
	}

	return rv
}

func MdHTTPBinding(b *deftree.MethodHttpBinding, depth int) string {
	rv := fmt.Sprintf("%v %v `%v`\n\n", strRepeat("#", depth), strings.ToUpper(b.Verb), b.Path)

	rv += b.GetDescription() + "\n\n"

	rv += "| Parameter Name | Location | Type |\n"
	rv += "| ---- | ---- | ------------ |\n"
	for _, param := range b.Params {
		rv += fmt.Sprintf("| %v | %v | %v |\n", param.GetName(), param.Location, nameLink(param.Type))
	}
	rv += "\n"

	return rv
}
