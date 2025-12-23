# loxls

loxls is a language server for the Lox programming language which implements the language server
protocol (LSP) as defined at
https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/.

## Installation

```sh
go install github.com/marcuscaisey/lox/loxls@latest
```

## Settings

loxls can be configured via the `initializationOptions` of the `initialize` request. The default
settings are shown below.

```jsonc
{
  // Enable the language server to understand the extra features that
  // https://github.com/marcuscaisey/lox implements but the base Lox language does not.
  "extraFeatures": true
}
```

## Implemented Features

### Language Features
* [textDocument/definition](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_definition)
* [textDocument/references](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_references)
* [textDocument/hover](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_hover)
* [textDocument/documentSymbol](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_documentSymbol)
* [textDocument/completion](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_completion)
* [textDocument/publishDiagnostics](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_publishDiagnostics)
* [textDocument/signatureHelp](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_signatureHelp)
* [textDocument/formatting](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_formatting)
* [textDocument/rename](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_rename)

### Window Features
* [window/showMessage](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#window_showMessage)
