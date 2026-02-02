// Entry point for the golox interpreter.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/chzyer/readline"

	"github.com/marcuscaisey/lox/golox/ast"
	"github.com/marcuscaisey/lox/golox/interpreter"
	"github.com/marcuscaisey/lox/golox/parser"
)

func main() {
	os.Exit(cli())
}

type usageError string

func (e usageError) Error() string {
	return fmt.Sprintf("error: %s", string(e))
}

func cli() int {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: golox [options] [<script>] [<argument>...]")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Options:")
		flag.PrintDefaults()
	}
	program := flag.String("program", "", "Program passed in as string")
	printAST := flag.Bool("ast", false, "Print the AST")
	printTokens := flag.Bool("tokens", false, "Print the lexical tokens")
	printHelp := flag.Bool("help", false, "Print this message")

	flag.Parse()

	if *printHelp {
		flag.Usage()
		return 0
	}

	if err := golox(flag.Args(), *program, *printTokens, *printAST); err != nil {
		fmt.Fprintln(os.Stderr, err)
		var usageErr usageError
		if errors.As(err, &usageErr) {
			fmt.Fprintln(os.Stderr)
			flag.Usage()
			return 2
		}
		return 1
	}

	return 0
}

func golox(args []string, program string, printAST bool, printTokens bool) error {
	if printAST && printTokens {
		return usageError("-ast and -tokens cannot be provided together")
	}

	if program != "" {
		filename := "<string>"
		argv := append([]string{filename}, args...)
		return exec(filename, strings.NewReader(program), interpreter.New(argv), printTokens, printAST)
	}

	if len(args) == 0 {
		return repl(printTokens, printAST)
	}

	filename := args[0]
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	argv := slices.Clone(args)
	argv[0] = filepath.Base(argv[0])
	return exec(filename, f, interpreter.New(argv), printTokens, printAST)
}

func exec(filename string, r io.Reader, interpreter *interpreter.Interpreter, printTokens bool, printAST bool) error {
	program, err := parser.Parse(r, filename, parser.WithPrintTokens(printTokens))
	if printTokens {
		return err
	}
	if printAST {
		ast.Print(program)
		return err
	}
	if err != nil {
		return err
	}
	return interpreter.Execute(program)
}

func repl(printTokens bool, printAST bool) error {
	cfg := &readline.Config{
		Prompt: ">>> ",
	}

	homeDir, err := os.UserHomeDir()
	if err == nil {
		cfg.HistoryFile = path.Join(homeDir, ".lox_history")
	} else {
		fmt.Fprintf(os.Stderr, "Can't get current user's home directory (%s). Command history will not be saved.\n", err)
	}

	rl, err := readline.NewEx(cfg)
	if err != nil {
		return fmt.Errorf("running Lox REPL: %s", err)
	}
	defer rl.Close()

	fmt.Fprintln(os.Stderr, "Welcome to the Lox REPL. Press Ctrl-D to exit.")

	argv := []string{"<repl>"}
	interpreter := interpreter.New(argv, interpreter.WithREPLMode(true))
	for {
		line, err := rl.Readline()
		if err != nil {
			if errors.Is(err, readline.ErrInterrupt) {
				continue
			}
			if errors.Is(err, io.EOF) {
				break
			}
			panic(fmt.Sprintf("unexpected error from readline: %s", err))
		}
		if err := exec("", strings.NewReader(line), interpreter, printTokens, printAST); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}

	return nil
}
