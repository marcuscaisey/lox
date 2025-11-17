package lsp

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"runtime/debug"
	"time"

	"github.com/marcuscaisey/lox/golox/stubbuiltins"
	"github.com/marcuscaisey/lox/loxls/lsp/protocol"
)

type initializationOptions struct {
	ExtraFeatures *bool `json:"extraFeatures"`
}

func (i *initializationOptions) GetExtraFeatures() *bool {
	if i == nil {
		return nil
	}
	return i.ExtraFeatures
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#initialize
func (h *Handler) initialize(params *protocol.InitializeParams[*initializationOptions]) (*protocol.InitializeResult, error) {
	h.capabilities = params.GetCapabilities()
	if extraFeatures := params.GetInitializationOptions().GetExtraFeatures(); extraFeatures != nil {
		h.extraFeatures = *extraFeatures
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("writing stub builtins to cache directory: %s", err)
	}
	h.stubBuiltinsFilename = fmt.Sprintf("%s/loxls/builtins.lox", cacheDir)
	h.stubBuiltins = stubbuiltins.MustParse(h.stubBuiltinsFilename, stubbuiltins.WithExtraFeatures(h.extraFeatures))
	if err := writeStubBuiltins(h.stubBuiltinsFilename, h.stubBuiltins[0].Start().File.Contents); err != nil {
		return nil, err
	}

	version, err := buildVersionStr()
	if err != nil {
		log.Errorf("initialize: %s", err)
	}

	h.initialized = true
	return &protocol.InitializeResult{
		Capabilities: &protocol.ServerCapabilities{
			PositionEncoding: protocol.PositionEncodingKindUTF16,
			TextDocumentSync: &protocol.TextDocumentSyncOptionsOrTextDocumentSyncKind{
				Value: &protocol.TextDocumentSyncOptions{
					OpenClose: true,
					Change:    protocol.TextDocumentSyncKindIncremental,
				},
			},
			CompletionProvider: &protocol.CompletionOptions{
				TriggerCharacters: []string{"."},
			},
			HoverProvider: &protocol.BooleanOrHoverOptions{
				Value: protocol.Boolean(true),
			},
			DefinitionProvider: &protocol.BooleanOrDefinitionOptions{
				Value: protocol.Boolean(true),
			},
			ReferencesProvider: &protocol.BooleanOrReferenceOptions{
				Value: protocol.Boolean(true),
			},
			DocumentSymbolProvider: &protocol.BooleanOrDocumentSymbolOptions{
				Value: protocol.Boolean(true),
			},
			DocumentFormattingProvider: &protocol.BooleanOrDocumentFormattingOptions{
				Value: protocol.Boolean(true),
			},
			RenameProvider: &protocol.BooleanOrRenameOptions{
				Value: protocol.Boolean(true),
			},
		},
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    "loxls",
			Version: version,
		},
	}, nil
}

func writeStubBuiltins(filename string, contents []byte) error {
	if data, err := os.ReadFile(filename); err == nil {
		if bytes.Equal(data, contents) {
			return nil
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("writing stub builtins to cache directory: checking if existing stubs up to date: %s", err)
	}
	if err := os.MkdirAll(path.Dir(filename), 0755); err != nil {
		return fmt.Errorf("writing stub builtins to cache directory: %s", err)
	}
	if err := os.WriteFile(filename, contents, 0644); err != nil {
		return fmt.Errorf("writing stub builtins to cache directory: %s", err)
	}
	return nil
}

func buildVersionStr() (string, error) {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown", nil
	}
	var vcsRevision string
	var vcsTime time.Time
	for _, setting := range buildInfo.Settings {
		switch setting.Key {
		case "vcs.revision":
			vcsRevision = setting.Value
		case "vcs.time":
			var err error
			vcsTime, err = time.Parse(time.RFC3339, setting.Value)
			if err != nil {
				return "", fmt.Errorf("building version string: parsing vcs.time value from build info: %s", err)
			}
		}
	}
	if vcsRevision == "" || vcsTime.IsZero() {
		return "dev", nil
	}
	return vcsTime.Format(time.DateOnly) + "-" + vcsRevision[:8], nil
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#shutdown
func (h *Handler) shutdown() (any, error) {
	h.shuttingDown = true
	return nil, nil
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#exit
func (h *Handler) exit() error {
	code := 0
	if !h.shuttingDown {
		code = 1
	}
	os.Exit(code)
	return nil
}
