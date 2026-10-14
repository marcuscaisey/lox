package main_test

import (
	"flag"
	"testing"

	"github.com/marcuscaisey/lox/test/loxtest"
)

var (
	interpreter = flag.String("interpreter", "", "Interpreter to run tests against instead of golox")
)

func TestGolox(t *testing.T) {
	goloxPath := *interpreter
	if goloxPath == "" {
		goloxPath = loxtest.MustBuildBinary(t, "golox")
	}
	runner := loxtest.NewInterpreterRunner(t, goloxPath)
	loxtest.Run(t, runner, loxtest.WithSkipSyntaxErrors(false))
}
