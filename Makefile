.PHONY: test test_golox test_loxfmt update_golox_tests update_loxfmt_tests lint lint_golangci_lint lint_go_sumtype

test:
	-$(MAKE) test_golox
	-$(MAKE) test_loxfmt

test_golox:
	$(MAKE) -C golox test

test_loxfmt:
	$(MAKE) -C loxfmt test

update_golox_tests:
	$(MAKE) -C golox update_tests

update_loxfmt_tests:
	$(MAKE) -C loxfmt update_tests

lint:
	-$(MAKE) lint_golangci_lint
	-$(MAKE) lint_go_sumtype

lint_golangci_lint:
	golangci-lint run

lint_go_sumtype:
	go run github.com/marcuscaisey/go-sumtype $$(go list ./...)
