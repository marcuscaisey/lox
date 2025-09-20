# Golox

Golox is a Go implementation of the Lox programming language.

## Installation

```sh
go install github.com/marcuscaisey/lox/golox@latest
```

## Usage

```
Usage: golox [options] [script]

Options:
  -help
        Print this message
  -print-ast
        Print the AST only
  -program string
        Program passed in as string
```

If no script is provided, a REPL is started, otherwise the supplied script is executed.
