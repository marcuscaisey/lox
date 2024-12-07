# loxls

loxls is a language server for the Lox programming language which implements the language server
protocol (LSP) as defined at
https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/.

## Installation

```sh
go install github.com/marcuscaisey/lox/loxls@latest
```

## Usage

```
Usage: loxls
```
## Implemented Features

### Language Features
* [textDocument/documentSymbol](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_documentSymbol)
* [textDocument/publishDiagnostics](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_publishDiagnostics)
* [textDocument/formatting](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_formatting)

### Window Features
* [window/showMessage](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#window_showMessage)
