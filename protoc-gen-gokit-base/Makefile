NO_COLOR=\e[0m
OK_COLOR=\e[38;5;118m

RUN_MSG=\n$(OK_COLOR)PLUGIN BUILT... GENERATING SERVICE:$(NO_COLOR)\n

BUILD_GENERATE_MSG=\n$(OK_COLOR)SERVICED GENERATED... BUILDING:$(NO_COLOR)\n

NAME=gokit-base

PLUGIN=protoc-gen-$(NAME)

PLUGIN_CMD_DIR=./cmd/$(PLUGIN)

OUTPUT=./generate

GENERATED_SERVICE=addsvc
GENERATED_SERVICE_CLIENT=addcli

GENERATED_OUT_DIR=$(OUTPUT)/cmd/$(GENERATED_SERVICE)
GENERATED_CLIENT_OUT_DIR=$(OUTPUT)/cmd/$(GENERATED_SERVICE_CLIENT)

all: build-generated-code

# -ignore swp ignores vim swap files
gobindata:
	go-bindata -pkg template -o ./template/template.go -ignore swp template_files/...

$(PLUGIN): gobindata
	go build ./cmd/protoc-gen-gokit-base

code-generation: $(PLUGIN)
	mkdir -p ./generate
	@echo -e '$(RUN_MSG)'
	protoc -I/usr/local/include -I. -I.. -I$(GOPATH)/src --plugin=$(PLUGIN) --$(NAME)_out=logtostderr=true:$(OUTPUT) service.proto

build-generated-code: code-generation
	@echo -e '$(BUILD_GENERATE_MSG)'
	go build $(GENERATED_OUT_DIR)/...
	go build $(GENERATED_CLIENT_OUT_DIR)/...

clean:
	rm -f $(PLUGIN)
	rm -rf $(OUTPUT)
	rm -f $(GENERATED_SERVICE)
	rm -f $(GENERATED_SERVICE_CLIENT)
