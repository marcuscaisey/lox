.PHONY: lint

lint:
	-$(MAKE) lint_golangci_lint
	-$(MAKE) lint_go_sumtype

lint_golangci_lint:
	golangci-lint run

lint_go_sumtype:
	go tool go-sumtype $$(go list ./...)
