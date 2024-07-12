/// <reference types="tree-sitter-cli/dsl" />
// @ts-check

module.exports = grammar({
  name: "lox",

  extras: ($) => [/\s/, $.comment],

  word: ($) => $.identifier,

  rules: {
    program: ($) => repeat(choice($._declaration, $._statement)),

    // Declarations
    _declaration: ($) => choice($.variable_declaration, $.function_declaration),

    variable_declaration: ($) =>
      seq(
        "var",
        field("name", $.identifier),
        optional(seq("=", field("initialiser", $._expression))),
        ";",
      ),

    function_declaration: ($) =>
      seq(
        "fun",
        field("name", $.identifier),
        field("parameters", $.parameters),
        field("body", $.block_statement),
      ),

    parameters: ($) =>
      seq(
        "(",
        optional(seq(optional($.identifier), repeat(seq(",", $.identifier)))),
        ")",
      ),

    _statement: ($) =>
      choice(
        $.expression_statement,
        $.print_statement,
        $.block_statement,
        $.if_statement,
        $.while_statement,
        $.for_statement,
        $.break_statement,
        $.continue_statement,
        $.return_statement,
      ),

    // Statements
    expression_statement: ($) => seq($._expression, ";"),

    print_statement: ($) => seq("print", $._expression, ";"),

    block_statement: ($) =>
      seq("{", repeat(choice($._declaration, $._statement)), "}"),

    if_statement: ($) =>
      prec.right(
        seq(
          "if",
          "(",
          field("condition", $._expression),
          ")",
          field("then", $._statement),
          optional(seq("else", field("else", $._statement))),
        ),
      ),

    while_statement: ($) =>
      seq(
        "while",
        "(",
        field("condition", $._expression),
        ")",
        field("body", $._statement),
      ),

    for_statement: ($) =>
      seq(
        "for",
        "(",
        choice(
          field("initialiser", choice($._declaration, $.expression_statement)),
          ";",
        ),
        field("condition", optional($._expression)),
        ";",
        field("update", optional($._expression)),
        ")",
        field("body", $._statement),
      ),

    break_statement: () => seq("break", ";"),

    continue_statement: () => seq("continue", ";"),

    return_statement: ($) => seq("return", $._expression, ";"),

    // Expressions
    _expression: ($) =>
      choice(
        $._literal,
        $.function_expression,
        $.group_expression,
        $.identifier,
        $.call_expression,
        $.unary_expression,
        $.binary_expression,
        $.ternary_expression,
        $.assignment_expression,
      ),

    _literal: ($) => choice($.number, $.string, $.boolean, $.nil),

    number: (_) => /\d+(\.\d+)?/,

    string: (_) => /"[^"\r\n]*"/,

    boolean: (_) => choice("true", "false"),

    nil: (_) => "nil",

    function_expression: ($) =>
      seq(
        "fun",
        field("parameters", $.parameters),
        field("body", $.block_statement),
      ),

    group_expression: ($) => seq("(", field("expression", $._expression), ")"),

    identifier: (_) => /[a-zA-Z][a-zA-Z0-9_]*/,

    call_expression: ($) =>
      prec(
        11,
        seq(field("callee", $._expression), field("arguments", $.arguments)),
      ),

    arguments: ($) =>
      seq(
        "(",
        optional(
          seq(
            optional($._expression),
            repeat(prec(2, seq(",", $._expression))),
          ),
        ),
        ")",
      ),

    unary_expression: ($) =>
      prec.right(10, seq(choice("!", "-"), field("right", $._expression))),

    binary_expression: ($) =>
      choice(
        prec.left(
          1,
          seq(field("left", $._expression), ",", field("right", $._expression)),
        ),
        prec.left(
          4,
          seq(
            field("left", $._expression),
            "or",
            field("right", $._expression),
          ),
        ),
        prec.left(
          5,
          seq(
            field("left", $._expression),
            "and",
            field("right", $._expression),
          ),
        ),
        prec.left(
          6,
          seq(
            field("left", $._expression),
            choice("==", "!="),
            field("right", $._expression),
          ),
        ),
        prec.left(
          7,
          seq(
            field("left", $._expression),
            choice("<", "<=", ">", ">="),
            field("right", $._expression),
          ),
        ),
        prec.left(
          8,
          seq(
            field("left", $._expression),
            choice("+", "-"),
            field("right", $._expression),
          ),
        ),
        prec.left(
          9,
          seq(
            field("left", $._expression),
            choice("*", "/", "%"),
            field("right", $._expression),
          ),
        ),
      ),

    ternary_expression: ($) =>
      prec.right(
        3,
        seq(
          field("condition", $._expression),
          "?",
          field("then", $._expression),
          ":",
          field("else", $._expression),
        ),
      ),

    assignment_expression: ($) =>
      prec.right(
        2,
        seq(field("left", $.identifier), "=", field("right", $._expression)),
      ),

    comment: (_) => choice(seq("//", /.*/), seq("/*", repeat(/./), "*/")),
  },
});
