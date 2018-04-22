NO_COLOR=\e[0m
OK_COLOR=\e[38;5;118m
UNDER=\n________________________________________________________________________________\n
END_COLOR_LINE=$(UNDER)$(NO_COLOR)

TRUSS_MSG=\n$(OK_COLOR)Running Truss...$(END_COLOR_LINE)

TEST_RUNNING_MSG=\n$(OK_COLOR)Running transport tests:$(END_COLOR_LINE)

TRUSS_AGAIN_MSG=\n$(OK_COLOR)Running Truss... again, to test regeneration$(END_COLOR_LINE)

# make test match=test_to_run
match := $(match)
ifndef match
	match := .
endif


all: test

setup:
	@echo -e '$(TRUSS_MSG)'
	mkdir -p transportpermutations-service
	truss -v --svcout github.com/tuneinc/truss/cmd/_integration-tests/transport/transportpermutations-service proto/transport-test.proto
	cp -r handlers transportpermutations-service

test: setup
	@echo -e '$(TEST_RUNNING_MSG)'
	go test -run=$(match) -v
	@echo -e '$(TRUSS_AGAIN_MSG)'
	mkdir -p transportpermutations-service
	truss -v --svcout github.com/tuneinc/truss/cmd/_integration-tests/transport/transportpermutations-service proto/transport-test.proto
	@echo -e '$(TEST_RUNNING_MSG)'
	go test -run=$(match) -v
	$(MAKE) clean

bench: setup
	go test -run=XXX -bench=$(match) -benchmem
	$(MAKE) clean


# Run this when you add a new rpc to the .proto file and it will update
# handlers/handlers.go
newrpc:
	mkdir -p transportpermutations-service
	truss --svcout github.com/tuneinc/truss/cmd/_integration-tests/transport/transportpermutations-service proto/transport-test.proto
	cp -r handlers transportpermutations-service
	truss --svcout github.com/tuneinc/truss/cmd/_integration-tests/transport/transportpermutations-service proto/transport-test.proto
	cp -r transportpermutations-service/handlers/handlers.go handlers

clean:
	rm -rf transportpermutations-service
	rm -f ./proto/transport-test.pb.go
