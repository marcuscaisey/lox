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
  -ast
        Print the AST
  -help
        Print this message
  -program string
        Program passed in as string
  -tokens
        Print the lexical tokens
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

golox -ast test.lox
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

### Print Tokens

```sh
cat << EOF > test.lox
fun add(x, y) {
    return x + y;
}

print add(3, 4);
EOF

golox -tokens test.lox
```

#### Output

```
test.lox:1:1: fun [Fun]
test.lox:1:5: add [Ident]
test.lox:1:8: ( [LeftParen]
test.lox:1:9: x [Ident]
test.lox:1:10: , [Comma]
test.lox:1:12: y [Ident]
test.lox:1:13: ) [RightParen]
test.lox:1:15: { [LeftBrace]
test.lox:2:5: return [Return]
test.lox:2:12: x [Ident]
test.lox:2:14: + [Plus]
test.lox:2:16: y [Ident]
test.lox:2:17: ; [Semicolon]
test.lox:3:1: } [RightBrace]
test.lox:5:1: print [Print]
test.lox:5:7: add [Ident]
test.lox:5:10: ( [LeftParen]
test.lox:5:11: 3 [Number]
test.lox:5:12: , [Comma]
test.lox:5:14: 4 [Number]
test.lox:5:15: ) [RightParen]
test.lox:5:16: ; [Semicolon]
test.lox:6:1:  [EOF]
```

### Pass program as string

```sh
golox -program 'fun add(x, y) { return x + y; } print add(5, 6);'
```

#### Output

```
11
```
