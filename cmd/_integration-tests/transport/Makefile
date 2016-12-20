NO_COLOR=\e[0m
OK_COLOR=\e[38;5;118m
UNDER=\n________________________________________________________________________________\n
END_COLOR_LINE=$(UNDER)$(NO_COLOR)

TRUSS_MSG=\n$(OK_COLOR)Running Truss...$(END_COLOR_LINE)

TEST_RUNNING_MSG=\n$(OK_COLOR)Running transport tests:$(END_COLOR_LINE)

TRUSS_AGAIN_MSG=\n$(OK_COLOR)Running Truss... again, to test regeneration$(END_COLOR_LINE)

all: test

test:
	@echo -e '$(TRUSS_MSG)'
	truss transport-test.proto
	cp -r handlers transportpermutations-service
	@echo -e '$(TEST_RUNNING_MSG)'
	go test -v
	@echo -e '$(TRUSS_AGAIN_MSG)'
	truss transport-test.proto
	@echo -e '$(TEST_RUNNING_MSG)'
	go test -v
	$(MAKE) clean

clean:
	rm -rf transportpermutations-service
