# Lox [![CI](https://github.com/marcuscaisey/lox/actions/workflows/ci.yml/badge.svg)](https://github.com/marcuscaisey/lox/actions/workflows/ci.yml)

Lox is the dynamically typed programming language defined in the book [Crafting
Interpreters](https://craftinginterpreters.com). This repository provides an implementation of the
language and a developer tooling ecosystem as follows:

- [golox](golox): An interpreter implemented in Go.
- [tree-sitter-lox](tree-sitter-lox): A grammar for [tree-sitter](https://github.com/tree-sitter/tree-sitter).
- [loxfmt](loxfmt): An opinionated code formatter.
- [loxls](loxls): A language server.
- [loxlint](loxlint): A linter.
- [vscode-lox](vscode-lox): A VS Code extension.

Working Lox code examples can be found under [examples](examples) and
[test/testdata](test/testdata).

[spec.md](spec.md) contains a full specification of the version of the Lox language which has been
implemented.
