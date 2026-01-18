# Lox [![CI](https://github.com/marcuscaisey/lox/actions/workflows/ci.yml/badge.svg)](https://github.com/marcuscaisey/lox/actions/workflows/ci.yml)

Lox is the dynamically typed programming language defined in the book [Crafting
Interpreters](https://craftinginterpreters.com). This repository contains:

- An interpreter implemented in Go: [golox](golox)
- A grammar for [tree-sitter](https://github.com/tree-sitter/tree-sitter):
  [tree-sitter-lox](tree-sitter-lox)
- A formatter: [loxfmt](loxfmt)
- A language server: [loxls](loxls)
- A linter: [loxlint](loxlint)
- A VS Code extension: [vscode-lox](vscode-lox)

Working Lox code examples can be found under [examples](examples) and
[test/testdata](test/testdata).

[spec.md](spec.md) contains a full specification of the version of the Lox language which has been
implemented.
