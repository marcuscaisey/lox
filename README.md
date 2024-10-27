# Lox [![CI](https://github.com/marcuscaisey/lox/actions/workflows/ci.yml/badge.svg)](https://github.com/marcuscaisey/lox/actions/workflows/ci.yml)

Lox is the dynamically typed programming language defined in the book [Crafting
Interpreters](https://craftinginterpreters.com). This repository contains:

- A Go implementation of the language: [golox](golox)
- A grammar for [tree-sitter](https://github.com/tree-sitter/tree-sitter):
  [tree-sitter-lox](tree-sitter-lox)
- A formatter for the language: [loxfmt](loxfmt)
- A language server: [loxls](loxls)

Working Lox code examples can be found under [test/testdata](test/testdata).

## Test Suite

golox and loxfmt are tested against a suite of test files defined under
[test/testdata](test/testdata). golox is tested by running each test file and comparing the output
with the expected output defined in the file. loxfmt is tested by formatting each test file and
asserting that the contents of the file are unchanged.

### Test File Format

Test files are regular Lox files containing comments which describe the expectations of the test:

- `// prints: <value>` defines a string that should be printed to stdout.
- `// error: <message>` defines an error message that should be printed to stderr.

Both special comments can appear multiple times in a test file.

For example:

```lox
print 3 % 2; // prints: 1
print 3 % 3; // prints: 0
print 3.5 % 2; // prints: 1.5
print 3 % 1.5; // prints: 0
```

If a `// noformat` comment appears at the start of a test file, the file will be not be formatted.
This is useful for files which contain syntax errors and can't be parsed.

### Running the Tests

Run all tests:

```sh
make test
```

Run the golox or loxfmt tests individually:

```sh
make test_golox
make test_loxfmt
```

Run a specific test:

```sh
make test_golox RUN=TestInterpreter/Number/Modulo
make test_loxfmt RUN=TestFormatter/Number/Modulo
```

### Updating the Test Expectations

The expectations of each test can be updated to match the current implementation by running either
of the following commands:

```sh
make update_golox_tests
make update_loxfmt_tests
```

As with running the tests, you can update the expectations of a specific test as well:

```sh
make update_golox_tests RUN=TestInterpreter/Number/Modulo
make update_loxfmt_tests RUN=TestFormatter/Number/Modulo
```

## Language

### Extra Features

Implemented is a superset of the Lox language defined in the book where the extra features originate
either from challenges in the book or my own ideas.

#### Challenges

- [Comma expression](#Binary-Expression) - [Parsing Expressions](https://craftinginterpreters.com/parsing-expressions.html#challenges)
- [Ternary expression](#Ternary-Expression) - [Parsing Expressions](https://craftinginterpreters.com/parsing-expressions.html#challenges)
- Error productions for [binary expressions](#Grammar) - [Parsing Expressions](https://craftinginterpreters.com/parsing-expressions.html#challenges)
- [`<`, `<=`, `>`, `>=` operators for strings](#Binary-Expression) - [Evaluating Expressions](https://craftinginterpreters.com/evaluating-expressions.html#challenges)
- [Division by zero handling](#Binary-Expression) - [Evaluating Expressions](https://craftinginterpreters.com/evaluating-expressions.html#challenges)
- Displaying of evaluated expressions in REPL - [Statements and State](https://craftinginterpreters.com/statements-and-state.html#challenges)
- [Runtime error](#Declarations) for accessing uninitialised variable - [Statements and State](https://craftinginterpreters.com/statements-and-state.html#challenges)
- [`break` statement](#Break-Statement) - [Control Flow](https://craftinginterpreters.com/control-flow.html#challenges)
- [Function expression](#Function-Expression) - [Functions](https://craftinginterpreters.com/functions.html#challenges)
- Reporting of [unused variables](#Blank-Identifier) - [Resolving and Binding](https://craftinginterpreters.com/resolving-and-binding.html#challenges)
- [Static method](#Class-Declaration) - [Classes](https://craftinginterpreters.com/classes.html#challenges)
- [Property getter method](#Class-Declaration) - [Classes](https://craftinginterpreters.com/classes.html#challenges)

#### Own Ideas

- [`%` operator](#Binary-Expression)
- [`continue` statement](#Continue-Statement)
- [`type` built-in function](#Built-in-Functions)
- [Error messages point to location of error in source code](#Errors)
- [Runtime error message includes stack trace](#Errors)
- [`error` built-in function](#Built-in-Functions)
- [Property setter method](#Class-Declaration)

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
print 123.4; // prints: 123.4
print "hello"; // prints: hello
print false; // prints: false
print nil; // prints: nil
```

#### Unary Expression

A unary expression is an operator followed by a single operand.

| Operator | Operand  | Result   | Description                           |
| -------- | -------- | -------- | ------------------------------------- |
| !        | All      | `bool`   | Negates the truthiness of the operand |
| -        | `number` | `number` | Negates the operand                   |

```lox
print !""; // prints: true
print -1; // prints: -1
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
print 2 * 3.5; // prints: 7
print 3 * "ab"; // prints: "ababab"
print 10 / 2; // prints: 5
print 3.5 % 2; // prints: 1.5
print 1 + 2; // prints: 3
print "a" + "b"; // prints: "ab"
print 3 - 1; // prints: 2
print 1 < 2; // prints: true
print "a" > "b"; // prints: false
print 1 == "1"; // prints: false
print 1 and "a"; // prints: a
print 1 or 2; // prints: 1
print 1, 2; // prints: 2
```

#### Ternary Expression

The ternary operator `?:` is a special operator that takes three operands. It evaluates the first
operand, and if it is truthy, it evaluates and returns the second operand. Otherwise, it evaluates
and returns the third operand.

```lox
print true ? 1 : 2; // prints: 1
print "" ? 1 : 2;   // prints: 2
```

#### Variable Expression

A variable expression produces the value of a variable.

```lox
var a = 1;
print a; // prints: 1
```

#### Assignment Expression

An assignment expression assigns a value to a variable and produces the value.

```lox
var a;
a = 1;
print a; // prints: 1
print a = 2; // prints: 2
print a; // prints: 2
```

#### Call Expression

A call expression calls a function with arguments.

```lox
fun add(a, b) {
    return a + b;
}

print add(1, 2); // prints: 3
```

#### Get Expression

A get expression produces the value of a property of an object.

```lox
class Foo {
    init(bar) {
        this.bar = bar;
    }
}
var foo = Foo(1);

print foo.bar; // prints: 1
```

#### Set Expression

A set expression assigns a value to a property of an object and produces the value.

```lox
class Foo {}
var foo = Foo();

foo.bar = 1;
print foo.bar; // prints: 1
print foo.bar = 2; // prints: 2
print foo.bar; // prints: 2
```

#### Function Expression

A function expression creates an anonymous function.

```lox
var add = fun(a, b) {
    return a + b;
};

print add(1, 2); // prints: 3
```

#### Operator Precedence and Associativity

From highest to lowest:

| Operators | Associativity |
| --------- | ------------- |
| () .      | left-to-right |
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
print 1 + 2; // prints: 3
```

#### Block Statement

A block statement groups multiple statements into a single statement. It is used to create a new
lexical scope.

```lox
var a = "global a";
var b = "global b";
{
    var a = "outer a";
    print a; // prints: outer a
    print b; // prints: global b
}
print a; // prints: global a
print b; // prints: global b
```

#### If Statement

An if statement evaluates an expression and executes a statement if the expression is truthy. An
optional else statement can be provided to execute a statement if the expression is falsy.

```lox
if (1 < 2) {
    print "1 is less than 2"; // prints: 1 is less than 2
}

if (1 > 2) {
    print "1 is greater than 2";
} else {
    print "1 is not greater than 2"; // prints: 1 is not greater than 2
}

if (1 > 2) {
    print "1 is greater than 2";
} else if (3 < 4) {
    print "3 is less than 4"; // prints: 3 is less than 4
}
```

#### While Statement

A while statement repeatedly executes a statement while the provided expression is truthy.

```lox
var i = 0;
while (i < 3) {
    // prints: 0
    // prints: 1
    // prints: 2
    print i;
    i = i + 1;
}
```

#### For Statement

A for statement is syntactic sugar for a while statement which initialises a variable before the
loop and modifies it at the end of each iteration.

The following while statement:

```lox
var i = 0;
while (i < 3) {
    // prints: 0
    // prints: 1
    // prints: 2
    print i;
    i = i + 1;
}
```

is equivalent to the following for statement:

```lox
for (var i = 0; i < 3; i = i + 1) {
    // prints: 0
    // prints: 1
    // prints: 2
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
    // prints: 0
    // prints: 1
    // prints: 2
    print i;
}
```

#### Break Statement

A break statement immediately exits the innermost enclosing loop.

```lox
for (var i = 0; i < 3; i = i + 1) {
    if (i == 1) {
        break;
    }
    print i; // prints: 0
}
```

#### Continue Statement

A continue statement immediately jumps to the end of the innermost enclosing for or while loop.

```lox
for (var i = 0; i < 5; i = i + 1) {
    if (i % 2 == 1) {
        continue;
    }
    // prints: 0
    // prints: 2
    // prints: 4
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

print add(1, 2); // prints: 3
greet(); // prints: Hello, World!
```

### Declarations

Declarations are constructs that bind an identifier (name) to a value. It is not valid to:

- declare a [non-blank](#blank-identifier) identifier more than once in the same lexical scope.
- use a [non-blank](#blank-identifier) identifier before it has been declared.
- use a declared identifier which has not been assigned a value (defined).
- declare a [non-blank](#blank-identifier) identifier in a local scope and not use it.

#### Variable Declaration

A variable declaration declares an identifier which can be assigned a value. You can optionally
assign (define) an initial value to the variable.

```lox
var a;
print a; // prints: nil
var b = 1;
print b; // prints: 1
```

#### Function Declaration

A function declaration declares a function which can be called with arguments. The function body is
a block statement which can return a value to the caller. A function which does not return a value
implicitly returns `nil`.

```lox
fun add(a, b) {
    return a + b;
}

print add(1, 2); // prints: 3
```

#### Class Declaration

A class declaration declares a class which can be instantiated to create objects. The class body is
a block which can contain method declarations. `this` is a special identifier which can be used
inside a method body to refer to the instance that the method was accessed from. The `init` method
is a special method which is called when an object is instantiated.

```lox
class Point {
    init(x, y) {
        this.x = x;
        this.y = y;
    }

    move(dx, dy) {
        this.x = this.x + dx;
        this.y = this.y + dy;
    }
}

var p1 = Point(1, 2);
var p2 = Point(3, 4);
p1.move(5, 6);
p2.move(7, 8);
print p1.x; // prints: 6
print p1.y; // prints: 8
print p2.x; // prints: 10
print p2.y; // prints: 12
```

Methods can be declared as static by prefixing the declaration with `static`. Static methods are
accessed from the class itself rather than the instance. `this` inside a static method refers to the
class.

```lox
class Math {
    static square(x) {
        return x * x;
    }
}

print Math.square(2); // prints: 4
```

Methods can be declared as property getters and setters by prefixing the declaration with `get` and
`set` respectively. Property getters and setters are accessed like properties but are actually calls
to the getter and setter methods. Property getters and setters can also be static.

```lox
var PI = 3;

class Circle {
    init(radius) {
        this.radius = radius;
    }

    get radius() {
        return this._radius;
    }

    set radius(value) {
        if (value <= 0) {
            error("radius must be positive");
        }
        this._radius = value;
    }

    get area() {
        return PI * this.radius * this.radius;
    }
}

var c = Circle(2);
print c.radius; // prints: 2
print c.area; // prints: 12
c.radius = -1;
print c.radius; // error: radius must be positive
```

#### Blank Identifier

The blank identifier `_` is a special identifier which:

- can be declared more than once in the same lexical scope.
- can be used before it has been declared.
- can be declared but not used.
- cannot be used in a non-assignment expression.

### Comments

Comments are bits of text in the source code that are ignored when evaluating the program. They can
appear:

- anywhere a declaration can appear.
- at the end of a statement.

```lox
// This is a comment
print "Hello, World!"; // This is also a comment
```

### Errors

If any errors are found before execution of a program has begun, they will all be reported and
execution will not begin.

```lox
class Foo {
    init(a) {
        this.a = a;
        return this;
    }
}

fun add(y) {
    return this.x + y;
}
```

```
test.lox:4:5: error: init() cannot return a value
    return this;
    ~~~~~~~~~~~~
test.lox:9:10: error: 'this' can only be used inside a method definition
  return this.x + y;
         ~~~~
```

If an error occurs during the execution of a program, execution will halt and the error will be
reported along with a stack trace.

```lox
var PI = 3;

class Circle {
    init(radius) {
        this.radius = radius;
    }

    get radius() {
        return this._radius;
    }

    set radius(value) {
        if (value <= 0) {
            error("radius must be positive");
        }
        this._radius = value;
    }

    get area() {
        return PI * this.radius * this.radius;
    }
}

fun main() {
    var c = Circle(-1);
    print c.area;
}

main();
```

```
test.lox:14:7: error: radius must be positive
      error("radius must be positive");
      ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Stack Trace (most recent call first):
  test.lox:14:7  in set Circle.radius error("radius must be positive");
  test.lox:5:10  in Circle.init       this.radius = radius;
  test.lox:25:11 in main              var c = Circle(-1);
  test.lox:29:1                       main();
```

### Built-in Functions

Lox has the following built-in functions.

| Name           | Returns  | Description                                         |
| -------------- | -------- | --------------------------------------------------- |
| `clock()`      | `number` | Returns the number of seconds since the Unix epoch. |
| `type(object)` | `string` | Returns the type of the object.                     |
| `error(msg)`   | `nil`    | Throws a runtime error with the message.            |

### Grammar

Below is the grammar of Lox defined using the flavour of [Extended Backus–Naur
form](https://en.wikipedia.org/wiki/Extended_Backus%E2%80%93Naur_form) described in [Extensible
Markup Language (XML) 1.0 (Fifth Edition)](https://www.w3.org/TR/xml/#sec-notation).

```ebnf
program =  decl* EOF ;

decl       = var_decl | fun_decl | class_decl | stmt ;
var_decl   = "var" IDENT ( "=" expr )? ";" ;
fun_decl   = "fun" function ;
function   = IDENT "(" parameters? ")" block_stmt ;
parameters = IDENT ( "," IDENT )* ;
class_decl = "class" IDENT "{" method* "}" ;
method     = "static"? ( "get" | "set" )? function ;

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
assignment_expr     = ( postfix_expr "." )? IDENT "=" assignment_expr | ternary_expr ;
ternary_expr        = logical_or_expr ( "?" expr ":" ternary_expr )? ;
logical_or_expr     = logical_and_expr ( "or" logical_and_expr )* ;
logical_and_expr    = equality_expr ( "and" equality_expr )* ;
equality_expr       = relational_expr ( ( "==" | "!=" ) relational_expr )* ;
relational_expr     = additive_expr ( ( "<" | "<=" | ">" | ">=" ) additive_expr )* ;
additive_expr       = multiplicative_expr ( ( "+" | "-" ) multiplicative_expr )* ;
multiplicative_expr = unary_expr ( ( "*" | "/" | "%" ) unary_expr )* ;
unary_expr          = ( "!" | "-" ) unary_expr | postfix_expr ;
postfix_expr        = primary_expr ( "(" arguments? ")" | "." IDENT )* ;
arguments           = assignment_expr ( "," assignment_expr )* ;
primary_expr        = NUMBER | STRING | "true" | "false" | "nil" | IDENT | "this" | group_expr
                    | fun_expr
                    /* Error productions */
                    | ( "==" | "!=" ) relational_expr
                    | ( "<" | "<=" | ">" | ">=" ) additive_expr
                    | "+" multiplicative_expr
                    | ( "*" | "/" ) unary_expr ;
group_expr          = "(" expr ")" ;
fun_expr            = "fun" "(" parameters? ")" block_stmt ;
```
