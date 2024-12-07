package protocol

import "github.com/marcuscaisey/lox/lox/token"

// NewRange creates a new [Range] from the given [token.Position].
func NewRange(start, end token.Position) *Range {
	return &Range{
		Start: &Position{
			Line:      start.Line - 1,
			Character: start.ColumnUTF16(),
		},
		End: &Position{
			Line:      end.Line - 1,
			Character: end.ColumnUTF16(),
		},
	}
}
