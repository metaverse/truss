package testproto

import (
	"bytes"

	"github.com/golang/protobuf/proto"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
	"github.com/pkg/errors"

	"github.com/TuneLab/go-truss/gendoc/doctree"
	"github.com/TuneLab/go-truss/gendoc/doctree/makedt"
)

type TestDoctreeBuilder struct {
	asset func(name string) ([]byte, error)
}

func New(assetFunc func(name string) ([]byte, error)) TestDoctreeBuilder {

	var tdb TestDoctreeBuilder
	tdb.asset = assetFunc

	return tdb
}

func (t TestDoctreeBuilder) AssetAsDoctree(name string) (doctree.Doctree, error) {
	proto.Message
	protocOut, err := t.asset("data/definitions/" + name)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to open protoc output of proto file: %v | have you gogenerated?", name)
	}

	codeGenRequest := new(plugin.CodeGeneratorRequest)
	if err = proto.Unmarshal(protocOut, codeGenRequest); err != nil {
		return nil, err
	}

	svcFile, err := t.asset("definitions/" + name)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to open proto file: %v | have you gogenerated?", name)
	}

	svcFileReader := bytes.NewReader(svcFile)

	dt, err := makedt.New(codeGenRequest, svcFileReader)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to create doctree from %v", name)
	}

	return dt, nil
}

func (t TestDoctreeBuilder) AssetAsProtoService(name string) (*doctree.ProtoService, error) {
	dt, err := t.AssetAsDoctree(name)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create doctree")
	}

	md := dt.(*doctree.MicroserviceDefinition)
	files := md.Files
	var service *doctree.ProtoService

	for _, file := range files {
		if len(file.Services) > 0 {
			service = file.Services[0]
		}
	}

	if service == nil {
		return nil, errors.New("no service found")
	}

	return service, nil
}

func (t TestDoctreeBuilder) AssetAsSliceServiceMethods(name string) ([]*doctree.ServiceMethod, error) {
	svc, err := t.AssetAsProtoService(name)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create *doctree.ProtoService")
	}

	return svc.Methods, nil
}

func (t TestDoctreeBuilder) AssetAsSliceMessages(name string) ([]*doctree.ProtoMessage, error) {
	dt, err := t.AssetAsDoctree(name)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create doctree")
	}

	md := dt.(*doctree.MicroserviceDefinition)
	files := md.Files
	var messages []*doctree.ProtoMessage
	msg := messages[0]

	for _, file := range files {
		messages = append(messages, file.Messages...)
	}

	return messages, nil
}
