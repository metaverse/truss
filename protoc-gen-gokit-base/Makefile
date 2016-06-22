
NAME=gokit-base

PLUGIN=protoc-gen-$(NAME)

OUTPUT=./generate

all: output

$(PLUGIN):
	go build

output: $(PLUGIN)
	mkdir ./generate
	protoc -I/usr/local/include -I. -I.. -I$(GOPATH)/src --plugin=$(PLUGIN) --$(NAME)_out=$(OUTPUT) service.proto

clean:
	rm -f $(PLUGIN)
	rm -rf $(OUTPUT)
