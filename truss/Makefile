default:
	$(MAKE) -C ..

test-integration: testclean
	$(MAKE) -C _integration-tests

# Removes generated code from tests
testclean:
	$(MAKE) -C _integration-tests clean
