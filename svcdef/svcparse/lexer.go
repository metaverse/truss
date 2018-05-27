package svcparse

import (
	"fmt"
	"io"
	"strings"
	"unicode"
)

func NewTokenGroup(scn *SvcScanner) *TokenGroup {
	// Since FastForward won't take us out of a service definition we're
	// already within, we can safely call it every time we attempt to get a
	// token
	if scn.BraceLevel == 0 {
		err := scn.FastForward()
		if err != nil {
			if err == io.EOF {
				return &TokenGroup{EOF, "", scn.GetLineNumber()}
			} else {
				return &TokenGroup{ILLEGAL, fmt.Sprint(err), scn.GetLineNumber()}
			}
		}
	}
	unit, err := scn.ReadUnit()

	if err != nil {
		if err == io.EOF {
			return &TokenGroup{EOF, string(unit), scn.GetLineNumber()}
		} else {
			return &TokenGroup{ILLEGAL, fmt.Sprint(err), scn.GetLineNumber()}
		}
	}
	switch {
	case len(unit) == 0:
		return &TokenGroup{ILLEGAL, "", scn.GetLineNumber()}
	case unicode.IsSpace(unit[0]):
		return &TokenGroup{WHITESPACE, string(unit), scn.GetLineNumber()}
	case isIdent(unit[0]):
		return &TokenGroup{IDENT, string(unit), scn.GetLineNumber()}
	case unit[0] == '"':
		return &TokenGroup{STRING_LITERAL, string(unit), scn.GetLineNumber()}
	case unit[0] == '(':
		return &TokenGroup{OPEN_PAREN, string(unit), scn.GetLineNumber()}
	case unit[0] == ')':
		return &TokenGroup{CLOSE_PAREN, string(unit), scn.GetLineNumber()}
	case unit[0] == '{':
		return &TokenGroup{OPEN_BRACE, string(unit), scn.GetLineNumber()}
	case unit[0] == '}':
		return &TokenGroup{CLOSE_BRACE, string(unit), scn.GetLineNumber()}
	case len(unit) > 1 && unit[0] == '/':
		str := string(unit)
		// Since a multi-line comment could be composed of many single line
		// comments
		for {
			one_pos := scn.UnitPos
			one, err := scn.ReadUnit()
			if err != nil {
				panic(err)
			}
			onestr := string(one)
			if len(one) > 1 && one[0] == '/' {
				str += onestr
			} else if unicode.IsSpace(one[0]) {
				if strings.Count(onestr, "\n") == 0 {
					two, err := scn.ReadUnit()
					if err != nil {
						panic(err)
					}
					twostr := string(two)
					if len(two) > 1 && two[0] == '/' {
						// Ensure that whitespace between multiline comments on
						// a single line are preserved.
						str += onestr + twostr
					} else {
						scn.UnReadToPosition(one_pos)
						break
					}
				} else {
					scn.UnReadToPosition(one_pos)
					break
				}
			} else {
				scn.UnReadToPosition(one_pos)
				break
			}
		}
		return &TokenGroup{COMMENT, str, scn.GetLineNumber()}
	case len(unit) == 1:
		return &TokenGroup{SYMBOL, string(unit), scn.GetLineNumber()}
	default:
		return &TokenGroup{ILLEGAL, string(unit), scn.GetLineNumber()}
	}
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

type SvcLexer struct {
	Scn    *SvcScanner
	Buf    []*TokenGroup
	tkPos  int
	lineNo int
}

func (self *SvcLexer) GetPosition() int {
	return self.tkPos
}

func (self *SvcLexer) GetLineNumber() int {
	return self.lineNo
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

func (self *SvcLexer) UnGetToPosition(position int) error {
	for {
		if self.GetPosition() != position {
			err := self.UnGetToken()
			if err != nil {
				return err
			}
		} else {
			break
		}
	}
	return nil
}
func (self *SvcLexer) GetTokenIgnoreCommentAndWhitespace() (Token, string) {
	for {
		t, s := self.GetToken()
		if t != COMMENT && t != WHITESPACE {
			return t, s
		}
	}
}
func (self *SvcLexer) getTokenIgnore(to_ignore Token) (Token, string) {
	for {
		t, s := self.GetToken()
		if t != to_ignore {
			return t, s
		}
	}
}

func (self *SvcLexer) GetTokenIgnoreWhitespace() (Token, string) {
	return self.getTokenIgnore(WHITESPACE)
}
