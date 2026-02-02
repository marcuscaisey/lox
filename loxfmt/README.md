# loxfmt

loxfmt is a code formatter for the Lox programming language.

## Installation

```sh
go install github.com/marcuscaisey/lox/loxfmt@latest
```

## Usage

```
Usage: loxfmt [options] [<path>]

If no path is provided, the file is read from stdin.

Options:
  -ast
        Print the AST
  -help
        Print this message
  -write
        Write result to (source) file instead of stdout
```

If no path is provided, the file is read from stdin.

## Examples

### Format stdin

```sh
echo 'fun add(x, y) { return x + y; } print add(1, 2);' | loxfmt
```

#### Output

```sh
fun add(x, y) {
    return x + y;
}
print add(1, 2);
```

### Format file

```sh
echo 'fun add(x, y) { return x + y; } print add(3, 4);' > test.lox
loxfmt test.lox
```

#### Output

```
fun add(x, y) {
    return x + y;
}
print add(3, 4);
```

### Print AST

```sh
echo 'fun add(x, y) { return x + y; } print add(5, 6);' | loxfmt -ast
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
      5
      6
    ])))
```

### Format file in-place

```sh
echo 'fun add(x, y) { return x + y; } print add(7, 8);' > test.lox
loxfmt -write test.lox
cat test.lox
```

#### Output

```
fun add(x, y) {
    return x + y;
}
print add(7, 8);
```
