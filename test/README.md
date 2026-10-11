# Test Suite

clox, golox, loxfmt, and loxlint are tested against a suite of test files defined under
[testdata](testdata).

clox, golox, and loxlint are tested by running each test file and comparing the output with the
expected output defined in the file.

loxfmt is tested by formatting each test file and asserting that the contents of the file are
unchanged.

## Test File Format

Test files are regular Lox files containing comments which describe the expectations of the test:

- `// prints: <value>` defines a string that should be printed to stdout by clox and golox.
- `// error: <message>` defines an error message that should be printed to stderr by clox and golox.
- `// lint error: <message>` defines an error message that should be printed to stderr by loxlint.
- `// lint warning: <message>` defines a warning message that should be printed to stderr by loxlint.
- `// lint hint: <message>` defines a hint message that should be printed to stderr by loxlint.

All special comments can appear multiple times in a test file.

For example:

```lox
print 3 % 2; // prints: 1
print 3 % 3; // prints: 0
print 3.5 % 2; // prints: 1.5
print 3 % 1.5; // prints: 0
```

If a `// syntaxerror` comment appears at the start of a test file, it will be skipped when not
testing clox or golox.

## Running the Tests

Run all tests:

```sh
go test ./...
```

Run the clox, golox, loxfmt, or loxlint tests individually:

```sh
go test ./clox/test
go test ./golox
go test ./loxfmt
go test ./loxlint
```

Run a specific test:

```sh
go test ./clox/test -test.run TestClox/Number/Modulo
go test ./golox -test.run TestGolox/Number/Modulo
go test ./loxfmt -test.run TestLoxfmt/Number/Modulo
go test ./loxlint -test.run TestLoxlint/Number/Modulo
```

## Updating the Test Expectations

The expectations of each test can be updated to match the current implementation by running either
of the following commands:

```sh
go test ./clox/test -update
go test ./golox -update
go test ./loxfmt -update
go test ./loxlint -update
```

As with running the tests, you can update the expectations of a specific test as well:

```sh
go test ./clox/test -test.run TestClox/Number/Modulo -update
go test ./golox -test.run TestGolox/Number/Modulo -update
go test ./loxfmt -test.run TestLoxfmt/Number/Modulo -update
go test ./loxlint -test.run TestLoxlint/Number/Modulo -update
```
