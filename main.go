package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/marcuscaisey/golox/scanner"
)

func main() {
	switch len(os.Args) {
	case 1:
		if err := runPrompt(); err != nil {
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

func runPrompt() error {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Printf(">>> ")
	for scanner.Scan() {
		if err := run(scanner.Text()); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		fmt.Printf(">>> ")
	}
	if scanner.Err() != nil {
		return fmt.Errorf("running Lox prompt: %s", scanner.Err())
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
		positions[i] = fmt.Sprintf("%d:%d", t.Line, t.Col)
	}
	for _, position := range positions {
		positionWidth = max(len(position), positionWidth)
	}
	for i := 0; i < len(tokens); i++ {
		fmt.Printf("%*s: %s [%s]\n", positionWidth, positions[i], tokens[i].Lexeme, tokens[i].Type)
	}
	if err != nil {
		return err
	}
	return nil
}
