package httpopts

import (
	//"strings"
	dt "github.com/TuneLab/gob/gendoc/doctree"
	"testing"
)

func TestGetPathParams(t *testing.T) {
	binding := &dt.MethodHttpBinding{
		Fields: []*dt.BindingField{
			&dt.BindingField{
				Kind:  "get",
				Value: `"/{a}/{b}"`,
			},
		},
	}
	params := getPathParams(binding)
	t.Log(params)
	if len(params) != 2 {
		t.Fatalf("Params (%v) is length '%v', expected length 2", params, len(params))
	}
}
