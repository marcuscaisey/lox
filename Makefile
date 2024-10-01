.PHONY: test test_golox

test: test_golox test_loxfmt

test_golox:
	$(MAKE) -C golox test

test_loxfmt:
	$(MAKE) -C loxfmt test
