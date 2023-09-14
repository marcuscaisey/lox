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
		return fmt.Errorf("running lox prompt: %s", scanner.Err())
	}
	return nil
}

func runFile(name string) error {
	srcBytes, err := os.ReadFile(name)
	if err != nil {
		return fmt.Errorf("running lox file: %s", err)
	}
	if err := run(string(srcBytes)); err != nil {
		return fmt.Errorf("running lox file: %s", err)
	}
	return nil
}

func run(src string) error {
	scanner := scanner.New(src)
	tokens := scanner.ScanTokens()
	for _, token := range tokens {
		fmt.Println(token)
	}
	return nil
}
