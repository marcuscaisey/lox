package lsp

import (
	"github.com/marcuscaisey/lox/lox/token"
	"github.com/marcuscaisey/lox/loxls/lsp/protocol"
)

// newRange creates a [protocol.Range] from a pair of [token.Position].
func newRange(start, end token.Position) *protocol.Range {
	return &protocol.Range{
		Start: &protocol.Position{
			Line:      start.Line - 1,
			Character: start.ColumnUTF16(),
		},
		End: &protocol.Position{
			Line:      end.Line - 1,
			Character: end.ColumnUTF16(),
		},
	}
}
