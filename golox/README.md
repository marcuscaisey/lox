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
  -c string
        Program passed in as string
  -p    Print the AST only
```

If no script is provided, a REPL is started, otherwise the supplied script is executed.
