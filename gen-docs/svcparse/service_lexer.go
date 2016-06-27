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
)

type RuneReader struct {
	Contents   []rune
	ContentLen int
	RunePos    int
	LineNo     int
	BraceLevel int
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
	case '{':
		self.BraceLevel -= 1
	case '}':
		self.BraceLevel += 1
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
}

func NewSvcScanner(r io.Reader) *SvcScanner {
	return &SvcScanner{
		R:            NewRuneReader(r),
		InDefinition: false,
		InBody:       false,
	}
}

func (self *SvcScanner) FastForward() error {
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

// Reads in a rough "syntactical unit" of a Protobuf file. Returns strings
// containing groups of letters, groups of whitespace, and entire comments.
// Every other type of unit is returned one character at a time.
func (self *SvcScanner) ReadUnit() ([]rune, error) {
	var ch rune
	buf := make([]rune, 0)

	// Shorthand function for reading a rune into the `ch` variable
	ch, err := self.R.ReadRune()
	if err != nil {
		//fmt.Fprintf(os.Stderr, "Error reading rune at beginning of ReadUnit: '%v'\n", err)
		return buf, err
	}
	buf = append(buf, ch)

	if ch == '/' {
		buf = append(buf, ch)
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
	} else if ch == '"' {
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
	} else if unicode.IsSpace(ch) {
		// Group consecutive spaces
		for {
			ch, err = self.R.ReadRune()
			if err != nil {
				return buf, err
			} else if !unicode.IsSpace(ch) {
				self.R.UnreadRune()
				break
			}
			buf = append(buf, ch)
		}
	} else if unicode.IsLetter(ch) {
		// Group consecutive letters
		for {
			ch, err = self.R.ReadRune()
			if err != nil {
				return buf, err
			} else if !unicode.IsLetter(ch) {
				self.R.UnreadRune()
				break
			}
			buf = append(buf, ch)
		}
	}
	// Implicitly, everything that's not a group of letters, not a group of
	// whitespace, not a comment, and not a string-literal, (common examples of
	// runes in this category are numbers and symbols like '&' or '9') will be
	// returned one rune at a time.

	return buf, nil
}

func main() {
	r := strings.NewReader("what\nservice service Test{}")
	scn := NewSvcScanner(r)
	//scn.FastForward()
	//out, _ := scn.R.Peek(9)
	for i := 0; i < 10; i++ {
		out, err := scn.ReadUnit()
		if err != nil {
			break
		}
		fmt.Printf("%v unit: '%v'\n", i, string(out))
	}
	//fmt.Printf("Last %v bytes of output: %v\n", len(out), string(out))

}
