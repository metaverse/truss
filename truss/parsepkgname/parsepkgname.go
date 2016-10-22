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

type Token int

const (
	IDENT Token = iota
	WHITESPACE
	COMMENT
	OTHER
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

func categorize(unit []rune) Token {
	rv := OTHER
	r := unit[0]
	switch {
	case unicode.IsLetter(r):
		rv = IDENT
	case unicode.IsDigit(r):
		rv = IDENT
	case r == '_':
		rv = IDENT
	case unicode.IsSpace(r):
		rv = WHITESPACE
	case r == '/' && len(unit) > 1:
		rv = COMMENT
	}
	return rv
}

// PackageNameFromFile accepts an io.Reader, the contents of which should be a
// valid proto3 file, and returns the name of the protobuf package for that
// file.
func PackageNameFromFile(protofile io.Reader) (string, error) {
	scanner := svcparse.NewSvcScanner(protofile)
	return GetPackageName(scanner)
}

// GetPackageName accepts a Scanner for a protobuf file and returns the name of
// the protobuf package that the file lives within.
func GetPackageName(scanner Scanner) (string, error) {
	foundpackage := false

	// A nice way to ignore comments. Recursively calls itself until it
	// recieves a unit from the scanner which is not a comment.
	var readIgnoreComment func(Scanner) (Token, []rune, error)
	readIgnoreComment = func(scn Scanner) (Token, []rune, error) {
		unit, err := scanner.ReadUnit()
		if err != nil {
			return OTHER, nil, err
		}
		tkn := categorize(unit)
		if tkn == COMMENT {
			return readIgnoreComment(scn)
		}
		return tkn, unit, err
	}

	for {
		tkn, unit, err := readIgnoreComment(scanner)
		// Err may only be io.EOF
		if err != nil {
			return "", err
		}
		if foundpackage {
			if tkn == IDENT {
				return string(unit), nil
			} else if tkn == WHITESPACE {
				continue
			} else {
				foundpackage = false
			}
		} else {
			if tkn == IDENT && string(unit) == "package" {
				foundpackage = true
			}
		}
	}
}
