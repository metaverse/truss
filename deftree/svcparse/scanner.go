package svcparse

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"unicode"
)

type RuneReader struct {
	Contents   []rune
	ContentLen int
	RunePos    int
	LineNo     int
}

func (self *RuneReader) ReadRune() (rune, error) {
	var toret rune = 0
	var err error

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

func isIdent(r rune) bool {
	switch {
	case unicode.IsLetter(r):
		return true
	case unicode.IsDigit(r):
		return true
	case r == '_':
		return true
	default:
		return false
	}
}

type ScanUnit struct {
	InRpcDefinition bool
	InRpcBody       bool
	BraceLevel      int
	LineNo          int
	Value           []rune
}

func (self ScanUnit) String() string {
	cleanval := strings.Replace(string(self.Value), "\n", "\\n", -1)
	cleanval = strings.Replace(cleanval, "\t", "\\t", -1)
	cleanval = strings.Replace(cleanval, "\"", "\\\"", -1)
	return fmt.Sprintf(`{"value": "%v", "InRpcDefinition": %v, "InRpcBody": %v, "BraceLevel": %v, "LineNo": %v},`, cleanval, self.InRpcDefinition, self.InRpcBody, self.BraceLevel, self.LineNo)
}

var (
	withinBody bool = false
	withinDef  bool = false
	braceLev   int  = 0
)

func BuildScanUnit(rr *RuneReader) (*ScanUnit, error) {
	rv := &ScanUnit{
		false,
		false,
		0,
		1,
		[]rune{},
	}
	var ch rune
	buf := make([]rune, 0)
	setReturn := func() *ScanUnit {
		rv.BraceLevel = braceLev
		rv.LineNo = rr.LineNo
		rv.InRpcBody = withinBody
		rv.InRpcDefinition = withinDef
		rv.Value = buf
		return rv
	}

	// Populate the buffer with at least one rune so even if it's an unknown
	// character it will at least return this
	ch, err := rr.ReadRune()
	if err != nil {
		return setReturn(), err
	}
	buf = append(buf, ch)

	switch {
	case ch == '/':
		// Searching for comments beginning with '/'
		ch, err = rr.ReadRune()
		if err != nil {
			return setReturn(), err
		} else if ch == '/' {
			// Handle single line comments of the form '//'
			buf = append(buf, ch)
			for {
				ch, err = rr.ReadRune()
				if err != nil {
					return setReturn(), err
				} else if ch == '\n' {
					buf = append(buf, ch)
					return setReturn(), nil
				}
				buf = append(buf, ch)
			}
		} else if ch == '*' {
			// Handle (potentially) multi-line comments of the form '/**/'
			buf = append(buf, ch)
			for {
				ch, err = rr.ReadRune()
				if err != nil {
					return setReturn(), err
				} else if ch == '*' {
					buf = append(buf, ch)
					ch, err = rr.ReadRune()
					if err != nil {
						return setReturn(), err
					} else if ch == '/' {
						buf = append(buf, ch)
						return setReturn(), nil
					}
				} else {
					// Add the body of the comment to the buffer
					buf = append(buf, ch)
				}
			}
		} else {
			// Not a comment, so unread the last Rune and return this '/' only
			rr.UnreadRune()
			return setReturn(), nil
		}
	case ch == '"':
		// Handle strings
		for {
			ch, err = rr.ReadRune()
			if err != nil {
				return setReturn(), err
			} else if ch == '\\' {
				// Handle escape sequences within strings
				buf = append(buf, ch)
				ch, err = rr.ReadRune()
				if err != nil {
					return setReturn(), err
				} else {
					buf = append(buf, ch)
				}
			} else if ch == '"' {
				// Closing quotation
				buf = append(buf, ch)
				return setReturn(), nil
			} else {
				buf = append(buf, ch)
			}
		}
	case unicode.IsSpace(ch):
		// Group consecutive white space characters
		for {
			ch, err = rr.ReadRune()
			if err != nil {
				// Don't pass along this EOF since we did find a valid 'Unit'
				// to return. This way, the next call of this function will
				// return EOF and nothing else, a more clear behavior.
				if err == io.EOF {
					return setReturn(), nil
				}
				return setReturn(), err
			} else if !unicode.IsSpace(ch) {
				rr.UnreadRune()
				break
			}
			buf = append(buf, ch)
		}
	case isIdent(ch):
		// Group consecutive letters
		for {
			ch, err = rr.ReadRune()
			if err != nil {
				if err == io.EOF {
					return setReturn(), nil
				}
				return setReturn(), err
			} else if !isIdent(ch) {
				rr.UnreadRune()
				if string(buf) == "service" {
					withinDef = true
				}
				break
			}
			buf = append(buf, ch)
		}
	case ch == '{':
		braceLev += 1
		if withinDef == true {
			withinDef = false
			withinBody = true
		}
	case ch == '}':
		braceLev -= 1
		if withinBody == true && braceLev == 0 {
			withinBody = false
		}
	}

	// Implicitly, everything that's not a group of letters, not a group of
	// whitespace, not a comment, and not a string-literal, (common examples of
	// runes in this category are numbers and symbols like '&' or '9') will be
	// returned one rune at a time.

	return setReturn(), nil
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
	Buf          []*ScanUnit
	UnitPos      int
	lineNo       int
}

func NewSvcScanner(r io.Reader) *SvcScanner {
	b := make([]*ScanUnit, 0)
	rr := NewRuneReader(r)
	for {
		unit, err := BuildScanUnit(rr)
		if err == nil {
			b = append(b, unit)
		} else {
			break
		}
	}
	return &SvcScanner{
		R:            NewRuneReader(r),
		InDefinition: false,
		InBody:       false,
		BraceLevel:   0,
		Buf:          b,
		UnitPos:      0,
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
			self.UnitPos -= 1
			break
		}
	}
	return nil
}

// ReadUnit returns the next "group" of runes found in the input stream. If the
// end of the stream is reached, io.EOF will be returned as error. No other
// errors will be returned.
func (self *SvcScanner) ReadUnit() ([]rune, error) {
	var rv []rune
	var err error
	if self.UnitPos < len(self.Buf) {
		unit := self.Buf[self.UnitPos]

		self.InBody = unit.InRpcBody
		self.InDefinition = unit.InRpcDefinition
		self.BraceLevel = unit.BraceLevel
		self.lineNo = unit.LineNo

		rv = unit.Value

		self.UnitPos += 1
	} else {
		err = io.EOF
	}

	return rv, err
}

func (self *SvcScanner) UnreadUnit() error {
	if self.UnitPos == 0 {
		return fmt.Errorf("Cannot unread when scanner is at start of input")
	}
	// If we're on the first unit, Unreading means setting the state of the
	// scanner back to it's defaults.
	if self.UnitPos == 1 {
		self.UnitPos = 0
		self.InBody = false
		self.InDefinition = false
		self.BraceLevel = 0
		self.lineNo = 0
	}
	self.UnitPos -= 1

	// Since the state of the scanner usually tracks one behind the `unit`
	// indicated by `UnitPos` we further subtract one when selecting the unit
	// to reflect the state of
	unit := self.Buf[self.UnitPos-1]
	self.InBody = unit.InRpcBody
	self.InDefinition = unit.InRpcDefinition
	self.BraceLevel = unit.BraceLevel
	self.lineNo = unit.LineNo

	return nil
}
func (self *SvcScanner) UnReadToPosition(position int) error {
	for {
		if self.UnitPos != position {
			err := self.UnreadUnit()
			if err != nil {
				return err
			}
		} else {
			break
		}
	}
	return nil
}

func (self *SvcScanner) GetLineNumber() int {
	return self.lineNo
}
