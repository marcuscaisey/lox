# Examples

This directory contains some example Lox programs which can be executed by `golox`. See
[golox](../golox) for how to install `golox`.

## Game Of Life

### Usage

```
Usage: game_of_life.lox [options]

Simulates Conway’s Game of Life in the terminal.

Options:
  --lines    Number of terminal lines used for the display area (default 40)
  --columns  Number of terminal columns used for the display area (default 100)
  --speed    Simulation speed in generations per second (default 30)
  --pattern  Initial pattern to seed the system with. Must be one of gun,
             spaceship, glider, blinker, toad, or beacon. (default gun)
  --help     Print this message
```

### Example

```sh
golox examples/game_of_life.lox --lines $LINES --columns $COLUMNS
```

```
Speed: 30 generations/s, Generations: 60. Press Ctrl-C to exit.
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                                                                                  │
│                                                                                                  │
│                                                                                                  │
│                                                                                                  │
│                                                                                                  │
│                                                                                                  │
│                                                                                                  │
│                                                                                                  │
│                                                                                                  │
│                                                                                                  │
│                                                                                                  │
│                                                ██                                                │
│                                            ██  ██                                                │
│                        ████            ████                        ████                          │
│                      ██      ██        ████                        ████                          │
│████                ██          ██      ████                                                      │
│████                ██      ██  ████        ██  ██                                                │
│                    ██          ██              ██                                                │
│                      ██      ██                                                                  │
│                        ████                                                                      │
│                                              ██                                                  │
│                                                ████                                              │
│                                              ████                                                │
│                                                                                                  │
│                                                                                                  │
│                                                                                                  │
│                                                                                                  │
│                                                                                                  │
│                                                            ██  ██                                │
│                                                              ████                                │
│                                                              ██                                  │
│                                                                                                  │
│                                                                                                  │
│                                                                                                  │
│                                                                                                  │
└──────────────────────────────────────────────────────────────────────────────────────────────────┘
```

## Fibonacci

### Usage

```
Usage: fibonacci.lox <n>

Prints the nth fibonacci number.
```

### Example

```sh
golox examples/fibonacci.lox 30
```

```
832040
```

## Primes Less Than

### Usage

```
Usage: primes_less_than.lox <n>

Generates and prints the prime numbers less than n using the sieve of Eratosthenes.
```

### Example

```sh
golox examples/primes_less_than.lox 20
```

```
2
3
5
7
11
13
17
19
```
