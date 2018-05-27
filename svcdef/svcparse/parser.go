// Svcparse, which stands for "service parser" will parse the 'service'
// declarations within a provided protobuf and associate comments within that
// file with the various components of the service. Specifically, it handles
// google's httpoptions and the association of those comments. This is
// necessary because while it is possible to derive the structure of
// httpbindings for a service using the mainline protoc, it will not allow
// access to comments associated with components of those options.
//
// Thus this parser was written to associate comments on http bindings with
// their components, since those comments are used for documentation.
//
// NOTE
//
// Currently, this parser assumes that it's input does contain EXACTLY ONE
// valid service definition. Providing an input file that does not contain a
// service definition will return an error.
package svcparse

import (
	"fmt"
	"io"
	"strconv"

	"github.com/pkg/errors"
)

type optionParseErr struct {
	err error
}

func (o optionParseErr) Optional() bool {
	return true
}

func (o optionParseErr) Error() string {
	return o.err.Error()
}

func optionalParseErr(expected string, line int, val string) error {
	return optionParseErr{
		err: fmt.Errorf("parser expected %v in line '%v', instead found '%v'", expected, line, val),
	}
}

type parserErr struct {
	expected string
	line     int
	val      string
}

func (pe parserErr) Error() string {
	return fmt.Sprintf("parser expected %v in line '%v', instead found '%v'", pe.expected, pe.line, pe.val)
}

// fastForwardTill moves the lexer forward till a token with a certain value
// has been found. If an illegal token or EOF is reached, returns an error
func fastForwardTill(lex *SvcLexer, delim string) error {
	for {
		tk, val := lex.GetTokenIgnoreWhitespace()
		if tk == EOF || tk == ILLEGAL {
			return fmt.Errorf("in fastForwardTill found token of type '%v' and val '%v'", tk, val)
		} else if val == delim {
			return nil
		}
	}
}

// Each of the following structs exists as a distillation of a corresponding
// deftree struct, only including what's necessary for this parser. The reason
// we define these structs instead of using the ones within deftree is because
// doing so would couple this package to that package, and cause import cycles.

// Service keeps track of the information extracted by the parser about each
// service in the file.
type Service struct {
	Name    string
	Methods []*Method
}

// Method holds information extracted by the parser about each method within
// each service.
type Method struct {
	Name         string
	Description  string
	RequestType  string
	ResponseType string
	HTTPBindings []*HTTPBinding
}

// HTTPBinding holds information extracted by the parser about each HTTP
// binding within each method.
type HTTPBinding struct {
	// At this time, the way to provide a "description" of an HTTP binding is
	// to write a comment directly above the "option" statement in an rpc.
	Description string
	Fields      []*Field
	// CustomHTTPPattern contains the fields for a `custom` HTTP verb. It's
	// name comes from the name for this construct in the http annotations
	// protobuf file. The following is an example of a protobuf service with
	// custom http verbs and the equivelent HTTPBinding literal:
	//
	// Protobuf code:
	//
	// service ExmplService {
	//   rpc ExmplMethod (RequestStrct) returns (ResponseStrct) {
	//     option (google.api.http) = {
	//       custom {
	//         // The verb itself goes in the "kind" field
	//         kind: "MYVERBHERE"
	//         // Likewise, path goes in the "path" field. As always, the path
	//         // may have parameters within it.
	//         path: "/foo/bar/{SomeFieldName}"
	//       }
	//       // This 'body' field is optional
	//       body: "*"
	//     };
	//   }
	// }
	//
	// Resulting HTTPBinding:
	//
	// HTTPBinding{
	//     Fields: []*Field{
	//         &Field{
	//             Description: "// This 'body' field is optional\n",
	//             Name:  "body",
	//             Kind:  "body",
	//             Value: "*",
	//         },
	//     },
	//     CustomHTTPPattern: []*Field{
	//         &Field{
	//             Description: "// The verb itself goes in the \"kind\" field\n",
	//             Name:        "kind",
	//             Kind:        "kind",
	//             Value:       "MYVERBHERE",
	//         },
	//         &Field{
	//             Description: "// Likewise, path goes in the \"path\" field. As always, the path\n\t\t\t\t\t// may have parameters within it.\n",
	//             Name:  "path",
	//             Kind:  "path",
	//             Value: "/foo/bar/{SomeFieldName}",
	//         },
	//     },
	// },
	CustomHTTPPattern []*Field
}

// Field holds information extracted by the parser about each field within each
// HTTP binding.
type Field struct {
	// Name acts as an 'alias' for the Field. Usually, it has the same value as
	// "Kind", though there are no guarantees that they'll be the same. Name
	// should never be used as part of "business logic" it is purely as a
	// human-readable decorative field.
	Name        string
	Description string
	Kind        string
	Value       string
}

// ParseService will parse a proto file and return the the struct
// representation of that service.
func ParseService(lex *SvcLexer) (*Service, error) {
	tk, val := lex.GetTokenIgnoreWhitespace()
	if tk == EOF {
		return nil, io.ErrUnexpectedEOF
	}
	if tk != IDENT && val != "service" {
		return nil, parserErr{
			expected: "'service' identifier",
			line:     lex.GetLineNumber(),
			val:      val,
		}
	}

	toret := &Service{}

	tk, val = lex.GetTokenIgnoreWhitespace()
	if tk != IDENT {
		return nil, parserErr{
			expected: "a string identifier",
			line:     lex.GetLineNumber(),
			val:      val,
		}
	}
	toret.Name = val

	tk, val = lex.GetTokenIgnoreWhitespace()
	if tk != OPEN_BRACE {
		return nil, parserErr{
			expected: "'{'",
			line:     lex.GetLineNumber(),
			val:      val,
		}
	}

	// Recursively parse the methods of this service
	for {
		rpc, err := ParseMethod(lex)

		if err != nil {
			return nil, err
		}

		if rpc != nil {
			toret.Methods = append(toret.Methods, rpc)
		} else {
			break
		}
	}

	return toret, nil
}

func ParseMethod(lex *SvcLexer) (*Method, error) {
	var desc string

	tk, val := lex.GetTokenIgnoreWhitespace()
	for {
		if tk == COMMENT {
			desc = val
			tk, val = lex.GetTokenIgnoreWhitespace()
		} else {
			break
		}
	}

	switch {
	// Hit the end of the service definition, so return a nil method
	case tk == CLOSE_BRACE:
		return nil, nil
	case tk != IDENT || val != "rpc":
		return nil, parserErr{
			expected: "identifier 'rpc'",
			line:     lex.GetLineNumber(),
			val:      val,
		}
	}

	toret := &Method{}
	toret.Description = desc

	tk, val = lex.GetTokenIgnoreWhitespace()
	if tk != IDENT {
		return nil, parserErr{
			expected: "a string identifier",
			line:     lex.GetLineNumber(),
			val:      val,
		}
	}
	toret.Name = val

	// TODO Add some kind of lookup of the existing messages instead of
	// creating a new message

	tk, val = lex.GetTokenIgnoreWhitespace()
	if tk != OPEN_PAREN {
		return nil, parserErr{
			expected: "'('",
			line:     lex.GetLineNumber(),
			val:      val,
		}
	}

	tk, val = lex.GetTokenIgnoreWhitespace()
	// Here we ignore the "stream" keyword which may appear in the arguments of
	// an RPC definition
	if val == "stream" {
		tk, val = lex.GetTokenIgnoreWhitespace()
	}
	if tk != IDENT {
		return nil, parserErr{
			expected: "a string identifier in first argument to method",
			line:     lex.GetLineNumber(),
			val:      val,
		}
	}

	toret.RequestType = val

	tk, val = lex.GetTokenIgnoreWhitespace()
	if tk != CLOSE_PAREN {
		return nil, parserErr{
			expected: "')'",
			line:     lex.GetLineNumber(),
			val:      val,
		}
	}

	tk, val = lex.GetTokenIgnoreWhitespace()
	if tk != IDENT || val != "returns" {
		return nil, parserErr{
			expected: "'returns' keyword",
			line:     lex.GetLineNumber(),
			val:      val,
		}
	}

	tk, val = lex.GetTokenIgnoreWhitespace()
	if tk != OPEN_PAREN {
		return nil, parserErr{
			expected: "'('",
			line:     lex.GetLineNumber(),
			val:      val,
		}
	}

	tk, val = lex.GetTokenIgnoreWhitespace()
	// Here we ignore the "stream" keyword which may appear in the arguments of
	// an RPC definition
	if val == "stream" {
		tk, val = lex.GetTokenIgnoreWhitespace()
	}
	if tk != IDENT {
		return nil, parserErr{
			expected: "a string identifier in return argument to method",
			line:     lex.GetLineNumber(),
			val:      val,
		}
	}

	toret.ResponseType = val

	tk, val = lex.GetTokenIgnoreWhitespace()
	if tk != CLOSE_PAREN {
		return nil, parserErr{
			expected: "')' after declaration of return type to method",
			line:     lex.GetLineNumber(),
			val:      val,
		}
	}

	tk, val = lex.GetTokenIgnoreWhitespace()
	if tk != OPEN_BRACE {
		return nil, parserErr{
			expected: "'{' after declaration of method signature",
			line:     lex.GetLineNumber(),
			val:      val,
		}
	}

	bindings, err := ParseHttpBindings(lex)
	if err != nil {
		return nil, err
	}
	// End of RPC (no httpoptions)
	if bindings == nil {
		return nil, nil
	}
	toret.HTTPBindings = bindings

	// There should be a semi-colon immediately following all 'option'
	// declarations, which we should check for
	tk, val = lex.GetTokenIgnoreCommentAndWhitespace()
	if tk != SYMBOL || val != ";" {
		return nil, parserErr{
			expected: "';' after declaration of http options",
			line:     lex.GetLineNumber(),
			val:      val + tk.String(),
		}
	}

	tk, val = lex.GetTokenIgnoreCommentAndWhitespace()
	if tk != CLOSE_BRACE {
		return nil, parserErr{
			expected: "'}' after declaration of http options marking end of rpc declarations",
			line:     lex.GetLineNumber(),
			val:      val + tk.String(),
		}
	}

	return toret, nil

}

func ParseHttpBindings(lex *SvcLexer) ([]*HTTPBinding, error) {

	rv := make([]*HTTPBinding, 0)
	new_opt := &HTTPBinding{}

	tk, val := lex.GetTokenIgnoreWhitespace()
	// If there's a comment before the declaration of a new HttpBinding, then
	// we set that comment as the description of that HttpBinding. Since we're
	// iterating through tokens while ignoring whitespace, this technically
	// means that a comment could be "detached" from an HttpBinding, but still
	// precede that HttpBinding, and this parser will still set that comment as
	// the description of the HttpBinding. This is potentially a bug.
	//
	// TODO: Fix this property so that newlines, or their absence, are the
	// basis for association.
	for {
		if tk == COMMENT {
			new_opt.Description = val
			tk, val = lex.GetTokenIgnoreWhitespace()
		} else if tk == EOF || tk == ILLEGAL {
			return nil, parserErr{
				expected: "non-illegal input",
				line:     lex.GetLineNumber(),
				val:      tk.String(),
			}
		} else {
			break
		}
	}

	switch {
	case val == "option":
		err := fastForwardTill(lex, "{")
		if err != nil {
			return nil, err
		}
		fields, custom, err := ParseBindingFields(lex)
		if err != nil {
			return nil, err
		}
		new_opt.Fields = fields
		new_opt.CustomHTTPPattern = custom
		good_position := lex.GetPosition()

		tk, val = lex.GetTokenIgnoreWhitespace()
		for {
			if tk == CLOSE_BRACE {
				return append(rv, new_opt), nil
			} else if tk == COMMENT {
				good_position = lex.GetPosition()
			} else if val == "additional_bindings" {
				lex.UnGetToPosition(good_position)
				more_bindings, err := ParseHttpBindings(lex)
				if err != nil {
					return nil, err
				}
				rv = append(rv, more_bindings...)
				good_position = lex.GetPosition()
			} else if tk == EOF || tk == ILLEGAL {
				return nil, parserErr{
					expected: "legal token while parsing HttpBindings",
					line:     lex.GetLineNumber(),
					val:      fmt.Sprintf("(%v) of type %v", val, tk),
				}
			} else {
				return nil, parserErr{
					expected: "close brace or comment while parsing http bindings",
					line:     lex.GetLineNumber(),
					val:      tk.String() + val,
				}
			}
			tk, val = lex.GetTokenIgnoreWhitespace()
		}
	case val == "additional_bindings":
		err := fastForwardTill(lex, "{")
		if err != nil {
			return nil, err
		}
		fields, custom, err := ParseBindingFields(lex)
		if err != nil {
			return nil, err
		}
		new_opt.Fields = fields
		new_opt.CustomHTTPPattern = custom
		err = fastForwardTill(lex, "}")
		if err != nil {
			return nil, err
		}
		return append(rv, new_opt), nil
	case val == "}":
		// End of RPC
		return nil, nil
	}

	optErr := optionalParseErr("'}', 'option' or 'additional_bindings' while parsing options", lex.GetLineNumber(), val)

	return nil, optErr
}

func ParseBindingFields(lex *SvcLexer) (fields []*Field, custom []*Field, err error) {
	field := &Field{}
	for {
		tk, val := lex.GetTokenIgnoreWhitespace()
		for {
			if tk == COMMENT {
				field.Description = val
				tk, val = lex.GetTokenIgnoreWhitespace()
			} else if tk == EOF || tk == ILLEGAL {
				return nil, nil, parserErr{
					expected: "legal token while parsing binding fields",
					line:     lex.GetLineNumber(),
					val:      val,
				}
			} else {
				break
			}
		}
		// No longer any more fields
		if tk == CLOSE_BRACE && val == "}" {
			lex.UnGetToken()
			break
		} else if val == "additional_bindings" {
			lex.UnGetToken()
			break
		}
		// Use recursion to parse custom HTTP verb sections
		if val == "custom" {
			err := fastForwardTill(lex, "{")
			if err != nil {
				return nil, nil, errors.Wrap(err, "cannot fastforward till opening brace")
			}
			// Since there cannot be a custom within a custom, we ignore custom
			// values returned from a recursive parsing of more fields
			c, _, err := ParseBindingFields(lex)
			if err != nil {
				return nil, nil, errors.Wrap(err, "cannot parse custom binding fields")
			}
			custom = c
			err = fastForwardTill(lex, "}")
			if err != nil {
				return nil, nil, errors.Wrap(err, "cannot fastforward to closing brace")
			}
			continue
		}

		field.Kind = val
		field.Name = val

		tk, val = lex.GetTokenIgnoreWhitespace()
		if tk != SYMBOL || val != ":" {
			return nil, nil, parserErr{
				expected: "symbol ':'",
				line:     lex.GetLineNumber(),
				val:      val,
			}
		}

		tk, val = lex.GetTokenIgnoreWhitespace()
		if tk != STRING_LITERAL {
			return nil, nil, parserErr{
				expected: "string literal",
				line:     lex.GetLineNumber(),
				val:      val,
			}
		}

		noqoute, err := strconv.Unquote(val)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "cannot unquote value %q", val)
		}
		field.Value = noqoute

		fields = append(fields, field)
		field = &Field{}
	}

	return fields, custom, nil
}
