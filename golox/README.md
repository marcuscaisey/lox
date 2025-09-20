# golox

golox is a Go implementation of the Lox programming language.

## Installation

```sh
go install github.com/marcuscaisey/lox/golox@latest
```

## Usage

```
Usage: golox [options] [script]

Options:
  -help
        Print this message
  -print-ast
        Print the AST only
  -program string
        Program passed in as string
```

If no script is provided, a REPL is started, otherwise the supplied script is executed.

## Examples

### Execute script

```sh
cat << EOF > test.lox
fun add(x, y) {
    return x + y;
}

print add(1, 2);
EOF

golox test.lox
```

#### Output

```
3
```

### Start REPL

```sh
golox
```

#### Output

```
Welcome to the Lox REPL. Press Ctrl-D to exit.
>>>
```

### Print AST

```sh
cat << EOF > test.lox
fun add(x, y) {
    return x + y;
}

print add(3, 4);
EOF

golox -print-ast test.lox
```

#### Output

```
(Program
  (FunDecl
    (Doc [])
    (Name add)
    (Function (Function
      (Params [
        (ParamDecl x)
        (ParamDecl y)
      ]
      (Body (Block
        (ReturnStmt (BinaryExpr
          (Left x)
          (Op +)
          (Right y))))))))
  (PrintStmt (CallExpr
    (Callee add)
    (Args [
      3
      4
    ])))
```

### Pass program as string

```sh
golox -program 'fun add(x, y) { return x + y; } print add(5, 6);'
```

#### Output

```
11
```
