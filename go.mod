module github.com/marcuscaisey/lox

go 1.23

require (
	github.com/chzyer/readline v1.5.1
	github.com/fatih/color v1.17.0
	github.com/mattn/go-runewidth v0.0.15
)

// Tools
require (
	github.com/BurntSushi/go-sumtype v0.0.0-20240512121737-f9f88f1fa1ac
	golang.org/x/tools v0.23.0
	gotest.tools/gotestsum v1.12.0
)

// Test-only
require (
	github.com/google/go-cmp v0.6.0
	github.com/hexops/gotextdiff v1.0.3
)

require (
	github.com/bitfield/gotestdox v0.2.2 // indirect
	github.com/dnephin/pflag v1.0.7 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	golang.org/x/mod v0.19.0 // indirect
	golang.org/x/sync v0.7.0 // indirect
	golang.org/x/sys v0.24.0 // indirect
	golang.org/x/term v0.18.0 // indirect
	golang.org/x/text v0.14.0 // indirect
)

replace (
	github.com/BurntSushi/go-sumtype => github.com/lantw44/go-sumtype v0.0.0-20230306011935-0ae65d6b318e
	github.com/fatih/color v1.17.0 => github.com/marcuscaisey/color v0.0.0-20240827185637-a96106a1b132
)
