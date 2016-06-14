
NAME=commenttest

INPUT=comment.go
OUTPUT=result.log


PLUGIN=protoc-gen-$(NAME)


all: $(OUTPUT)

$(PLUGIN): $(INPUT)
	go build -o $@ $^

$(OUTPUT): $(PLUGIN)
	protoc -I/usr/local/include -I. -I.. --plugin=$(PLUGIN) --$(NAME)_out=. service.proto

clean:
	rm -f $(OUTPUT)
	rm -f $(PLUGIN)

