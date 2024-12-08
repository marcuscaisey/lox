# Test Suite

golox and loxfmt are tested against a suite of test files defined under [testdata](testdata). golox
is tested by running each test file and comparing the output with the expected output defined in the
file. loxfmt is tested by formatting each test file and asserting that the contents of the file are
unchanged.

## Test File Format

Test files are regular Lox files containing comments which describe the expectations of the test:

- `// prints: <value>` defines a string that should be printed to stdout.
- `// error: <message>` defines an error message that should be printed to stderr.

Both special comments can appear multiple times in a test file.

For example:

```lox
print 3 % 2; // prints: 1
print 3 % 3; // prints: 0
print 3.5 % 2; // prints: 1.5
print 3 % 1.5; // prints: 0
```

If a `// noformat` comment appears at the start of a test file, the file will be not be formatted.
This is useful for files which contain syntax errors and can't be parsed.

## Running the Tests

Run all tests:

```sh
make test
```

Run the golox or loxfmt tests individually:

```sh
make test_golox
make test_loxfmt
```

Run a specific test:

```sh
make test_golox RUN=TestInterpreter/Number/Modulo
make test_loxfmt RUN=TestFormatter/Number/Modulo
```

## Updating the Test Expectations

The expectations of each test can be updated to match the current implementation by running either
of the following commands:

```sh
make update_golox_tests
make update_loxfmt_tests
```

As with running the tests, you can update the expectations of a specific test as well:

```sh
make update_golox_tests RUN=TestInterpreter/Number/Modulo
make update_loxfmt_tests RUN=TestFormatter/Number/Modulo
```
