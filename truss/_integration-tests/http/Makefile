NO_COLOR=\e[0m
OK_COLOR=\e[38;5;118m
UNDER=\n________________________________________________________________________________\n
END_COLOR_LINE=$(UNDER)$(NO_COLOR)

TRUSS_MSG=\n$(OK_COLOR)Running Truss...$(END_COLOR_LINE)

TEST_RUNNING_MSG=\n$(OK_COLOR)Running http tests:$(END_COLOR_LINE)

TRUSS_AGAIN_MSG=\n$(OK_COLOR)Running Truss... again, to test regeneration$(END_COLOR_LINE)

all: test

test:
	@echo -e '$(TRUSS_MSG)'
	truss httptest.proto
	cp -r handlers httptest-service
	@echo -e '$(TEST_RUNNING_MSG)'
	go test
	@echo -e '$(TRUSS_AGAIN_MSG)'
	truss httptest.proto
	@echo -e '$(TEST_RUNNING_MSG)'
	go test
	$(MAKE) clean

clean:
	rm -rf httptest-service
	rm -rf third_party
