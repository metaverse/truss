NO_COLOR=\e[0m
OK_COLOR=\e[38;5;118m
UNDER=\n________________________________________________________________________________\n
END_COLOR_LINE=$(UNDER)$(NO_COLOR)

HTTPTEST_MSG=\n$(OK_COLOR)Starting http end to end test:$(END_COLOR_LINE)

CLITEST_MSG=\n$(OK_COLOR)Start server and cliclient generate, build, and run test:$(END_COLOR_LINE)

all: test

test: clean httptest clitest

httptest:
	@echo -e '$(HTTPTEST_MSG)'
	$(MAKE) -C http

clitest:
	@echo -e '$(CLITEST_MSG)'
	go test -v ./cli

clean:
	go test ./cli -clean
