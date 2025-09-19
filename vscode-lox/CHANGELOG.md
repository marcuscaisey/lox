# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- `lox.trace.server` setting to enable tracing the communication between VS Code and the Lox
  language server.

### Changes

- Report unused declarations as hints instead of errors.
- Stop mentioning "loxls" in extension logs where unneccessary.

## [1.0.0] - 2025-09-16

### Added

- IntelliSense - Results appear for symbols as you type.
- Code navigation - Jump to or peek at a symbol's declaration.
- Code editing - Support for formatting.
- Diagnostics - Build and lint errors shown as you type.
- Syntax highlighting.

[Unreleased]: https://github.com/marcuscaisey/lox/compare/vscode-lox/v1.0.0...HEAD
[1.0.0]: https://github.com/marcuscaisey/lox/tree/vscode-lox/v1.0.0/vscode-lox
