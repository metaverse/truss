/*
Package parsepkgname provides functions for extracting the name of a package
from a protocol buffer definition file. For example, given a protocol buffer 3
file like this:

	// A comment about this proto file
	package  examplepackage;

	// and the rest of the file goes here

The functions in this package would extract the name "examplepackage" as the
name of the protobuf package.
*/
package parsepkgname

import (
	"io"
	"unicode"

	"github.com/TuneLab/go-truss/deftree/svcparse"
)

type token int

const (
	ident token = iota
	whitespaceToken
	comment
	other
)

type Scanner interface {
	// ReadUnit must return groups of runes representing at least the following
	// lexical groups:
	//
	//     ident
	//     comments (c++ style single line comments and block comments)
	//     whitespace
	//
	// If you need a scanner which provides these out of the box, see the
	// SvcScanner struct in github.com/TuneLab/go-truss/deftree/svcparse
	ReadUnit() ([]rune, error)
}

func categorize(unit []rune) token {
	rv := other
	r := unit[0]
	switch {
	case unicode.IsLetter(r):
		rv = ident
	case unicode.IsDigit(r):
		rv = ident
	case r == '_':
		rv = ident
	case unicode.IsSpace(r):
		rv = whitespaceToken
	case r == '/' && len(unit) > 1:
		rv = comment
	}
	return rv
}

// FromReader accepts an io.Reader, the contents of which should be a
// valid proto3 file, and returns the name of the protobuf package which that
// file belongs to.
func FromReader(protofile io.Reader) (string, error) {
	scanner := svcparse.NewSvcScanner(protofile)
	return FromScanner(scanner)
}

// FromScanner accepts a Scanner for a protobuf file and returns the name of
// the protobuf package that the file belongs to.
func FromScanner(scanner Scanner) (string, error) {
	foundpackage := false

	// A nice way to ignore comments. Recursively calls itself until it
	// recieves a unit from the scanner which is not a comment.
	var readIgnoreComment func(Scanner) (token, []rune, error)
	readIgnoreComment = func(scn Scanner) (token, []rune, error) {
		unit, err := scn.ReadUnit()
		if err != nil {
			return other, nil, err
		}
		tkn := categorize(unit)
		if tkn == comment {
			return readIgnoreComment(scn)
		}
		return tkn, unit, err
	}

	// A tiny state machine to find two sequential idents: the ident "package"
	// and the ident immediately following. That second ident will be the name
	// of the package.
	for {
		tkn, unit, err := readIgnoreComment(scanner)
		if err != nil {
			return "", err
		}
		if foundpackage {
			if tkn == ident {
				return string(unit), nil
			} else if tkn == whitespaceToken {
				continue
			} else {
				foundpackage = false
			}
		} else {
			if tkn == ident && string(unit) == "package" {
				foundpackage = true
			}
		}
	}
}
