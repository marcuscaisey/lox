.PHONY: test test_golox test_loxfmt update_golox_tests update_loxfmt_tests

test: test_golox test_loxfmt

test_golox:
	$(MAKE) -C golox test

test_loxfmt:
	$(MAKE) -C loxfmt test

update_golox_tests:
	$(MAKE) -C golox update_tests

update_loxfmt_tests:
	$(MAKE) -C loxfmt update_tests
