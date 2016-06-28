package svcparse

//package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"
)

type RuneReader struct {
	Contents   []rune
	ContentLen int
	RunePos    int
	LineNo     int
}

func (self *RuneReader) ReadRune() (rune, error) {
	var toret rune = 0
	var err error = nil

	if self.RunePos < self.ContentLen {
		toret = self.Contents[self.RunePos]
		if toret == '\n' {
			self.LineNo += 1
		}
		self.RunePos += 1
	} else {
		err = io.EOF
	}
	return toret, err
}

func (self *RuneReader) UnreadRune() error {
	if self.RunePos == 0 {
		return bufio.ErrInvalidUnreadRune
	}
	self.RunePos -= 1
	switch self.Contents[self.RunePos] {
	case '\n':
		self.LineNo -= 1
	}
	return nil
}

func NewRuneReader(r io.Reader) *RuneReader {
	//contents := bytes.Runes(ioutil.ReadAll(r))
	b, _ := ioutil.ReadAll(r)
	contents := bytes.Runes(b)
	return &RuneReader{
		Contents:   contents,
		ContentLen: len(contents),
		RunePos:    0,
		LineNo:     1,
	}
}

// Service scanner conducts many of the basic scanning operatiions of a Lexer,
// with some additional service-specific behavior.
//
// Since this scanners is specifically for scanning the Protobuf service
// definitions, it will only scan the sections of the input from the reader
// that it believes are part of a service definition. This means that it will
// "fast forward" through its input reader until it finds the start of a
// service definition. It will keep track of braces (the "{}" characters) till
// it finds the final closing brace marking the end of the service definition.
type SvcScanner struct {
	R            *RuneReader
	InDefinition bool
	InBody       bool
	BraceLevel   int
}

func NewSvcScanner(r io.Reader) *SvcScanner {
	return &SvcScanner{
		R:            NewRuneReader(r),
		InDefinition: false,
		InBody:       false,
		BraceLevel:   0,
	}
}

// FastForward will move the current position of the internal RuneReader to the
// beginning of the next service definition. If the scanner is in the middle of
// an existing service definition, this method will do nothing.
func (self *SvcScanner) FastForward() error {
	if self.InBody || self.InDefinition {
		return nil
	}
	search_str := string("service")
	for {
		buf, err := self.ReadUnit()
		if err != nil {
			return err
		}
		if string(buf) == search_str {
			for i := 0; i < len(search_str); i++ {
				err = self.R.UnreadRune()
				if err != nil {
					fmt.Fprintf(os.Stderr, "%v Error unreading: %v\n", i, err)
					return err
				}
			}
			break
		}
	}
	return nil
}

// Returns one rough "syntactical unit" of a Protobuf file. Returns strings
// containing groups of letters, groups of whitespace, and entire comments.
// Every other type of unit is returned one character at a time.
func (self *SvcScanner) ReadUnit() ([]rune, error) {
	var ch rune
	buf := make([]rune, 0)

	// Populate the buffer with at least one rune so even if it's an unknown
	// character it will at least return this
	ch, err := self.R.ReadRune()
	if err != nil {
		return buf, err
	}
	buf = append(buf, ch)

	switch {
	case ch == '/':
		// Searching for comments beginning with '/'
		ch, err = self.R.ReadRune()
		if err != nil {
			return buf, err
		} else if ch == '/' {
			// Handle single line comments of the form '//'
			buf = append(buf, ch)
			for {
				ch, err = self.R.ReadRune()
				if err != nil {
					return buf, err
				} else if ch == '\n' {
					buf = append(buf, ch)
					return buf, nil
				}
				buf = append(buf, ch)
			}
		} else if ch == '*' {
			// Handle (potentially) multi-line comments of the form '/**/'
			buf = append(buf, ch)
			for {
				ch, err = self.R.ReadRune()
				if err != nil {
					return buf, err
				} else if ch == '*' {
					buf = append(buf, ch)
					ch, err = self.R.ReadRune()
					if err != nil {
						return buf, err
					} else if ch == '/' {
						buf = append(buf, ch)
						return buf, nil
					}
				}
			}
		} else {
			// Not a comment, so unread the last Rune and return this '/' only
			self.R.UnreadRune()
			return buf, nil
		}
	case ch == '"':
		// Handle strings
		buf = append(buf, ch)
		for {
			ch, err = self.R.ReadRune()
			if err != nil {
				return buf, err
			} else if ch == '\\' {
				// Handle escape sequences within strings
				buf = append(buf, ch)
				ch, err = self.R.ReadRune()
				if err != nil {
					return buf, err
				} else {
					buf = append(buf, ch)
				}
			} else if ch == '"' {
				// Closing quotation
				buf = append(buf, ch)
				return buf, nil
			}
		}
	case unicode.IsSpace(ch):
		// Group consecutive white space characters
		for {
			ch, err = self.R.ReadRune()
			if err != nil {
				// Don't pass along this EOF since we did find a valid 'Unit'
				// to return. This way, the next call of this function will
				// return EOF and nothing else, a more clear behavior.
				if err == io.EOF {
					return buf, nil
				}
				return buf, err
			} else if !unicode.IsSpace(ch) {
				self.R.UnreadRune()
				break
			}
			buf = append(buf, ch)
		}
	case unicode.IsLetter(ch):
		// Group consecutive letters
		for {
			ch, err = self.R.ReadRune()
			if err != nil {
				if err == io.EOF {
					return buf, nil
				}
				return buf, err
			} else if !unicode.IsLetter(ch) {
				self.R.UnreadRune()
				if string(buf) == "service" {
					self.InDefinition = true
				}
				break
			}
			buf = append(buf, ch)
		}
	case ch == '{':
		self.BraceLevel += 1
		if self.InDefinition == true {
			self.InDefinition = false
			self.InBody = true
		}
	case ch == '}':
		self.BraceLevel -= 1
		if self.InBody == true && self.BraceLevel == 0 {
			self.InBody = false
		}
	}

	// Implicitly, everything that's not a group of letters, not a group of
	// whitespace, not a comment, and not a string-literal, (common examples of
	// runes in this category are numbers and symbols like '&' or '9') will be
	// returned one rune at a time.

	return buf, nil
}

type SvcLexer struct {
	Scn *SvcScanner
}

func NewSvcLexer(r io.Reader) *SvcLexer {
	return &SvcLexer{
		Scn: NewSvcScanner(r),
	}
}

func (self *SvcLexer) GetToken() (Token, string) {
	// Since FastForward won't take us out of a service definition we're
	// already within, we can safely call it every time we attempt to get a
	// token
	unit, err := self.Scn.ReadUnit()

	if err != nil {
		if err == io.EOF {
			return EOF, string(unit)
		} else {
			panic(err)
		}
	}
	switch {
	case len(unit) == 0:
		return ILLEGAL, ""
	case unicode.IsSpace(unit[0]):
		return WHITESPACE, string(unit)
	case unicode.IsLetter(unit[0]):
		return IDENT, string(unit)
	case unit[0] == '"':
		return STRING_LITERAL, string(unit)
	case unit[0] == '(':
		return OPEN_PAREN, string(unit)
	case unit[0] == ')':
		return CLOSE_PAREN, string(unit)
	case unit[0] == '{':
		return OPEN_BRACE, string(unit)
	case unit[0] == '}':
		return CLOSE_BRACE, string(unit)
	case len(unit) > 1 && unit[0] == '/':
		tk, addit_comment := self.buildCommentToken()
		if tk != ILLEGAL {
			return COMMENT, string(unit) + addit_comment
		} else {
			return COMMENT, string(unit)
		}
	case len(unit) == 1:
		return SYMBOL, string(unit)
	default:
		return ILLEGAL, string(unit)
	}
}

// Since a multi-line comment could be composed of many single line comments,
// this method exists to handle such cases.
func (self *SvcLexer) buildCommentToken() (Token, string) {
	one_tk, one_str := self.GetToken()
	// Since the newline at the end of each single-line comment is included
	// within that comment, if there's whitespace between the last comment and
	// the next, but they're on consecutive lines, then there should be 0
	// newlines in the whitespace between them.
	if one_tk == WHITESPACE && strings.Count(one_str, "\n") == 0 {
		two_tk, two_str := self.GetToken()
		if two_tk == COMMENT {
			return COMMENT, one_str + two_str
		} else {
			for i := 0; i < utf8.RuneCountInString(one_str+two_str); i++ {
				self.Scn.R.UnreadRune()
			}
			return ILLEGAL, ""
		}
	} else if one_tk == COMMENT {
		return COMMENT, one_str
	} else {
		for i := 0; i < utf8.RuneCountInString(one_str); i++ {
			self.Scn.R.UnreadRune()
		}
		return ILLEGAL, ""
	}
}

func (self *SvcLexer) GetTokenIgnoreWhitespace() (Token, string) {
	return self.getTokenIgnore(WHITESPACE)
}

func (self *SvcLexer) getTokenIgnore(to_ignore Token) (Token, string) {
	for {
		t, s := self.GetToken()
		if t != to_ignore {
			return t, s
		}
	}
}
