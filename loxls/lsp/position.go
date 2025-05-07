package lsp

import (
	"unicode/utf16"

	"github.com/marcuscaisey/lox/golox/token"
	"github.com/marcuscaisey/lox/loxls/lsp/protocol"
)

func newPosition(p token.Position) *protocol.Position {
	return &protocol.Position{
		Line:      p.Line - 1,
		Character: columnUTF16(p),
	}
}

func newRange(rang token.Range) *protocol.Range {
	start := rang.Start()
	end := rang.End()
	return &protocol.Range{
		Start: &protocol.Position{
			Line:      start.Line - 1,
			Character: columnUTF16(start),
		},
		End: &protocol.Position{
			Line:      end.Line - 1,
			Character: columnUTF16(end),
		},
	}
}

// inRange reports whether a [protocol.Position] is contained within a [token.Range].
func inRange(pos *protocol.Position, rang token.Range) bool {
	start := rang.Start()
	end := rang.End()
	line := pos.Line + 1
	col := pos.Character
	if start.Line == end.Line {
		return line == start.Line && col >= columnUTF16(start) && col < columnUTF16(end)
	} else if line == start.Line {
		return col >= columnUTF16(start)
	} else if line == end.Line {
		return col < columnUTF16(end)
	} else {
		return line > start.Line && line < end.Line
	}
}

func columnUTF16(p token.Position) int {
	line := p.File.Line(p.Line)
	return len(utf16.Encode([]rune(string(line[:p.Column]))))
}
