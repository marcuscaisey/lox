/// <reference types="tree-sitter-cli/dsl" />
// @ts-check

module.exports = grammar({
  name: "lox",

  extras: ($) => [/\s/, $.comment],

  precedences: () => [
    [
      "postfix",
      "unary",
      "multiplicative",
      "additive",
      "relational",
      "equality",
      "logical_and",
      "logical_or",
      "ternary",
      "arguments",
      "assignment",
      "comma",
    ],
  ],

  word: ($) => $.identifier,

  rules: {
    program: ($) => repeat(choice($._declaration, $._statement)),

    // Declarations
    _declaration: ($) =>
      choice(
        $.variable_declaration,
        $.function_declaration,
        $.class_declaration,
      ),

    variable_declaration: ($) =>
      seq(
        "var",
        field("name", $.identifier),
        optional(seq("=", field("initialiser", $._expression))),
        ";",
      ),

    function_declaration: ($) => seq("fun", $._function),

    class_declaration: ($) =>
      seq("class", field("name", $.identifier), field("body", $.class_body)),

    class_body: ($) => seq("{", repeat($.method_declaration), "}"),

    method_declaration: ($) => seq(optional($.modifiers), $._function),

    modifiers: () => repeat1(choice("static")),

    _function: ($) =>
      seq(
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
        $.this_expression,
        $.call_expression,
        $.get_expression,
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

    identifier: (_) => /[a-zA-Z_][a-zA-Z0-9_]*/,

    this_expression: (_) => "this",

    call_expression: ($) =>
      prec(
        "postfix",
        seq(field("callee", $._expression), field("arguments", $.arguments)),
      ),

    get_expression: ($) =>
      prec(
        "postfix",
        seq(field("object", $._expression), ".", field("name", $.identifier)),
      ),

    arguments: ($) =>
      seq(
        "(",
        optional(
          seq(
            optional($._expression),
            repeat(prec("arguments", seq(",", $._expression))),
          ),
        ),
        ")",
      ),

    unary_expression: ($) =>
      prec.right("unary", seq(choice("!", "-"), field("right", $._expression))),

    binary_expression: ($) =>
      choice(
        prec.left(
          "comma",
          seq(field("left", $._expression), ",", field("right", $._expression)),
        ),
        prec.left(
          "logical_or",
          seq(
            field("left", $._expression),
            "or",
            field("right", $._expression),
          ),
        ),
        prec.left(
          "logical_and",
          seq(
            field("left", $._expression),
            "and",
            field("right", $._expression),
          ),
        ),
        prec.left(
          "equality",
          seq(
            field("left", $._expression),
            choice("==", "!="),
            field("right", $._expression),
          ),
        ),
        prec.left(
          "relational",
          seq(
            field("left", $._expression),
            choice("<", "<=", ">", ">="),
            field("right", $._expression),
          ),
        ),
        prec.left(
          "additive",
          seq(
            field("left", $._expression),
            choice("+", "-"),
            field("right", $._expression),
          ),
        ),
        prec.left(
          "multiplicative",
          seq(
            field("left", $._expression),
            choice("*", "/", "%"),
            field("right", $._expression),
          ),
        ),
      ),

    ternary_expression: ($) =>
      prec.right(
        "ternary",
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
        "assignment",
        seq(
          field("left", choice($.identifier, $.get_expression)),
          "=",
          field("right", $._expression),
        ),
      ),

    comment: (_) => choice(seq("//", /.*/), seq("/*", repeat(/./), "*/")),
  },
});
