// Entry point for the golox interpreter.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strings"

	"github.com/chzyer/readline"

	"github.com/marcuscaisey/lox/golox/ast"
	"github.com/marcuscaisey/lox/golox/interpreter"
	"github.com/marcuscaisey/lox/golox/parser"
)

var (
	program     = flag.String("program", "", "Program passed in as string")
	printAST    = flag.Bool("ast", false, "Print the AST")
	printTokens = flag.Bool("tokens", false, "Print the lexical tokens")
	printHelp   = flag.Bool("help", false, "Print this message")
)

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: golox [options] [script]")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Options:")
	flag.PrintDefaults()
}

func exitWithUsageErr(msg string) {
	fmt.Fprintf(os.Stderr, "error: %s\n\n", msg)
	flag.Usage()
	os.Exit(2)
}

func main() {
	log.SetFlags(0)

	flag.Usage = usage
	flag.Parse()

	if *printHelp {
		flag.Usage()
		os.Exit(0)
	}

	if *printAST && *printTokens {
		exitWithUsageErr("-ast and -tokens cannot be provided together")
	}

	if *program != "" {
		if err := run("", strings.NewReader(*program), interpreter.New()); err != nil {
			log.Fatal(err)
		}
		return
	}

	switch len(flag.Args()) {
	case 0:
		if err := runREPL(); err != nil {
			log.Fatal(err)
		}
	case 1:
		if err := runFile(flag.Arg(0)); err != nil {
			log.Fatal(err)
		}
	default:
		exitWithUsageErr("at most one path can be provided")
	}
}

func run(filename string, r io.Reader, interpreter *interpreter.Interpreter) error {
	program, err := parser.Parse(r, filename, parser.WithPrintTokens(*printTokens))
	if *printTokens {
		return err
	}
	if *printAST {
		ast.Print(program)
		return err
	}
	if err != nil {
		return err
	}
	return interpreter.Interpret(program)
}

func runREPL() error {
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

	interpreter := interpreter.New(interpreter.WithREPLMode(true))
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
		if err := run("", strings.NewReader(line), interpreter); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}

	return nil
}

func runFile(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return run(filename, f, interpreter.New())
}
