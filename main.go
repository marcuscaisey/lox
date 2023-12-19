package main

import (
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/chzyer/readline"
	"github.com/marcuscaisey/golox/scanner"
)

func main() {
	switch len(os.Args) {
	case 1:
		if err := runREPL(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(65)
		}
	case 2:
		if err := runFile(os.Args[1]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(65)
		}
	default:
		fmt.Fprintln(os.Stderr, "Usage: golox [script]")
		os.Exit(64)
	}
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
			break // err is io.EOF
		}
		if err := run(line); err != nil {
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
	return run(string(srcBytes))
}

func run(src string) error {
	s := scanner.New(src)
	tokens, err := s.Scan()
	positions := make([]string, len(tokens))
	positionWidth := 0
	for i, t := range tokens {
		positions[i] = fmt.Sprintf("%d:%d", t.Line, t.Byte)
		positionWidth = max(len(positions[i]), positionWidth)
	}
	for i := 0; i < len(tokens); i++ {
		fmt.Printf("%*s: %s [%s]\n", positionWidth, positions[i], tokens[i].Lexeme, tokens[i].Type)
	}
	if err != nil {
		return err
	}
	return nil
}
