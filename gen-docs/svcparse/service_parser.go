package svcparse

import (
	"fmt"
	"unicode/utf8"

	"github.com/TuneLab/gob/gen-docs/doctree"
)

func parseErr(expected string, line int, val string) error {
	return fmt.Errorf("Parser expected %v in line '%v', instead found '%v'\n", expected, line, val)
}

func ParseService(lex *SvcLexer) (*doctree.ProtoService, error) {
	tk, val := lex.GetTokenIgnoreWhitespace()
	if tk != IDENT && val != "service" {
		return nil, parseErr("'service' identifier", lex.Scn.R.LineNo, val)
	}

	toret := &doctree.ProtoService{}

	tk, val = lex.GetTokenIgnoreWhitespace()
	if tk != IDENT {
		return nil, parseErr("a string identifier", lex.Scn.R.LineNo, val)
	}
	toret.SetName(val)

	tk, val = lex.GetTokenIgnoreWhitespace()
	if tk != OPEN_BRACE {
		return nil, parseErr("'{'", lex.Scn.R.LineNo, val)
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
		return nil, parseErr("identifier 'rpc'", lex.Scn.R.LineNo, val)
	}

	toret := &doctree.ServiceMethod{}
	toret.SetDescription(desc)

	tk, val = lex.GetTokenIgnoreWhitespace()
	if tk != IDENT {
		return nil, parseErr("a string identifier", lex.Scn.R.LineNo, val)
	}
	toret.SetName(val)

	// TODO Add some kind of lookup of the existing messages instead of
	// creating a new message

	tk, val = lex.GetTokenIgnoreWhitespace()
	if tk != OPEN_PAREN {
		return nil, parseErr("'('", lex.Scn.R.LineNo, val)
	}

	tk, val = lex.GetTokenIgnoreWhitespace()
	if tk != IDENT {
		return nil, parseErr("a string identifier in first argument to method", lex.Scn.R.LineNo, val)
	}

	toret.RequestType = doctree.ProtoMessage{}
	toret.RequestType.SetName(val)

	tk, val = lex.GetTokenIgnoreWhitespace()
	if tk != CLOSE_PAREN {
		return nil, parseErr("')'", lex.Scn.R.LineNo, val)
	}

	tk, val = lex.GetTokenIgnoreWhitespace()
	if tk != IDENT || val != "returns" {
		return nil, parseErr("'returns' keyword", lex.Scn.R.LineNo, val)
	}

	tk, val = lex.GetTokenIgnoreWhitespace()
	if tk != OPEN_PAREN {
		return nil, parseErr("'('", lex.Scn.R.LineNo, val)
	}

	tk, val = lex.GetTokenIgnoreWhitespace()
	if tk != IDENT {
		return nil, parseErr("a string identifier in return argument to method", lex.Scn.R.LineNo, val)
	}

	toret.ResponseType = doctree.ProtoMessage{}
	toret.ResponseType.SetName(val)

	tk, val = lex.GetTokenIgnoreWhitespace()
	if tk != CLOSE_PAREN {
		return nil, parseErr("')' after declaration of return type to method", lex.Scn.R.LineNo, val)
	}

	tk, val = lex.GetTokenIgnoreWhitespace()
	if tk != OPEN_BRACE {
		return nil, parseErr("'{' after declaration of method signature", lex.Scn.R.LineNo, val)
	}

	opt, err := ParseHttpOption(lex)

	if err != nil {
		return nil, err
	}

	toret.HttpOption = opt

	tk, val = lex.GetTokenIgnoreWhitespace()
	if tk != CLOSE_BRACE {
		return nil, parseErr("'}' after declaration of http options", lex.Scn.R.LineNo, val)
	}

	return toret, nil

}

func ParseHttpOption(lex *SvcLexer) (*doctree.ServiceHttpOption, error) {
	// This basically assumes that every single method declaration has exactly
	// one, no more, no less, option declaration, and that that option is an
	// http option. This should be changed in the future.
	toret := &doctree.ServiceHttpOption{}

	tk, val := lex.GetTokenIgnoreWhitespace()

	if tk == COMMENT {
		toret.Description = val
	} else {
		for i := 0; i < utf8.RuneCountInString(val); i++ {
			lex.Scn.R.UnreadRune()
		}
	}

	for _, good_val := range []string{
		"option",
		"(",
		"google",
		".",
		"api",
		".",
		"http",
		")",
		"=",
		"{",
	} {
		_, val := lex.GetTokenIgnoreWhitespace()
		if val != good_val {
			return nil, parseErr("'"+good_val+"'", lex.Scn.R.LineNo, val)
		}
	}

	// Parse all the fields
	for {
		field, err := ParseOptionField(lex)
		if err != nil {
			return nil, err
		}
		if field == nil {
			break
		}
		toret.Fields = append(toret.Fields, field)
	}

	tk, val = lex.GetTokenIgnoreWhitespace()
	if tk != SYMBOL {
		return nil, parseErr("';' after http option", lex.Scn.R.LineNo, val)
	}

	return toret, nil
}

func ParseOptionField(lex *SvcLexer) (*doctree.OptionField, error) {
	toret := &doctree.OptionField{}

	tk, val := lex.GetTokenIgnoreWhitespace()

	if tk == COMMENT {
		toret.SetDescription(val)
		tk, val = lex.GetTokenIgnoreWhitespace()
	}

	// No longer any more options
	if tk == CLOSE_BRACE && val == "}" {
		return nil, nil
	}

	if tk != IDENT {
		return nil, parseErr("string identifier", lex.Scn.R.LineNo, val)
	}
	toret.SetName(val)
	toret.Kind = val

	tk, val = lex.GetTokenIgnoreWhitespace()
	if tk != SYMBOL || val != ":" {
		return nil, parseErr("symbol ':'", lex.Scn.R.LineNo, val)
	}

	tk, val = lex.GetTokenIgnoreWhitespace()
	if tk != STRING_LITERAL {
		return nil, parseErr("string literal", lex.Scn.R.LineNo, val)
	}
	toret.Value = val

	return toret, nil
}
