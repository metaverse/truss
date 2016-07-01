package svcparse

import (
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"
)

func NewTokenGroup(scn *SvcScanner) *TokenGroup {
	// Since FastForward won't take us out of a service definition we're
	// already within, we can safely call it every time we attempt to get a
	// token
	err := scn.FastForward()
	if err != nil {
		if err == io.EOF {
			return &TokenGroup{EOF, "", scn.R.LineNo}
		} else {
			return &TokenGroup{ILLEGAL, fmt.Sprint(err), scn.R.LineNo}
		}
	}
	unit, err := scn.ReadUnit()

	if err != nil {
		if err == io.EOF {
			return &TokenGroup{EOF, string(unit), scn.R.LineNo}
		} else {
			return &TokenGroup{ILLEGAL, fmt.Sprint(err), scn.R.LineNo}
		}
	}
	switch {
	case len(unit) == 0:
		return &TokenGroup{ILLEGAL, "", scn.R.LineNo}
	case unicode.IsSpace(unit[0]):
		return &TokenGroup{WHITESPACE, string(unit), scn.R.LineNo}
	case isIdent(unit[0]):
		return &TokenGroup{IDENT, string(unit), scn.R.LineNo}
	case unit[0] == '"':
		return &TokenGroup{STRING_LITERAL, string(unit), scn.R.LineNo}
	case unit[0] == '(':
		return &TokenGroup{OPEN_PAREN, string(unit), scn.R.LineNo}
	case unit[0] == ')':
		return &TokenGroup{CLOSE_PAREN, string(unit), scn.R.LineNo}
	case unit[0] == '{':
		return &TokenGroup{OPEN_BRACE, string(unit), scn.R.LineNo}
	case unit[0] == '}':
		return &TokenGroup{CLOSE_BRACE, string(unit), scn.R.LineNo}
	case len(unit) > 1 && unit[0] == '/':
		tk, addit_comment := buildCommentToken(scn)
		if tk != ILLEGAL {
			return &TokenGroup{COMMENT, string(unit) + addit_comment, scn.R.LineNo}
		} else {
			return &TokenGroup{COMMENT, string(unit), scn.R.LineNo}
		}
	case len(unit) == 1:
		return &TokenGroup{SYMBOL, string(unit), scn.R.LineNo}
	default:
		return &TokenGroup{ILLEGAL, string(unit), scn.R.LineNo}
	}
}

// Since a multi-line comment could be composed of many single line comments,
// this method exists to handle such cases.
func buildCommentToken(scn *SvcScanner) (Token, string) {
	onegrp := NewTokenGroup(scn)
	// Since the newline at the end of each single-line comment is included
	// within that comment, if there's whitespace between the last comment and
	// the next, but they're on consecutive lines, then there should be 0
	// newlines in the whitespace between them.
	if onegrp.token == WHITESPACE && strings.Count(onegrp.value, "\n") == 0 {
		twogrp := NewTokenGroup(scn)
		if twogrp.token == COMMENT {
			return COMMENT, onegrp.value + twogrp.value
		} else {
			for i := 0; i < utf8.RuneCountInString(onegrp.value+twogrp.value); i++ {
				scn.R.UnreadRune()
			}
			return ILLEGAL, ""
		}
	} else if onegrp.token == COMMENT {
		return COMMENT, onegrp.value
	} else {
		for i := 0; i < utf8.RuneCountInString(onegrp.value); i++ {
			scn.R.UnreadRune()
		}
		return ILLEGAL, ""
	}
}

type SvcLexer struct {
	Scn    *SvcScanner
	Buf    []*TokenGroup
	tkPos  int
	lineNo int
}

func NewSvcLexer(r io.Reader) *SvcLexer {
	b := make([]*TokenGroup, 0)
	scn := NewSvcScanner(r)
	for {
		grp := NewTokenGroup(scn)
		if grp.token != ILLEGAL && grp.token != EOF {
			b = append(b, grp)
		} else {
			break
		}
	}
	return &SvcLexer{
		Scn:   scn,
		Buf:   b,
		tkPos: 0,
	}
}

func (self *SvcLexer) GetToken() (Token, string) {
	var tk Token
	var val string

	if self.tkPos < len(self.Buf) {
		grp := self.Buf[self.tkPos]
		self.lineNo = grp.line
		tk = grp.token
		val = grp.value

		self.tkPos += 1
	} else {
		tk = EOF
	}

	return tk, val
}

func (self *SvcLexer) UnGetToken() error {
	if self.tkPos == 0 {
		return fmt.Errorf("Cannot unread when Lexer is at start of input")
	}
	self.tkPos -= 1
	self.lineNo = self.Buf[self.tkPos].line
	return nil
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
