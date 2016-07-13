
NAME=gokit-base

PLUGIN=protoc-gen-$(NAME)

PLUGIN_CMD_DIR=./cmd/$(PLUGIN)

OUTPUT=./generate
NON_EDITABLE_OUTPUT=$(OUTPUT)/DONOTEDIT

GENERATED_SERVICE=addsvc
GENERATED_SERVICE_CLIENT=addcli

GENERATED_OUT_BIN_DIR=$(NON_EDITABLE_OUTPUT)/cmd/$(GENERATED_SERVICE)
GENERATED_CLIENT_OUT_BIN_DIR=$(NON_EDITABLE_OUTPUT)/cmd/$(GENERATED_SERVICE_CLIENT)


# GOOGLE_HTTP_API_DIR Must contain the following directory structure
# google/
# └── api
#     ├── annotations.pb.go
#     ├── annotations.proto
#     ├── http.pb.go
#     └── http.proto
GOOGLE_HTTP_API_DIR=github.com/TuneLab/gob/third_party/googleapis

# Debug colors and messages
NO_COLOR=\e[0m
OK_COLOR=\e[38;5;118m

RUN_MSG=\n$(OK_COLOR)PLUGIN BUILT... GENERATING SERVICE:$(NO_COLOR)\n
BUILD_GENERATE_MSG=\n$(OK_COLOR)SERVICED GENERATED... BUILDING:$(NO_COLOR)\n

PROTOFILES=pb/*.proto

all: build-generated-code

# -ignore swp ignores vim swap files
gobindata:
	go-bindata -pkg template -o ./template/template.go -ignore swp template_files/...

$(PLUGIN): gobindata
	go build ./cmd/protoc-gen-gokit-base

code-generation: $(PLUGIN)
	mkdir -p $(NON_EDITABLE_OUTPUT)
	mkdir -p $(NON_EDITABLE_OUTPUT)/pb
	protoc  -I/usr/local/include -I. \
	   		-I.. \
			-I$(GOPATH)/src \
     		-I$(GOPATH)/src/$(GOOGLE_HTTP_API_DIR) \
     		--go_out=Mgoogle/api/annotations.proto=$(GOOGLE_HTTP_API_DIR)/google/api,plugins=grpc:$(NON_EDITABLE_OUTPUT)/ \
			$(PROTOFILES)
	@echo -e '$(RUN_MSG)'
	# The plugin is copied into the output directory so that it can be called from within that directory
	cp $(PLUGIN) $(OUTPUT)
	# Currently the plugin assumes that the output directory of the plugin is the current directory (.), otherwise it will fail
	mkdir -p $(OUTPUT)/pb
	cp $(PROTOFILES) $(OUTPUT)/pb/
	cd $(OUTPUT) && \
	protoc -I/usr/local/include -I. \
     	   -I.. \
	   	   -I$(GOPATH)/src \
     		-I$(GOPATH)/src/$(GOOGLE_HTTP_API_DIR) \
		   --plugin=$(PLUGIN) \
		   --$(NAME)_out=logtostderr=true:. \
		   $(PROTOFILES)

build-generated-code: code-generation
	@echo -e '$(BUILD_GENERATE_MSG)'
	go build $(GENERATED_OUT_BIN_DIR)/...
	go build $(GENERATED_CLIENT_OUT_BIN_DIR)/...

clean:
	rm -f $(PLUGIN)
	rm -rf $(OUTPUT)
	rm -f $(GENERATED_SERVICE)
	rm -f $(GENERATED_SERVICE_CLIENT)

test:
	git --no-pager diff --color=always --no-index original/ $(OUTPUT)
