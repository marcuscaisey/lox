# Golox [![CI](https://github.com/marcuscaisey/golox/actions/workflows/ci.yml/badge.svg)](https://github.com/marcuscaisey/golox/actions/workflows/ci.yml)

Golox is a Go implementation of the Lox programming language as defined in the book [Crafting
Interpreters](https://craftinginterpreters.com/).

Working Lox code examples can be found under [test/testdata](test/testdata).

## Installation

```sh
go install github.com/marcuscaisey/golox@latest
```

## Usage

```
Usage: golox [options] [script]

Options:
  -c string
        Program passed in as string
  -p    Print the AST only
```

If no script is provided, a REPL is started, otherwise the supplied script is executed.

## Language

### Types

Lox has four primitive types:

| Name   | Description                  | Literal syntax | Truthiness                           |
| ------ | ---------------------------- | -------------- | ------------------------------------ |
| number | 64-bit floating point number | `123.4`        | `false` if `0`, `true` otherwise     |
| string | UTF-8 string                 | `"hello"`      | `false` if `""`, `true` otherwise    |
| bool   | Boolean value                | `true` `false` | `false` if `false`, `true` otherwise |
| nil    | Absence of a value           | `nil`          | `false`                              |

#### Unary Operations

| Operator | Operand type | Result type | Description                           |
| -------- | ------------ | ----------- | ------------------------------------- |
| !        | All          | `bool`      | Negates the truthiness of the operand |
| -        | `number`     | `number`    | Negates the operand                   |

#### Binary Operations

| Operator  | Operand 1 type | Operand 2 type | Result type               | Description                                                        |
| --------- | -------------- | -------------- | ------------------------- | ------------------------------------------------------------------ |
| \*        | `number`       | `number`       | `number`                  | Multiplies the operands                                            |
| \*        | `number`       | `string`       | `string`                  | Repeats the string                                                 |
| /         | `number`       | `number`       | `number`                  | Divides the operands                                               |
| +         | `number`       | `number`       | `number`                  | Adds the operands                                                  |
| +         | `string`       | `string`       | `string`                  | Concatenates the operands                                          |
| -         | `number`       | `number`       | `number`                  | Subtracts the operands                                             |
| < <= > >= | `number`       | `number`       | `bool`                    | Compares the operands                                              |
| < <= > >= | `string`       | `string`       | `bool`                    | Compares the operands lexicographically                            |
| == !=     | All            | All            | `bool`                    | Compares the operands and their types                              |
| ,         | All            | All            | Type of the right operand | Evaluates the left then right operand<br>Returns the second result |

#### Ternary Operation

The ternary operator `?:` is a special operator that takes three operands. It evaluates the first
operand, and if it is truthy, it evaluates and returns the second operand. Otherwise, it evaluates
and returns the third operand.

```lox
print true ? 1 : 2; // prints 1
print "" ? 1 : 2;   // prints 2
```

#### Operator Precedence and Associativity

From highest to lowest:

| Operators | Associativity |
| --------- | ------------- |
| ! -       | right-to-left |
| \* /      | left-to-right |
| + -       | left-to-right |
| < <= > >= | left-to-right |
| == !=     | left-to-right |
| ?:        | right-to-left |
| ,         | left-to-right |

Any expression can be wrapped in `()` to override the default precedence.

### Comments

Both single line and multi line comments are supported.

```lox
// This is a single line comment
print "Hello, World!"; // This is also a single line comment

/*
This is a multi line comment
*/

/*
 * /* Nested multi-line are also supported */
 */

print 1 /* Multi line comments can be used anywhere */ + 2;
```

### Statements

#### Expression statement

An expression statement is an expression followed by a semicolon. It evaluates the expression and
then discards the result.

```lox
1 + 2;
```

#### Print statement

A print statement is the keyword `print` followed by an expression followed by a semicolon. It
evaluates the expression and prints the result to stdout.

```lox
print 1 + 2; // prints 3
```

### Grammar

Below is the grammar of Lox defined using the flavour of [Extended Backus–Naur
form](https://en.wikipedia.org/wiki/Extended_Backus%E2%80%93Naur_form) described in [Extensible
Markup Language (XML) 1.0 (Fifth Edition)](https://www.w3.org/TR/xml/#sec-notation).

```ebnf
program = decl* EOF ;

decl     = var_decl | stmt ;
var_decl = "var" IDENT ( "=" expr )? ";" ;

stmt       = expr_stmt | print_stmt ;
expr_stmt  = expr ";" ;
print_stmt = "print" expr ";" ;

expr                = comma_expr ;
comma_expr          = assignment_expr ( "," assignment_expr )* ;
assignment_expr     = IDENT "=" assignment_expr | ternary_expr ;
ternary_expr        = equality_expr ( "?" expr ":" ternary_expr )? ;
equality_expr       = relational_expr ( ( "==" | "!=" ) relational_expr )* ;
relational_expr     = additive_expr ( ( "<" | "<=" | ">" | ">=" ) additive_expr )* ;
additive_expr       = multiplicative_expr ( ( "+" | "-" ) multiplicative_expr )* ;
multiplicative_expr = unary_expr ( ( "*" | "/" ) unary_expr )* ;
unary_expr          = ( "!" | "-" ) unary_expr | primary_expr ;
primary_expr        = NUMBER | STRING | "true" | "false" | "nil" | "(" expr ")" | IDENT
                    /* Error productions */
                    | ( "==" | "!=" ) relational_expr
                    | ( "<" | "<=" | ">" | ">=" ) additive_expr
                    | "+" multiplicative_expr
                    | ( "*" | "/" ) unary_expr ;
```
