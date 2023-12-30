// Entry point for the golox interpreter.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/chzyer/readline"

	"github.com/marcuscaisey/golox/ast"
	"github.com/marcuscaisey/golox/lexer"
	"github.com/marcuscaisey/golox/parser"
)

var (
	cmd      = flag.String("c", "", "Program passed in as string")
	printTok = flag.Bool("t", false, "Print the tokens only")
)

//nolint:revive
func Usage() {
	fmt.Fprintf(os.Stderr, "Usage: golox [options] [script]\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Options:\n")
	flag.PrintDefaults()
}

func main() {
	flag.Usage = Usage
	flag.Parse()

	if *cmd != "" {
		if err := runSrc(*cmd); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	switch len(flag.Args()) {
	case 0:
		if err := runREPL(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case 1:
		if err := runFile(os.Args[1]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	default:
		flag.Usage()
		os.Exit(2)
	}
}

func runSrc(src string) error {
	l := lexer.New(src)
	tokens, err := l.Lex()
	if err != nil {
		return err
	}
	if *printTok {
		for _, t := range tokens {
			fmt.Printf("%-10s %s\n", t.Type, t.Literal)
		}
		return nil
	}
	p := parser.New(tokens)
	root, err := p.Parse()
	if err != nil {
		return err
	}
	ast.Print(root)
	return nil
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
		if err := runSrc(line); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}

	return nil
}

func runFile(name string) error {
	srcBytes, err := os.ReadFile(name)
	if err != nil {
		return fmt.Errorf("running Lox file: %s", err)
	}
	return runSrc(string(srcBytes))
}
