package lsp

import (
	"unicode/utf16"

	"github.com/marcuscaisey/lox/golox/ast"
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
	return &protocol.Range{
		Start: newPosition(rang.Start()),
		End:   newPosition(rang.End()),
	}
}

// inRange reports whether a [protocol.Position] is contained within a [token.Range].
func inRange(pos *protocol.Position, rang token.Range) bool {
	return inRangePositions(pos, rang.Start(), rang.End())
}

// inRangeOrFollows reports whether a [protocol.Position] is at the end of or contained with a [token.Range].
func inRangeOrFollows(pos *protocol.Position, rang token.Range) bool {
	end := newPosition(rang.End())
	return (pos.Line == end.Line && pos.Character == end.Character) || inRange(pos, rang)
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

// outermostNodeAt returns the outermost node of a [*ast.Program] which has type T and contains a [*protocol.Position].
func outermostNodeAt[T ast.Node](program *ast.Program, pos *protocol.Position) (T, bool) {
	return ast.Find(program, func(node T) bool {
		return inRange(pos, node)
	})
}

// outermostNodeAtOrBefore returns the outermost node of a [*ast.Program] which has type T and contains or precedes a
// [*protocol.Position].
func outermostNodeAtOrBefore[T ast.Node](node ast.Node, pos *protocol.Position) (T, bool) {
	return ast.Find(node, func(node T) bool {
		return inRangeOrFollows(pos, node)
	})
}

// innermostNodeAt returns the innermost node of a [*ast.Program] which has type T and contains a [*protocol.Position].
func innermostNodeAt[T ast.Node](node ast.Node, pos *protocol.Position) (T, bool) {
	return ast.FindLast(node, func(node T) bool {
		return inRange(pos, node)
	})
}

func columnUTF16(p token.Position) int {
	line := p.File.Line(p.Line)
	return len(utf16.Encode([]rune(string(line[:p.Column]))))
}
