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
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"strings"

	"github.com/chzyer/readline"

	"github.com/marcuscaisey/lox/golox/ast"
	"github.com/marcuscaisey/lox/golox/interpreter"
	"github.com/marcuscaisey/lox/golox/parser"
	"github.com/marcuscaisey/lox/golox/resolver"
)

var (
	cmd      = flag.String("c", "", "Program passed in as string")
	printAST = flag.Bool("p", false, "Print the AST only")

	cpuProfile = flag.String("cpuprofile", "", "Write a CPU profile to the specified file before exiting.")
	memProfile = flag.String("memprofile", "", "Write an allocation profile to the file before exiting.")
	traceFile  = flag.String("trace", "", " Write an execution trace to the specified file before exiting.")
)

// nolint:revive
func Usage() {
	fmt.Fprintf(flag.CommandLine.Output(), "Usage: golox [options] [script]\n")
	fmt.Fprintf(flag.CommandLine.Output(), "\n")
	fmt.Fprintf(flag.CommandLine.Output(), "Options:\n")
	flag.PrintDefaults()
}

func main() {
	log.SetFlags(0)

	flag.Usage = Usage
	flag.Parse()

	if *cpuProfile != "" {
		f, err := os.Create(*cpuProfile)
		if err != nil {
			log.Fatalf("failed to create CPU profile: %s", err)
		}
		defer func() {
			if err := f.Close(); err != nil {
				log.Fatalf("failed to close CPU profile: %s", err)
			}
		}()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatalf("failed to start CPU profile: %s", err)
		}
		defer pprof.StopCPUProfile()
	}
	if *memProfile != "" {
		defer func() {
			f, err := os.Create(*memProfile)
			if err != nil {
				log.Fatalf("failed to create memory profile: %s", err)
			}
			defer func() {
				if err := f.Close(); err != nil {
					log.Fatalf("failed to close memory profile: %s", err)
				}
			}()
			runtime.GC()
			if err := pprof.WriteHeapProfile(f); err != nil {
				log.Fatalf("failed to start memory profile: %s", err)
			}
		}()
	}
	if *traceFile != "" {
		f, err := os.Create(*traceFile)
		if err != nil {
			log.Fatalf("failed to create trace output file: %s", err)
		}
		defer func() {
			if err := f.Close(); err != nil {
				log.Fatalf("failed to close trace file: %s", err)
			}
		}()

		if err := trace.Start(f); err != nil {
			log.Fatalf("failed to start trace: %s", err)
		}
		defer trace.Stop()
	}

	if *cmd != "" {
		if err := run(strings.NewReader(*cmd), interpreter.New()); err != nil {
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
		flag.Usage()
		os.Exit(2)
	}
}

func run(r io.Reader, interpreter *interpreter.Interpreter) error {
	root, err := parser.Parse(r)
	if *printAST {
		ast.Print(root)
		return err
	}
	if err != nil {
		return err
	}
	localDeclDistancesByIdent, err := resolver.Resolve(root)
	if err != nil {
		return err
	}
	return interpreter.Interpret(root, localDeclDistancesByIdent)
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

	fmt.Fprintln(os.Stderr, "Welcome to Lox!")

	interpreter := interpreter.New(interpreter.REPLMode())
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
		if err := run(strings.NewReader(line), interpreter); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}

	return nil
}

func runFile(name string) error {
	f, err := os.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()
	return run(f, interpreter.New())
}
