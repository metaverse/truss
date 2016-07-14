package main

import (
	"fmt"
	"os"

	"github.com/TuneLab/gob/gendoc/doctree"
	"github.com/TuneLab/gob/gendoc/svcparse"
)

func main() {
	file, _ := os.Open("test00.proto")

	lex := svcparse.NewSvcLexer(file)

	svc, err := svcparse.ParseService(lex)
	if err != nil {
		panic(err)
	}

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
	parent.SetName("Currency Exchange example")
	parent.Files[0].SetName("dig.pb")
	fmt.Printf("%v\n", parent.Markdown())

	return
	// Print remaining tokens
	for {
		tk, str := lex.GetTokenIgnoreWhitespace()
		fmt.Printf("Token: '%15v', str: '%v'\n", tk, str)
		if tk == svcparse.ILLEGAL || tk == svcparse.EOF {
			break
		}
	}
}
