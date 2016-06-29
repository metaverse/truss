package svcparse

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
