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
)

func parseErr(expected string, line int, val string) error {
	err := fmt.Errorf("parser expected %v in line '%v', instead found '%v'", expected, line, val)
	return err
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
	Description string
	Fields      []*Field
}

// Field holds information extracted by the parser about each field within each
// HTTP binding.
type Field struct {
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
		return nil, parseErr("'service' identifier", lex.GetLineNumber(), val)
	}

	toret := &Service{}

	tk, val = lex.GetTokenIgnoreWhitespace()
	if tk != IDENT {
		return nil, parseErr("a string identifier", lex.GetLineNumber(), val)
	}
	toret.Name = val

	tk, val = lex.GetTokenIgnoreWhitespace()
	if tk != OPEN_BRACE {
		return nil, parseErr("'{'", lex.GetLineNumber(), val)
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
	if tk == COMMENT {
		desc = val
		tk, val = lex.GetTokenIgnoreWhitespace()
	}

	switch {
	// Hit the end of the service definition, so return a nil method
	case tk == CLOSE_BRACE:
		return nil, nil
	case tk != IDENT || val != "rpc":
		return nil, parseErr("identifier 'rpc'", lex.GetLineNumber(), val)
	}

	toret := &Method{}
	toret.Description = desc

	tk, val = lex.GetTokenIgnoreWhitespace()
	if tk != IDENT {
		return nil, parseErr("a string identifier", lex.GetLineNumber(), val)
	}
	toret.Name = val

	// TODO Add some kind of lookup of the existing messages instead of
	// creating a new message

	tk, val = lex.GetTokenIgnoreWhitespace()
	if tk != OPEN_PAREN {
		return nil, parseErr("'('", lex.GetLineNumber(), val)
	}

	tk, val = lex.GetTokenIgnoreWhitespace()
	// Here we ignore the "stream" keyword which may appear in the arguments of
	// an RPC definition
	if val == "stream" {
		tk, val = lex.GetTokenIgnoreWhitespace()
	}
	if tk != IDENT {
		return nil, parseErr("a string identifier in first argument to method", lex.GetLineNumber(), val)
	}

	toret.RequestType = val

	tk, val = lex.GetTokenIgnoreWhitespace()
	if tk != CLOSE_PAREN {
		return nil, parseErr("')'", lex.GetLineNumber(), val)
	}

	tk, val = lex.GetTokenIgnoreWhitespace()
	if tk != IDENT || val != "returns" {
		return nil, parseErr("'returns' keyword", lex.GetLineNumber(), val)
	}

	tk, val = lex.GetTokenIgnoreWhitespace()
	if tk != OPEN_PAREN {
		return nil, parseErr("'('", lex.GetLineNumber(), val)
	}

	tk, val = lex.GetTokenIgnoreWhitespace()
	// Here we ignore the "stream" keyword which may appear in the arguments of
	// an RPC definition
	if val == "stream" {
		tk, val = lex.GetTokenIgnoreWhitespace()
	}
	if tk != IDENT {
		return nil, parseErr("a string identifier in return argument to method", lex.GetLineNumber(), val)
	}

	toret.ResponseType = val

	tk, val = lex.GetTokenIgnoreWhitespace()
	if tk != CLOSE_PAREN {
		return nil, parseErr("')' after declaration of return type to method", lex.GetLineNumber(), val)
	}

	tk, val = lex.GetTokenIgnoreWhitespace()
	if tk != OPEN_BRACE {
		return nil, parseErr("'{' after declaration of method signature", lex.GetLineNumber(), val)
	}

	bindings, err := ParseHttpBindings(lex)
	if err != nil {
		return nil, err
	}
	toret.HTTPBindings = bindings

	// There should be a semi-colon immediately following all 'option'
	// declarations, which we should check for
	tk, val = lex.GetTokenIgnoreCommentAndWhitespace()
	if tk != SYMBOL || val != ";" {
		return nil, parseErr("';' after declaration of http options", lex.GetLineNumber(), val+tk.String())
	}

	tk, val = lex.GetTokenIgnoreCommentAndWhitespace()
	if tk != CLOSE_BRACE {
		return nil, parseErr("'}' after declaration of http options marking end of rpc declarations", lex.GetLineNumber(), val+tk.String())
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
			return nil, parseErr("non-illegal input", lex.GetLineNumber(), tk.String())
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
		fields, err := ParseBindingFields(lex)
		if err != nil {
			return nil, err
		}
		new_opt.Fields = fields
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
				return nil, parseErr("non-illegal token while parsing HttpBindings", lex.GetLineNumber(), fmt.Sprintf("(%v) of type %v", val, tk))
			} else {
				return nil, parseErr("close brace or comment while parsing http bindings", lex.GetLineNumber(), tk.String()+val)
			}
			tk, val = lex.GetTokenIgnoreWhitespace()
		}
	case val == "additional_bindings":
		err := fastForwardTill(lex, "{")
		if err != nil {
			return nil, err
		}
		fields, err := ParseBindingFields(lex)
		if err != nil {
			return nil, err
		}
		new_opt.Fields = fields
		err = fastForwardTill(lex, "}")
		if err != nil {
			return nil, err
		}
		return append(rv, new_opt), nil
	}

	return nil, parseErr("'option' or 'additional_bindings' while parsing options", lex.GetLineNumber(), val)
}

func ParseBindingFields(lex *SvcLexer) ([]*Field, error) {

	rv := make([]*Field, 0)
	field := &Field{}
	for {
		tk, val := lex.GetTokenIgnoreWhitespace()
		for {
			if tk == COMMENT {
				field.Description = val
				tk, val = lex.GetTokenIgnoreWhitespace()
			} else if tk == EOF || tk == ILLEGAL {
				return nil, parseErr("non-illegal token whil parsing binding fields", lex.GetLineNumber(), val)
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

		field.Kind = val
		field.Name = val

		tk, val = lex.GetTokenIgnoreWhitespace()
		if tk != SYMBOL || val != ":" {
			return nil, parseErr("symbol ':'", lex.GetLineNumber(), val)
		}

		tk, val = lex.GetTokenIgnoreWhitespace()
		if tk != STRING_LITERAL {
			return nil, parseErr("string literal", lex.GetLineNumber(), val)
		}

		noqoute, err := strconv.Unquote(val)
		if err != nil {
			return nil, err
		}
		field.Value = noqoute

		rv = append(rv, field)
		field = &Field{}
	}

	return rv, nil
}
