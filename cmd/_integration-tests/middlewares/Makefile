NO_COLOR=\e[0m
OK_COLOR=\e[38;5;118m
UNDER=\n________________________________________________________________________________\n
END_COLOR_LINE=$(UNDER)$(NO_COLOR)

TRUSS_MSG=\n$(OK_COLOR)Running Truss...$(END_COLOR_LINE)

TEST_RUNNING_MSG=\n$(OK_COLOR)Running middlewares tests:$(END_COLOR_LINE)

all: test

test:
	@echo -e '$(TRUSS_MSG)'
	mkdir -p middlewarestest-service
	truss --svcout github.com/tuneinc/truss/cmd/_integration-tests/middlewares/middlewarestest-service proto/middlewares-test.proto
	cp -r handlers middlewarestest-service
	@echo -e '$(TEST_RUNNING_MSG)'
	go test -v
	$(MAKE) clean

clean:
	rm -rf middlewarestest-service
	rm -f ./proto/middlewares-test.pb.go
