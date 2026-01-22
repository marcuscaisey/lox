package lsp

// This file contains handlers for the methods described under
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_synchronization.

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/marcuscaisey/lox/golox/analyse"
	"github.com/marcuscaisey/lox/golox/ast"
	"github.com/marcuscaisey/lox/golox/loxerr"
	"github.com/marcuscaisey/lox/golox/parser"
	"github.com/marcuscaisey/lox/loxls/jsonrpc"
	"github.com/marcuscaisey/lox/loxls/lsp/protocol"
)

type document struct {
	// Client provided
	URI     string
	Version int
	Text    string

	// Server generated
	Filename       string
	Program        *ast.Program
	HasParseErrors bool
	IdentBindings  map[*ast.Ident][]ast.Binding
	Completor      *completor
}

// document returns the document with the given URI, or an error if it doesn't exist.
func (h *Handler) document(uri string) (*document, error) {
	doc, ok := h.docs[uri]
	if !ok {
		return nil, jsonrpc.NewError(jsonrpc.InvalidParams, "Document not found", map[string]any{"uri": uri})
	}
	return doc, nil
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_didOpen
func (h *Handler) textDocumentDidOpen(params *protocol.DidOpenTextDocumentParams) error {
	if err := h.updateDoc(params.TextDocument.Uri, params.TextDocument.Version, string(params.TextDocument.Text)); err != nil {
		return fmt.Errorf("textDocument/didOpen: %s", err)
	}
	return nil
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_didChange
func (h *Handler) textDocumentDidChange(params *protocol.DidChangeTextDocumentParams) error {
	doc, err := h.document(params.TextDocument.Uri)
	if err != nil {
		return err
	}
	src := doc.Text
	for _, change := range params.ContentChanges {
		switch change := change.Value.(type) {
		case *protocol.IncrementalTextDocumentContentChangeEvent:
			src, err = applyIncrementalTextChange(src, change)
			if err != nil {
				return fmt.Errorf("textDocument/didChange: %s", err)
			}
		case *protocol.FullTextDocumentContentChangeEvent:
			src = change.Text
		}
	}
	if err := h.updateDoc(params.TextDocument.Uri, params.TextDocument.Version, src); err != nil {
		return fmt.Errorf("textDocument/didChange: %s", err)
	}
	return nil
}

func applyIncrementalTextChange(text string, change *protocol.IncrementalTextDocumentContentChangeEvent) (string, error) {
	utf16IdxToByteIdx := func(text string, utf16Idx int) (int, bool) {
		if utf16Idx == 0 {
			return 0, true
		}
		if i := strings.IndexRune(text, '\n'); i != -1 {
			text = text[:i+1]
		}
		curUTF16Idx := 0
		for i, r := range text {
			curUTF16Idx += utf16.RuneLen(r)
			if curUTF16Idx == utf16Idx {
				return i + utf8.RuneLen(r), true
			}
			if curUTF16Idx > utf16Idx {
				return 0, false
			}
		}
		return 0, false
	}

	low := -1
	high := -1
	if change.Range.Start.Line == 0 && change.Range.Start.Character == 0 {
		low = 0
	}
	if change.Range.Start.Line == 0 && change.Range.Start.Character == 0 {
		high = 0
	}

	line := 0
	for i, r := range text {
		if r == '\n' || i == 0 {
			if r == '\n' {
				line++
				i++
			}
			if line == change.Range.Start.Line && low < 0 {
				byteIdx, ok := utf16IdxToByteIdx(text[i:], change.Range.Start.Character)
				if !ok {
					return "", fmt.Errorf("applying incremental text change: range start character %d on line %d not found", change.Range.Start.Character, line)
				}
				low = i + byteIdx
			}
			if line == change.Range.End.Line {
				byteIdx, ok := utf16IdxToByteIdx(text[i:], change.Range.End.Character)
				if !ok {
					return "", fmt.Errorf("applying incremental text change: range end character %d on line %d not found", change.Range.Start.Character, line)
				}
				high = i + byteIdx
				break
			}
		}
	}

	if low == -1 {
		return "", fmt.Errorf("applying incremental text change: range start line %d not found", change.Range.Start.Line)
	}
	if high == -1 {
		return "", fmt.Errorf("applying incremental text change: range end line %d not found", change.Range.End.Line)
	}

	return text[:low] + change.Text + text[high:], nil
}

func (h *Handler) updateDoc(uri string, version int, src string) error {
	filename, err := uriToFilename(uri)
	if err != nil {
		return fmt.Errorf("updating document: %w", err)
	}
	program, err := parser.Parse(strings.NewReader(string(src)), filename, parser.WithComments(true), parser.WithExtraFeatures(h.extraFeatures))
	var parseLoxErrs loxerr.Errors
	if err != nil && !errors.As(err, &parseLoxErrs) {
		return fmt.Errorf("updating document: %w", err)
	}

	var builtins []ast.Decl
	if filename != h.stubBuiltinsFilename {
		builtins = h.stubBuiltins
	}
	identBindings, resolveErr := analyse.ResolveIdents(program, builtins, analyse.WithExtraFeatures(h.extraFeatures))

	h.docs[uri] = &document{
		URI:            uri,
		Version:        version,
		Text:           src,
		Filename:       filename,
		Program:        program,
		HasParseErrors: len(parseLoxErrs) > 0,
		IdentBindings:  identBindings,
		Completor:      newCompletor(program, identBindings, h.stubBuiltins),
	}

	semanticsErr := analyse.CheckSemantics(program, analyse.WithExtraFeatures(h.extraFeatures))
	var resolveLoxErrs, semanticsLoxErrs loxerr.Errors
	errors.As(resolveErr, &resolveLoxErrs)
	errors.As(semanticsErr, &semanticsLoxErrs)
	loxErrs := slices.Concat(parseLoxErrs, resolveLoxErrs, semanticsLoxErrs)
	loxErrs.Sort()

	var diagnostics []*protocol.Diagnostic
	if filename != h.stubBuiltinsFilename {
		diagnostics = make([]*protocol.Diagnostic, len(loxErrs))
		for i, e := range loxErrs {
			var severity protocol.DiagnosticSeverity
			var tags []protocol.DiagnosticTag
			switch e.Type {
			case loxerr.Fatal:
				severity = protocol.DiagnosticSeverityError
			case loxerr.Warning:
				severity = protocol.DiagnosticSeverityWarning
			case loxerr.Hint:
				severity = protocol.DiagnosticSeverityHint
				if strings.HasSuffix(e.Msg, "has been declared but is never used") {
					tags = append(tags, protocol.DiagnosticTagUnnecessary)
				}
			}
			diagnostics[i] = &protocol.Diagnostic{Range: newRange(e), Severity: severity, Source: "loxls", Message: e.Msg, Tags: tags}
		}
	} else {
		diagnostics = []*protocol.Diagnostic{}
	}

	return h.client.TextDocumentPublishDiagnostics(&protocol.PublishDiagnosticsParams{
		Uri:         uri,
		Version:     protocol.NewOptional(version),
		Diagnostics: diagnostics,
	})
}

func uriToFilename(uri string) (string, error) {
	if !strings.HasPrefix(uri, "file://") {
		return "", fmt.Errorf("invalid URI %q: must start with file://", uri)
	}
	return strings.TrimPrefix(uri, "file://"), nil
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_didClose
func (h *Handler) textDocumentDidClose(params *protocol.DidCloseTextDocumentParams) error {
	doc, err := h.document(params.TextDocument.Uri)
	if err != nil {
		return err
	}
	delete(h.docs, doc.URI)
	return nil
}
