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

// posInRange reports whether a [protocol.Position] is contained within a [token.Range].
func posInRange(pos *protocol.Position, rang token.Range) bool {
	start := rang.Start()
	end := rang.End()
	line := pos.Line + 1
	col := pos.Character
	if start.Line == end.Line {
		return line == start.Line && col >= start.ColumnUTF16() && col < end.ColumnUTF16()
	} else if line == start.Line {
		return col >= start.ColumnUTF16()
	} else if line == end.Line {
		return col < end.ColumnUTF16()
	} else {
		return line > start.Line && line < end.Line
	}
}
