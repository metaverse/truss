NO_COLOR=\e[0m
OK_COLOR=\e[38;5;118m
UNDER=\n________________________________________________________________________________\n
END_COLOR_LINE=$(UNDER)$(NO_COLOR)

TRANSPORT_TEST_MSG=\n$(OK_COLOR)Starting transport end to end test:$(END_COLOR_LINE)

CLI_TEST_MSG=\n$(OK_COLOR)Start server and cliclient generate, build, and run test:$(END_COLOR_LINE)


all: test

test: clean test-transport test-cli

test-transport:
	@printf '$(TRANSPORT_TEST_MSG)'
	$(MAKE) -C transport

test-cli:
	@printf '$(CLI_TEST_MSG)'
	go test -v ./cli

clean:
	go test ./cli -clean
	$(MAKE) -C transport clean

