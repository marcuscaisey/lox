//go:build tools

// This file is used to declare tool dependencies of the project. This allows us to version them in the go.mod file.
package tools

import (
	_ "github.com/rakyll/gotest"
	_ "golang.org/x/tools/cmd/stringer"
)
