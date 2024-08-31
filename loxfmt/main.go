// Entry point for the loxfmt Lox formatter.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/marcuscaisey/lox/golox/parser"
)

var (
	write = flag.Bool("w", false, "Write result to (source) file instead of stdout")
)

// nolint:revive
func Usage() {
	fmt.Fprintf(flag.CommandLine.Output(), "Usage: loxfmt [flags] [path]\n")
	fmt.Fprintf(flag.CommandLine.Output(), "\n")
	fmt.Fprintf(flag.CommandLine.Output(), "Options:\n")
	flag.PrintDefaults()
}

func main() {
	log.SetFlags(0)

	flag.Usage = Usage
	flag.Parse()

	if len(flag.Args()) != 1 {
		flag.Usage()
		os.Exit(2)
	}

	if err := run(flag.Arg(0)); err != nil {
		log.Fatal(err)
	}
}

func run(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	program, err := parser.Parse(bytes.NewReader(data))
	if err != nil {
		return err
	}

	formatted := format(program)
	if *write {
		if err := os.WriteFile(path, []byte(formatted), 0644); err != nil {
			return err
		}
	} else {
		fmt.Print(formatted)
	}

	return nil
}
