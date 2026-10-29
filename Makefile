.PHONY: lint cloc

lint:
	golangci-lint run

cloc:
	cloc --config cloc-options.txt --read-lang-def cloc-languages.txt .
