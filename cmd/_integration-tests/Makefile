NO_COLOR=\e[0m
OK_COLOR=\e[38;5;118m
UNDER=\n________________________________________________________________________________\n
END_COLOR_LINE=$(UNDER)$(NO_COLOR)

TRANSPORT_TEST_MSG=\n$(OK_COLOR)Starting transport end to end test:$(END_COLOR_LINE)

CLI_TEST_MSG=\n$(OK_COLOR)Start server and cliclient generate, build, and run test:$(END_COLOR_LINE)

export PATH := $(CURDIR):$(PATH)

all: test

test: clean test-transport test-cli

local-tools:
	go build -o protoc-gen-truss-protocast github.com/TuneLab/go-truss/cmd/protoc-gen-truss-protocast
	go build -o truss github.com/TuneLab/go-truss/cmd/truss

test-transport: local-tools
	@which truss
	@printf '$(TRANSPORT_TEST_MSG)'
	$(MAKE) -C transport

test-cli: local-tools
	@which truss
	@printf '$(CLI_TEST_MSG)'
	go test -v ./cli

clean:
	rm -f ./truss ./protoc-gen-truss-protocast
	go test ./cli -clean
	$(MAKE) -C transport clean

