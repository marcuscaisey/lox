(identifier) @variable

[
  (this_expression)
  (super_expression)
] @variable.builtin


(
  (identifier) @variable.builtin
  (#eq? @variable.builtin "argv")
)

(parameter_list
  (identifier) @variable.parameter)

(property_expression
  name: (identifier) @variable.member)

((identifier) @constant
 (#match? @constant "^[A-Z][A-Z_0-9]*$"))

(nil) @constant.builtin

(string) @string

(boolean) @boolean

(number) @number

((identifier) @type
  (#match? @type "^[A-Z][a-z]"))

(class_declaration
  name: (identifier) @type)

(function_declaration
  name: (identifier) @function)

(call_expression
  callee: (identifier) @function.call)

((identifier) @function.builtin (#any-of? @function.builtin "clock" "type" "error"))

(method_declaration
  name: (identifier) @function.method)

(method_signature
  "method" @variable
  class: (identifier) @type
  method: (identifier) @function.method)

(property_signature
  "property" @variable
  class: (identifier) @type
  name: (identifier) @variable.member)

(call_expression
  callee: (property_expression
    name: (identifier) @function.method.call))

(method_declaration
  name: (identifier) @constructor (#eq? @constructor "init"))

(method_signature
  method: (identifier) @constructor (#eq? @constructor "init"))

(call_expression
  callee: (identifier) @constructor (#match? @constructor "^[A-Z]"))

(call_expression
  callee: (property_expression
    name: (identifier) @constructor (#match? @constructor "^[A-Z]")))

[
  "!"
  "=="
  "!="
  "<"
  "<="
  ">"
  ">="
  "+"
  "-"
  "*"
  "/"
  "%"
  "="
] @operator

[
  "print"
  "var"
  "break"
  "continue"
  "return"
] @keyword

"fun" @keyword.function

"class" @keyword.type

(modifiers [
  "static"
  "get"
  "set"
] @keyword.modifier)

[
  "or"
  "and"
] @keyword.operator

[
  "while"
  "for"
] @keyword.repeat

[
  "if"
  "else"
] @keyword.conditional

[
  "?"
  ":"
] @keyword.conditional.ternary

[
  ","
  "."
  ";"
] @punctuation.delimiter

[
  "("
  ")"
  "["
  "]"
  "{"
  "}"
] @punctuation.bracket

(comment) @comment
