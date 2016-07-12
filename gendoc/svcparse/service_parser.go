package svcparse

import (
	"fmt"
	"io"

	"github.com/TuneLab/gob/gendoc/doctree"
)

func parseErr(expected string, line int, val string) error {
	err := fmt.Errorf("Parser expected %v in line '%v', instead found '%v'\n", expected, line, val)
	return err
}

// fastForwardTill moves the lexer forward till a token with a certain value
// has been found. If an illegal token or EOF is reached, returns an error
func fastForwardTill(lex *SvcLexer, delim string) error {
	for {
		tk, val := lex.GetTokenIgnoreWhitespace()
		if tk == EOF || tk == ILLEGAL {
			return fmt.Errorf("In fastForwardTill found token of type '%v' and val '%v'\n", tk, val)
		} else if val == delim {
			return nil
		}
	}
}

func ParseService(lex *SvcLexer) (*doctree.ProtoService, error) {
	tk, val := lex.GetTokenIgnoreWhitespace()
	if tk == EOF {
		return nil, io.EOF
	}
	if tk != IDENT && val != "service" {
		return nil, parseErr("'service' identifier", lex.GetLineNumber(), val)
	}

	toret := &doctree.ProtoService{}

	tk, val = lex.GetTokenIgnoreWhitespace()
	if tk != IDENT {
		return nil, parseErr("a string identifier", lex.GetLineNumber(), val)
	}
	toret.SetName(val)

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

func ParseMethod(lex *SvcLexer) (*doctree.ServiceMethod, error) {
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

	toret := &doctree.ServiceMethod{}
	toret.SetDescription(desc)

	tk, val = lex.GetTokenIgnoreWhitespace()
	if tk != IDENT {
		return nil, parseErr("a string identifier", lex.GetLineNumber(), val)
	}
	toret.SetName(val)

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

	toret.RequestType = &doctree.ProtoMessage{}
	toret.RequestType.SetName(val)

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

	toret.ResponseType = &doctree.ProtoMessage{}
	toret.ResponseType.SetName(val)

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
	toret.HttpBindings = bindings

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

func ParseHttpBindings(lex *SvcLexer) ([]*doctree.MethodHttpBinding, error) {

	rv := make([]*doctree.MethodHttpBinding, 0)
	new_opt := &doctree.MethodHttpBinding{}

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
			new_opt.SetDescription(val)
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

func ParseBindingFields(lex *SvcLexer) ([]*doctree.BindingField, error) {

	rv := make([]*doctree.BindingField, 0)
	field := &doctree.BindingField{}
	for {
		tk, val := lex.GetTokenIgnoreWhitespace()
		for {
			if tk == COMMENT {
				field.SetDescription(val)
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
		field.SetName(val)

		tk, val = lex.GetTokenIgnoreWhitespace()
		if tk != SYMBOL || val != ":" {
			return nil, parseErr("symbol ':'", lex.GetLineNumber(), val)
		}

		tk, val = lex.GetTokenIgnoreWhitespace()
		if tk != STRING_LITERAL {
			return nil, parseErr("string literal", lex.GetLineNumber(), val)
		}

		field.Value = val

		rv = append(rv, field)
		field = &doctree.BindingField{}
	}

	return rv, nil
}
