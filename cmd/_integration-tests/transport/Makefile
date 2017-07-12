NO_COLOR=\e[0m
OK_COLOR=\e[38;5;118m
UNDER=\n________________________________________________________________________________\n
END_COLOR_LINE=$(UNDER)$(NO_COLOR)

TRUSS_MSG=\n$(OK_COLOR)Running Truss...$(END_COLOR_LINE)

TEST_RUNNING_MSG=\n$(OK_COLOR)Running transport tests:$(END_COLOR_LINE)

TRUSS_AGAIN_MSG=\n$(OK_COLOR)Running Truss... again, to test regeneration$(END_COLOR_LINE)

all: test

setup:
	@echo -e '$(TRUSS_MSG)'
	truss transport-test.proto
	cp -r handlers transportpermutations-service

test: setup
	@echo -e '$(TEST_RUNNING_MSG)'
	go test -v
	@echo -e '$(TRUSS_AGAIN_MSG)'
	truss transport-test.proto
	@echo -e '$(TEST_RUNNING_MSG)'
	go test -v
	$(MAKE) clean

bench: setup
	go test -run=XXX -bench=.
	$(MAKE) clean


# Run this when you add a new rpc to the .proto file and it will update
# handlers/handlers.go
newrpc:
	truss transport-test.proto
	cp -r handlers transportpermutations-service
	truss transport-test.proto
	cp -r transportpermutations-service/handlers/handlers.go handlers

clean:
	rm -rf transportpermutations-service
