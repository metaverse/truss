
NAME=gokit-base

PLUGIN=protoc-gen-$(NAME)

OUTPUT=./generate

all: output

gobindata:
	$(MAKE) -C template gobindataexecute

$(PLUGIN): gobindata
	go build

output: $(PLUGIN)
	mkdir ./generate
	protoc -I/usr/local/include -I. -I.. -I$(GOPATH)/src --plugin=$(PLUGIN) --$(NAME)_out=logtostderr=true:$(OUTPUT) service.proto

clean:
	rm -f $(PLUGIN)
	rm -rf $(OUTPUT)
