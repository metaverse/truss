package main

import (
	"fmt"
	"os"

	"github.com/TuneLab/gob/gen-docs/doctree"
	"github.com/TuneLab/gob/gen-docs/svcparse"
	"github.com/davecgh/go-spew/spew"
)

func main() {
	file, _ := os.Open("test00.proto")

	lex := svcparse.NewSvcLexer(file)

	svc, err := svcparse.ParseService(lex)
	if err != nil {
		panic(err)
	}

	// Dump basic ast
	out := spew.Sdump(svc)
	fmt.Printf("%v\n", out)

	// Couch created AST within parent objects in order to call the "String"
	// method which calls the internal only 'describe' method
	parent := &doctree.MicroserviceDefinition{
		Files: []*doctree.ProtoFile{
			&doctree.ProtoFile{
				Services: []*doctree.ProtoService{
					svc,
				},
			},
		},
	}
	fmt.Printf("%v\n", parent)

	// Print remaining tokens
	for {
		tk, str := lex.GetTokenIgnoreWhitespace()
		fmt.Printf("Token: '%15v', str: '%v'\n", tk, str)
		if tk == svcparse.ILLEGAL || tk == svcparse.EOF {
			break
		}
	}
}
