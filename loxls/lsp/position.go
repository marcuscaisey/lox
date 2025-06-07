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

func equalPositions(x *protocol.Position, y token.Position) bool {
	yProto := newPosition(y)
	return x.Line == yProto.Line && x.Character == yProto.Character
}

// newRange creates a [*protocol.Range] from a [token.Range].
func newRange(rang token.Range) *protocol.Range {
	return newRangeSpanningRanges(rang, rang)
}

// newRangeSpanningRanges creates a [*protocol.Range] which spans the given [token.Range]s.
func newRangeSpanningRanges(start token.Range, end token.Range) *protocol.Range {
	return &protocol.Range{
		Start: newPosition(start.Start()),
		End:   newPosition(end.End()),
	}
}

// inRange reports whether a [protocol.Position] is contained within a [token.Range].
func inRange(pos *protocol.Position, rang token.Range) bool {
	return inRangePositions(pos, rang.Start(), rang.End())
}

// inRangePositions is like [inRange] but accepts a start and end position instead.
func inRangePositions(pos *protocol.Position, start token.Position, end token.Position) bool {
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
