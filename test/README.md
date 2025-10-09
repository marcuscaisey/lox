# Test Suite

golox, loxlint, and loxfmt are tested against a suite of test files defined under
[testdata](testdata).

golox and loxlint are tested by running each test file and comparing the output with the expected
output defined in the file.

loxfmt is tested by formatting each test file and asserting that the contents of the file are
unchanged.

## Test File Format

Test files are regular Lox files containing comments which describe the expectations of the test:

- `// prints: <value>` defines a string that should be printed to stdout.
- `// error: <message>` defines an error message that should be printed to stderr.
- `// hint: <message>` defines a hint message that should be printed to stderr.

All special comments can appear multiple times in a test file.

For example:

```lox
print 3 % 2; // prints: 1
print 3 % 3; // prints: 0
print 3.5 % 2; // prints: 1.5
print 3 % 1.5; // prints: 0
```

If a `// syntaxerror` comment appears at the start of a test file, it will be skipped when not
testing golox.

## Running the Tests

Run all tests:

```sh
go test ./...
```

Run the golox or loxfmt tests individually:

```sh
go test ./golox
go test ./loxlint
go test ./loxfmt
```

Run a specific test:

```sh
go test ./golox -test.run TestGolox/Number/Modulo
go test ./loxlint -test.run TestLoxlint/Number/Modulo
go test ./loxfmt -test.run TestLoxfmt/Number/Modulo
```

## Updating the Test Expectations

The expectations of each test can be updated to match the current implementation by running either
of the following commands:

```sh
go test ./golox -update
go test ./loxlint -update
go test ./loxfmt -update
```

As with running the tests, you can update the expectations of a specific test as well:

```sh
go test ./golox -test.run TestGolox/Number/Modulo -update
go test ./loxlint -test.run TestLoxlint/Number/Modulo -update
go test ./loxfmt -test.run TestLoxfmt/Number/Modulo -update
```
