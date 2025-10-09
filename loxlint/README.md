# loxlint

loxlint is a linter for the Lox programming language.

## Installation

```sh
go install github.com/marcuscaisey/lox/loxlint@latest
```

## Usage

```
Usage: loxlint [flags] [path]

If no path is provided, the file is read from stdin.

Options:
  -help
        Print this message
```

## Examples

### Lint stdin

```sh
cat << EOF | loxlint
fun add(x, y, z) {
  return x + y;
}

print add(1, 2);
EOF
```

#### Output

```
1:15: hint: z has been declared but is never used
fun add(x, y, z) {
              ~
```

### Lint file

```sh
cat << EOF > test.lox
fun add(x, y, z) {
  return x + y;
}

print add(3, 4);
EOF
loxlint test.lox
```

#### Output

```
1:15: hint: z has been declared but is never used
fun add(x, y, z) {
              ~
```
