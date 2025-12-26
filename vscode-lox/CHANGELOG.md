# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

[Unreleased]

### Changed

- Improve sorting of class property completions

### Removed

- Omit "init" from class property completions

## [3.0.0] - 2025-12-26

### Added

- Suggest method snippets inside class bodies.
- Allow renaming methods and object properties.
- Suggest keywords where applicable.
- Signature help for functions, methods, and classes.

### Fixed

- Reduce snippet indent size from 4 to 2.
- Highlight comments in the middle of expressions.
- Allow comments anywhere when formatting.
- Show docstrings in completions and hover text again.

### Changed

- Change variable snippet detail from "variable declaration" to "variable".
- Change function snippet detail from "function declaration" to "function".
- Change class snippet detail from "class declaration" to "class".
- Add placeholders to all snippets where appropriate.
- Add function name to detail of completion and outline items
- Add class name to completions of class methods and fields

### Removed

- `print` snippet.
- Function call brackets snippet.

## [2.1.0] - 2025-11-21

### Fixed

- Suggest statement keyword snippets after comments.

### Changed

- Change
  [kind](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#completionItemKind)
  of keyword snippet completion items from Keyword to Snippet.

## [2.0.1] - 2025-11-15

### Fixed

- Removed unintended items from change log.

## [2.0.0] - 2025-11-15

### Added

- Add `lox.enableExtraFeatures` setting to enable extra features that
  https://github.com/marcuscaisey/lox implements but the base Lox language does not.

### Fixed

- Allow multiline strings.
- Allow globals to be redeclared.
- Allow undefined variable access.
- Allow comments everywhere.
- Disallow referencing local in its initialiser.
- Reduce indent size from 4 spaces to 2.
- Don't report 'x can only be used inside a method definition' error inside nested functions.
- Report 'x has been used before its declaration' as warning instead of error.
- Report 'x has not been declared' as warning instead of error.

### Changed

- Disable by default extra features that https://github.com/marcuscaisey/lox implements but the base
  Lox language does not.

## [1.1.0] - 2025-09-19

### Added

- `lox.trace.server` setting to enable tracing the communication between VS Code and the Lox
  language server.

### Changed

- Report 'x has been declared but is never used' as hint instead of error.
- Stop mentioning "loxls" in extension logs where unneccessary.

## [1.0.0] - 2025-09-16

### Added

- IntelliSense - Results appear for symbols as you type.
- Code navigation - Jump to or peek at a symbol's declaration.
- Code editing - Support for formatting.
- Diagnostics - Build and lint errors shown as you type.
- Syntax highlighting.

[Unreleased]: https://github.com/marcuscaisey/lox/compare/vscode-lox/v3.0.0...HEAD
[3.0.0]: https://github.com/marcuscaisey/lox/tree/vscode-lox/v3.0.0/vscode-lox
[2.1.0]: https://github.com/marcuscaisey/lox/tree/vscode-lox/v2.1.0/vscode-lox
[2.0.1]: https://github.com/marcuscaisey/lox/tree/vscode-lox/v2.0.1/vscode-lox
[2.0.0]: https://github.com/marcuscaisey/lox/tree/vscode-lox/v2.0.0/vscode-lox
[1.1.0]: https://github.com/marcuscaisey/lox/tree/vscode-lox/v1.1.0/vscode-lox
[1.0.0]: https://github.com/marcuscaisey/lox/tree/vscode-lox/v1.0.0/vscode-lox
