.PHONY: lint

lint:
	-$(MAKE) lint_golangci_lint
	-$(MAKE) lint_go_check_sumtype

lint_golangci_lint:
	golangci-lint run

lint_go_check_sumtype:
	go tool go-check-sumtype ./...
