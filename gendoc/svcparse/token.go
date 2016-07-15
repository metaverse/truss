package svcparse

import (
	"fmt"
	"strings"
)

type Token int

const (
	ILLEGAL Token = iota
	EOF
	WHITESPACE
	COMMENT
	SYMBOL

	IDENT

	STRING_LITERAL

	OPEN_PAREN
	CLOSE_PAREN

	OPEN_BRACE
	CLOSE_BRACE
)

var tokenLookup = map[Token]string{
	ILLEGAL:    `ILLEGAL`,
	EOF:        `EOF`,
	WHITESPACE: `WHITESPACE`,
	COMMENT:    `COMMENT`,
	SYMBOL:     `SYMBOL`,

	IDENT: `IDENT`,

	STRING_LITERAL: `STRING_LITERAL`,

	OPEN_PAREN:  `OPEN_PAREN`,
	CLOSE_PAREN: `CLOSE_PAREN`,

	OPEN_BRACE:  `OPEN_BRACE`,
	CLOSE_BRACE: `CLOSE_BRACE`,
}

func (self Token) String() string {
	return tokenLookup[self]
}

type TokenGroup struct {
	token Token
	value string
	line  int
}

func (self TokenGroup) String() string {
	cleanval := strings.Replace(self.value, "\n", "\\n", -1)
	cleanval = strings.Replace(cleanval, "\t", "\\t", -1)
	cleanval = strings.Replace(cleanval, "\"", "\\\"", -1)
	return fmt.Sprintf(`{"token": "%v", "value": "%v", "line": %v},`, self.token, cleanval, self.line)
}
