
NAME=gokit-base

PLUGIN=protoc-gen-$(NAME)

OUTPUT=./generate

GENERATED_OUT_PATH=./generate/gokit-base-cmd

all: build-generated-code

gobindata:
	go-bindata -o template.go template/...

$(PLUGIN): gobindata
	go build

code-generation: $(PLUGIN)
	mkdir ./generate
	protoc -I/usr/local/include -I. -I.. -I$(GOPATH)/src --plugin=$(PLUGIN) --$(NAME)_out=logtostderr=true:$(OUTPUT) service.proto

build-generated-code: code-generation
	go build $(GENERATED_OUT_PATH)

clean:
	rm -f $(PLUGIN)
	rm -rf $(OUTPUT)
