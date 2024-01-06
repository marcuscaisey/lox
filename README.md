# Golox

Golox is a Go implementation of the Lox programming language as defined in the book [Crafting
Interpreters](https://craftinginterpreters.com/).

## Grammar

Below is the grammar of Lox defined using the flavour of [Extended Backus–Naur
form](https://en.wikipedia.org/wiki/Extended_Backus%E2%80%93Naur_form) described in [Extensible
Markup Language (XML) 1.0 (Fifth Edition)](https://www.w3.org/TR/xml/#sec-notation).

```ebnf
expr                = comma_expr ;
comma_expr          = ternary_expr ( "," ternary_expr )* ;
ternary_expr        = equality_expr ( "?" expr ":" ternary_expr )? ;
equality_expr       = relational_expr ( ( "==" | "!=" ) relational_expr )* ;
relational_expr     = additive_expr ( ( "<" | "<=" | ">" | ">=" ) additive_expr )* ;
additive_expr       = multiplicative_expr ( ( "+" | "-" ) multiplicative_expr )* ;
multiplicative_expr = unary_expr ( ( "*" | "/" ) unary_expr )* ;
unary_expr          = ( ( "!" | "-" ) unary_expr ) | primary_expr ;
primary_expr        = NUMBER | STRING | "true" | "false" | "nil" | "(" expr ")"
                    /* Error productions */
                    | ( "==" | "!=" ) relational_expr
                    | ( "<" | "<=" | ">" | ">=" ) additive_expr
                    | "+" multiplicative_expr
                    | ( "*" | "/" ) unary_expr ;
```
