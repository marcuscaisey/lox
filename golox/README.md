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
  -cpuprofile string
        Write a CPU profile to the specified file before exiting.
  -memprofile string
        Write an allocation profile to the file before exiting.
  -p    Print the AST only
  -trace string
         Write an execution trace to the specified file before exiting.
```

If no script is provided, a REPL is started, otherwise the supplied script is executed.
