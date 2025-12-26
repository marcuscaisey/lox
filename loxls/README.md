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
  "extraFeatures": true,
}
```

## Features

### [textDocument/definition](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_definition)

![textDocument/definition demo](demos/text-document-definition.gif)

### [textDocument/references](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_references)

![textDocument/references demo](demos/text-document-references.gif)

### [textDocument/hover](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_hover)

![textDocument/hover demo](demos/text-document-hover.gif)

### [textDocument/documentSymbol](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_documentSymbol)

![textDocument/documentSymbol demo](demos/text-document-document-symbol.gif)

### [textDocument/completion](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_completion)

![textDocument/completion demo](demos/text-document-completion.gif)

### [textDocument/publishDiagnostics](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_publishDiagnostics)

![textDocument/publishDiagnostics demo](demos/text-document-publish-diagnostics.gif)

### [textDocument/signatureHelp](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_signatureHelp)

![textDocument/signatureHelp demo](demos/text-document-signature-help.gif)

### [textDocument/formatting](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_formatting)

![textDocument/formatting demo](demos/text-document-formatting.gif)

### [textDocument/rename](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_rename)

![textDocument/rename demo](demos/text-document-rename.gif)
