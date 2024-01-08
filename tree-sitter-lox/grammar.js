module.exports = grammar({
  name: "lox",

  rules: {
    program: ($) => repeat($._statement),

    _statement: ($) => choice($.expression_statement, $.print_statement),

    expression_statement: ($) => seq($._expression, ";"),

    print_statement: ($) => seq("print", $._expression, ";"),

    _expression: ($) =>
      choice(
        $.unary_expression,
        $.binary_expression,
        $.ternary_expression,
        $._literal_expression,
        $.group_expression,
      ),

    binary_expression: ($) =>
      choice(
        prec.left(
          1,
          seq(field("left", $._expression), ",", field("right", $._expression)),
        ),
        prec.left(
          3,
          seq(
            field("left", $._expression),
            choice("==", "!="),
            field("right", $._expression),
          ),
        ),
        prec.left(
          4,
          seq(
            field("left", $._expression),
            choice("<", "<=", ">", ">="),
            field("right", $._expression),
          ),
        ),
        prec.left(
          5,
          seq(
            field("left", $._expression),
            choice("+", "-"),
            field("right", $._expression),
          ),
        ),
        prec.left(
          6,
          seq(
            field("left", $._expression),
            choice("*", "/"),
            field("right", $._expression),
          ),
        ),
      ),

    ternary_expression: ($) =>
      prec.right(
        2,
        seq(
          field("condition", $._expression),
          "?",
          field("then", $._expression),
          ":",
          field("else", $._expression),
        ),
      ),

    unary_expression: ($) =>
      prec.right(7, seq(choice("!", "-"), field("right", $._expression))),

    group_expression: ($) => seq("(", field("expression", $._expression), ")"),

    _literal_expression: ($) => choice($.number, $.string, $.boolean, $.nil),

    number: (_) => /\d+(\.\d+)?/,

    string: (_) => /"[^"]*"/,

    boolean: (_) => choice("true", "false"),

    nil: (_) => "nil",
  },
});
