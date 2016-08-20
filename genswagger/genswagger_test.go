package genswagger

import (
	"testing"

	"github.com/TuneLab/go-truss/genswagger/testprotodata"
	"github.com/TuneLab/go-truss/testproto"
	spew "github.com/davecgh/go-spew/spew"
	"os"
)

var testData testproto.TestDoctreeBuilder
var _ = spew.Dump
var _ = os.DevNull

func init() {
	testData = testproto.New(testprotodata.Asset)
}

func TestGenSwagSchemaProperties(t *testing.T) {
	messages, err := testData.AssetAsSliceServiceMethods("general.proto")
	_ = messages
	if err != nil {
		t.Error(err)
	}

}

func TestGenSwagParametersQuery(t *testing.T) {
	meths, _ := testData.AssetAsSliceServiceMethods("only-query-params.proto")
	meth := meths[0]
	httpbind := meth.HttpBindings[0]

	swagParams := genSwagParameters(meth, httpbind)
	//spew.Fdump(os.Stderr, swagParams)

	param := swagParams[0]

	if got, want := param.Type, "string"; got != want {
		t.Errorf("param type differ;\ngot  = %+v\nwant = %+v\n", got, want)
	}
}
