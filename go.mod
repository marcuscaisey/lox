module github.com/marcuscaisey/lox

go 1.24

require (
	github.com/chzyer/readline v1.5.1
	github.com/mattn/go-runewidth v0.0.15
	golang.org/x/term v0.29.0
)

// Test-only
require (
	github.com/google/go-cmp v0.6.0
	github.com/hexops/gotextdiff v1.0.3
)

require (
	github.com/bitfield/gotestdox v0.2.2 // indirect
	github.com/dnephin/pflag v1.0.7 // indirect
	github.com/fatih/color v1.17.0 // indirect
	github.com/fsnotify/fsnotify v1.8.0 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/marcuscaisey/go-sumtype v0.0.0-20241208122212-4c96c503b8ce // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	golang.org/x/mod v0.24.0 // indirect
	golang.org/x/sync v0.12.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/text v0.17.0 // indirect
	golang.org/x/tools v0.31.0 // indirect
	gotest.tools/gotestsum v1.12.1 // indirect
)

tool (
	github.com/marcuscaisey/go-sumtype
	golang.org/x/tools/cmd/stringer
	gotest.tools/gotestsum
)
