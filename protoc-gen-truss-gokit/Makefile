
NAME=truss-gokit

PLUGIN=protoc-gen-$(NAME)

PLUGIN_CMD_DIR=./cmd/$(PLUGIN)

OUTPUT=./generate
TRUSS_GOKIT_GEN_DIR=service
NON_EDITABLE_OUTPUT=./$(TRUSS_GOKIT_GEN_DIR)/DONOTEDIT

GENERATED_SERVICE=svc
GENERATED_SERVICE_CLIENT=cliclient

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

PROTOFILES=*.proto

all: build-generated-code

# -ignore swp ignores vim swap files
gobindata:
	go-bindata -pkg template -o ./template/template.go -ignore swp template_files/...

$(PLUGIN): gobindata
	go install ./cmd/protoc-gen-truss-gokit

code-generation: $(PLUGIN)
	protoc -I/usr/local/include -I. \
     		-I$(GOPATH)/src/$(GOOGLE_HTTP_API_DIR) \
		   --$(NAME)_out=logtostderr=true:. \
		   $(PROTOFILES)
	mkdir -p $(NON_EDITABLE_OUTPUT)/pb
	mkdir -p $(NON_EDITABLE_OUTPUT)/third_party/googleapis
	cp -r $(GOPATH)/src/$(GOOGLE_HTTP_API_DIR) $(NON_EDITABLE_OUTPUT)/third_party/googleapis/
	protoc  -I/usr/local/include -I. \
     		-I$(GOPATH)/src/$(GOOGLE_HTTP_API_DIR) \
     		--go_out=Mgoogle/api/annotations.proto=$(GOOGLE_HTTP_API_DIR)/google/api,plugins=grpc:$(NON_EDITABLE_OUTPUT)/pb \
			$(PROTOFILES)

build-generated-code: code-generation
	@echo -e '$(BUILD_GENERATE_MSG)'
	go build $(GENERATED_OUT_BIN_DIR)/...
	go build $(GENERATED_CLIENT_OUT_BIN_DIR)/...

clean:
	rm -f $(PLUGIN)
	rm -f $(GENERATED_SERVICE)
	rm -f $(GENERATED_SERVICE_CLIENT)
	rm -rf $(TRUSS_GOKIT_GEN_DIR)

test:
	git --no-pager diff --color=always --no-index original/ $(OUTPUT)
