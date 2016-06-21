
NAME=commenttest

INPUT=comment.go
OUTPUT=result.log


PLUGIN=protoc-gen-$(NAME)


all: $(OUTPUT)

$(PLUGIN): $(INPUT)
	go build -o $@ $^

$(OUTPUT): $(PLUGIN)
	protoc -I/usr/local/include -I. -I.. --plugin=$(PLUGIN) --$(NAME)_out=. service.proto

protoc-gen-buildast: build_ast.go
	go build -o $@ $^

ast.log: protoc-gen-buildast
	protoc -I/usr/local/include -I. -I.. --plugin=protoc-gen-buildast --buildast_out=. service.proto

clean:
	rm -f $(OUTPUT)
	rm -f $(PLUGIN)
	rm -f protoc-gen-buildast
	rm -f ast.log

