module github.com/marcuscaisey/lox

go 1.24

require (
	github.com/chzyer/readline v1.5.1
	github.com/mattn/go-runewidth v0.0.15
	golang.org/x/term v0.29.0
)

// Test-only
require github.com/hexops/gotextdiff v1.0.3

require (
	github.com/marcuscaisey/go-sumtype v0.0.0-20241208122212-4c96c503b8ce // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	golang.org/x/mod v0.24.0 // indirect
	golang.org/x/sync v0.12.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/tools v0.31.0 // indirect
)

tool (
	github.com/marcuscaisey/go-sumtype
	golang.org/x/tools/cmd/stringer
)
