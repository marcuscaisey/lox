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

### Expressions

Expressions are constructs that produce a value.

#### Literal Expression

A literal expression produces a value directly.

```lox
print 123.4; // prints 123.4
print "hello"; // prints hello
print false; // prints false
print nil; // prints nil
```

#### Variable Expression

A variable expression produces the value of a variable. It is not valid to access an uninitialised
variable.

```lox
var a = 1;
print a; // prints 1
```

#### Call Expression

A call expression calls a function with arguments.

```lox
fun add(a, b) {
    return a + b;
}

print add(1, 2); // prints 3
```

#### Assignment Expression

An assignment expression assigns a value to a variable and produces the value.

```lox
var a;
a = 1;
print a; // prints 1
print a = 2; // prints 2
print a; // prints 2
```

#### Unary Expression

A unary expression is an operator followed by a single operand.

| Operator | Operand  | Result   | Description                           |
| -------- | -------- | -------- | ------------------------------------- |
| !        | All      | `bool`   | Negates the truthiness of the operand |
| -        | `number` | `number` | Negates the operand                   |

```lox
print !""; // prints true
print -1; // prints -1
```

#### Binary Expression

A binary expression is an operator surrounded by two operands.

| Operator  | Operand 1 | Operand 2 | Result                    | Description                                                            |
| --------- | --------- | --------- | ------------------------- | ---------------------------------------------------------------------- |
| \*        | `number`  | `number`  | `number`                  | Multiplies the operands                                                |
| \*        | `number`  | `string`  | `string`                  | Repeats the string                                                     |
| /         | `number`  | `number`  | `number`                  | Divides the operands                                                   |
| %         | `number`  | `number`  | `number`                  | Returns the remainder of the division of the operands                  |
| +         | `number`  | `number`  | `number`                  | Adds the operands                                                      |
| +         | `string`  | `string`  | `string`                  | Concatenates the operands                                              |
| -         | `number`  | `number`  | `number`                  | Subtracts the operands                                                 |
| < <= > >= | `number`  | `number`  | `bool`                    | Compares the operands                                                  |
| < <= > >= | `string`  | `string`  | `bool`                    | Compares the operands lexicographically                                |
| == !=     | All       | All       | `bool`                    | Compares the operands and their types                                  |
| and       | `bool`    | `bool`    | `bool`                    | Returns the second operand if the first is truthy, otherwise the first |
| or        | `bool`    | `bool`    | `bool`                    | Returns the first operand if it is truthy, otherwise the second        |
| ,         | All       | All       | Type of the right operand | Evaluates the left then right operand<br>Returns the second result     |

```lox
print 2 * 3.5; // prints 7
print 3 * "ab"; // prints "ababab"
print 10 / 2; // prints 5
print 3.5 % 2; // prints 1.5
print 1 + 2; // prints 3
print "a" + "b"; // prints "ab"
print 3 - 1; // prints 2
print 1 < 2; // prints true
print "a" > "b"; // prints false
print 1 == "1"; // prints false
print 1 and "a"; // prints a
print 1 or 2; // prints 1
print 1, 2; // prints 2
```

#### Ternary Expression

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
| \* / %    | left-to-right |
| + -       | left-to-right |
| < <= > >= | left-to-right |
| == !=     | left-to-right |
| ?:        | right-to-left |
| =         | right-to-left |
| ,         | left-to-right |

Any expression can be wrapped in `()` to override the default precedence.

### Statements

Statements are constructs that perform some action.

#### Expression Statement

An expression statement evaluates an expression and discards the result.

```lox
1 + 2;
```

#### Print Statement

A print statement evaluates an expression and prints the result.

```lox
print 1 + 2; // prints 3
```

#### Block Statement

A block statement groups multiple statements into a single statement. It is used to create a new
lexical scope.

```lox
var a = "global a";
var b = "global b";
{
    var a = "outer a";
    print a; // prints outer a
    print b; // prints global b
}
print a; // prints global a
print b; // prints global b
```

#### If Statement

An if statement evaluates an expression and executes a statement if the expression is truthy. An
optional else statement can be provided to execute a statement if the expression is falsy.

```lox
if (1 < 2) {
    print "1 is less than 2"; // prints 1 is less than 2
}


if (1 > 2) {
    print "1 is greater than 2";
} else {
    print "1 is not greater than 2"; // prints 1 is not greater than 2
}

if (1 > 2) {
    print "1 is greater than 2";
} else if (3 < 4) {
    print "3 is less than 4"; // prints 3 is less than 4
}
```

#### While Statement

A while statement repeatedly executes a statement while the provided expression is truthy.

```lox
var i = 0;
while (i < 3) {
    // prints 0
    // prints 1
    // prints 2
    print i;
    i = i + 1;
}
```

#### Break Statement

A break statement immediately exits the innermost enclosing loop.

```lox
for (var i = 0; i < 3; i++) {
    if (i == 1) {
        break;
    }
    print i; // prints 0
}
```

#### Continue Statement

A continue statement immediately jumps to the end of the innermost enclosing for or while loop.

```lox
for (var i = 0; i < 5; i++) {
    if (i % 2 == 1) {
        continue;
    }
    // prints 0
    // prints 2
    // prints 4
    print i;
}
```

#### Return Statement

A return statement immediately exits the enclosing function and optionally returns a value to the
caller.

```lox
fun add(a, b) {
    return a + b;
}

fun greet() {
    print "Hello, World!";
    return;
    print "This is unreachable";
}

print add(1, 2); // prints 3
greet(); // prints Hello, World!
```

#### For Statement

A for statement is syntactic sugar for a while statement which initialises a variable before the
loop and modifies it at the end of each iteration.

The following while statement:

```lox
var i = 0;
while (i < 3) {
    // prints 0
    // prints 1
    // prints 2
    print i;
    i = i + 1;
}
```

is equivalent to the following for statement:

```lox
for (var i = 0; i < 3; i = i + 1) {
    // prints 0
    // prints 1
    // prints 2
    print i;
}
```

All three sections of the for statement are optional. The following is an infinite loop:

```lox
for (;;) {
    print "infinite loop";
}
```

The initialisation section can either be a variable declaration or an expression.

```lox
var i;
for (i = 0; i < 3; i = i + 1) {
    // prints 0
    // prints 1
    // prints 2
    print i;
}
```

### Declarations

Declarations are constructs that bind a name to a value.

#### Variable Declaration

A variable declaration declares a name which can be assigned a value. You can optionally assign an
initial value to the variable.

```lox
var a;
print a; // prints nil
var b = 1;
print b; // prints 1
```

#### Function Declaration

A function declaration declares a name which can be called with arguments. The function body is a
block statement which can return a value to the caller. A function which does not return a value
implicitly returns `nil`.

```lox
fun add(a, b) {
    return a + b;
}

print add(1, 2); // prints 3
```

### Comments

Comments are bits of text in the source code that are ignored when evaluating the program. Both
single line and multi line comments are supported.

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

### Built-in Functions

Lox has the following built-in functions.

| Name      | Returns  | Description                                         |
| --------- | -------- | --------------------------------------------------- |
| `clock()` | `number` | Returns the number of seconds since the Unix epoch. |

### Grammar

Below is the grammar of Lox defined using the flavour of [Extended Backus–Naur
form](https://en.wikipedia.org/wiki/Extended_Backus%E2%80%93Naur_form) described in [Extensible
Markup Language (XML) 1.0 (Fifth Edition)](https://www.w3.org/TR/xml/#sec-notation).

```ebnf
program =  decl* EOF ;

decl       = var_decl | fun_decl | stmt ;
var_decl   = "var" IDENT ( "=" expr )? ";" ;
fun_decl   = "fun" function ;
function   = IDENT "(" parameters? ")" block_stmt ;
parameters = IDENT ( "," IDENT )* ;

stmt          = expr_stmt | print_stmt | block_stmt | if_stmt | while_stmt | for_stmt | break_stmt
              | continue_stmt ;
expr_stmt     = expr ";" ;
print_stmt    = "print" expr ";" ;
block_stmt    = "{" decl* "}" ;
if_stmt       = "if" "(" expr ")" stmt ( "else" stmt )? ;
while_stmt    = "while" "(" expr ")" stmt ;
for_stmt      = "for" "(" ( var_decl | expr_stmt | ";" ) expr? ";" expr? ")" stmt ;
break_stmt    = "break" ";" ;
continue_stmt = "continue" ";" ;
return_stmt   = "return" expression? ";" ;

expr                = comma_expr ;
comma_expr          = assignment_expr ( "," assignment_expr )* ;
assignment_expr     = IDENT "=" assignment_expr | ternary_expr ;
ternary_expr        = logical_or_expr ( "?" expr ":" ternary_expr )? ;
logical_or_expr     = logical_and_expr ( "or" logical_and_expr )* ;
logical_and_expr    = equality_expr ( "and" equality_expr )* ;
equality_expr       = relational_expr ( ( "==" | "!=" ) relational_expr )* ;
relational_expr     = additive_expr ( ( "<" | "<=" | ">" | ">=" ) additive_expr )* ;
additive_expr       = multiplicative_expr ( ( "+" | "-" ) multiplicative_expr )* ;
multiplicative_expr = unary_expr ( ( "*" | "/" | "%" ) unary_expr )* ;
unary_expr          = ( "!" | "-" ) unary_expr | call_expr ;
call_expr           = primary_expr ( "(" arguments? ")" )* ;
arguments           = assignment_expr ( "," assignment_expr )* ;
primary_expr        = NUMBER | STRING | "true" | "false" | "nil" | "(" expr ")" | IDENT
                    /* Error productions */
                    | ( "==" | "!=" ) relational_expr
                    | ( "<" | "<=" | ">" | ">=" ) additive_expr
                    | "+" multiplicative_expr
                    | ( "*" | "/" ) unary_expr ;
```
