#include <tree_sitter/parser.h>

#if defined(__GNUC__) || defined(__clang__)
#pragma GCC diagnostic push
#pragma GCC diagnostic ignored "-Wmissing-field-initializers"
#endif

#define LANGUAGE_VERSION 14
#define STATE_COUNT 30
#define LARGE_STATE_COUNT 4
#define SYMBOL_COUNT 36
#define ALIAS_COUNT 0
#define TOKEN_COUNT 24
#define EXTERNAL_TOKEN_COUNT 0
#define FIELD_COUNT 6
#define MAX_ALIAS_SEQUENCE_LENGTH 5
#define PRODUCTION_ID_COUNT 5

enum {
  anon_sym_SEMI = 1,
  anon_sym_print = 2,
  anon_sym_COMMA = 3,
  anon_sym_EQ_EQ = 4,
  anon_sym_BANG_EQ = 5,
  anon_sym_LT = 6,
  anon_sym_LT_EQ = 7,
  anon_sym_GT = 8,
  anon_sym_GT_EQ = 9,
  anon_sym_PLUS = 10,
  anon_sym_DASH = 11,
  anon_sym_STAR = 12,
  anon_sym_SLASH = 13,
  anon_sym_QMARK = 14,
  anon_sym_COLON = 15,
  anon_sym_BANG = 16,
  anon_sym_LPAREN = 17,
  anon_sym_RPAREN = 18,
  sym_number = 19,
  sym_string = 20,
  anon_sym_true = 21,
  anon_sym_false = 22,
  sym_nil = 23,
  sym_program = 24,
  sym__statement = 25,
  sym_expression_statement = 26,
  sym_print_statement = 27,
  sym__expression = 28,
  sym_binary_expression = 29,
  sym_ternary_expression = 30,
  sym_unary_expression = 31,
  sym_group_expression = 32,
  sym__literal_expression = 33,
  sym_boolean = 34,
  aux_sym_program_repeat1 = 35,
};

static const char * const ts_symbol_names[] = {
  [ts_builtin_sym_end] = "end",
  [anon_sym_SEMI] = ";",
  [anon_sym_print] = "print",
  [anon_sym_COMMA] = ",",
  [anon_sym_EQ_EQ] = "==",
  [anon_sym_BANG_EQ] = "!=",
  [anon_sym_LT] = "<",
  [anon_sym_LT_EQ] = "<=",
  [anon_sym_GT] = ">",
  [anon_sym_GT_EQ] = ">=",
  [anon_sym_PLUS] = "+",
  [anon_sym_DASH] = "-",
  [anon_sym_STAR] = "*",
  [anon_sym_SLASH] = "/",
  [anon_sym_QMARK] = "\?",
  [anon_sym_COLON] = ":",
  [anon_sym_BANG] = "!",
  [anon_sym_LPAREN] = "(",
  [anon_sym_RPAREN] = ")",
  [sym_number] = "number",
  [sym_string] = "string",
  [anon_sym_true] = "true",
  [anon_sym_false] = "false",
  [sym_nil] = "nil",
  [sym_program] = "program",
  [sym__statement] = "_statement",
  [sym_expression_statement] = "expression_statement",
  [sym_print_statement] = "print_statement",
  [sym__expression] = "_expression",
  [sym_binary_expression] = "binary_expression",
  [sym_ternary_expression] = "ternary_expression",
  [sym_unary_expression] = "unary_expression",
  [sym_group_expression] = "group_expression",
  [sym__literal_expression] = "_literal_expression",
  [sym_boolean] = "boolean",
  [aux_sym_program_repeat1] = "program_repeat1",
};

static const TSSymbol ts_symbol_map[] = {
  [ts_builtin_sym_end] = ts_builtin_sym_end,
  [anon_sym_SEMI] = anon_sym_SEMI,
  [anon_sym_print] = anon_sym_print,
  [anon_sym_COMMA] = anon_sym_COMMA,
  [anon_sym_EQ_EQ] = anon_sym_EQ_EQ,
  [anon_sym_BANG_EQ] = anon_sym_BANG_EQ,
  [anon_sym_LT] = anon_sym_LT,
  [anon_sym_LT_EQ] = anon_sym_LT_EQ,
  [anon_sym_GT] = anon_sym_GT,
  [anon_sym_GT_EQ] = anon_sym_GT_EQ,
  [anon_sym_PLUS] = anon_sym_PLUS,
  [anon_sym_DASH] = anon_sym_DASH,
  [anon_sym_STAR] = anon_sym_STAR,
  [anon_sym_SLASH] = anon_sym_SLASH,
  [anon_sym_QMARK] = anon_sym_QMARK,
  [anon_sym_COLON] = anon_sym_COLON,
  [anon_sym_BANG] = anon_sym_BANG,
  [anon_sym_LPAREN] = anon_sym_LPAREN,
  [anon_sym_RPAREN] = anon_sym_RPAREN,
  [sym_number] = sym_number,
  [sym_string] = sym_string,
  [anon_sym_true] = anon_sym_true,
  [anon_sym_false] = anon_sym_false,
  [sym_nil] = sym_nil,
  [sym_program] = sym_program,
  [sym__statement] = sym__statement,
  [sym_expression_statement] = sym_expression_statement,
  [sym_print_statement] = sym_print_statement,
  [sym__expression] = sym__expression,
  [sym_binary_expression] = sym_binary_expression,
  [sym_ternary_expression] = sym_ternary_expression,
  [sym_unary_expression] = sym_unary_expression,
  [sym_group_expression] = sym_group_expression,
  [sym__literal_expression] = sym__literal_expression,
  [sym_boolean] = sym_boolean,
  [aux_sym_program_repeat1] = aux_sym_program_repeat1,
};

static const TSSymbolMetadata ts_symbol_metadata[] = {
  [ts_builtin_sym_end] = {
    .visible = false,
    .named = true,
  },
  [anon_sym_SEMI] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_print] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_COMMA] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_EQ_EQ] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_BANG_EQ] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_LT] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_LT_EQ] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_GT] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_GT_EQ] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_PLUS] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_DASH] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_STAR] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_SLASH] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_QMARK] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_COLON] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_BANG] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_LPAREN] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_RPAREN] = {
    .visible = true,
    .named = false,
  },
  [sym_number] = {
    .visible = true,
    .named = true,
  },
  [sym_string] = {
    .visible = true,
    .named = true,
  },
  [anon_sym_true] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_false] = {
    .visible = true,
    .named = false,
  },
  [sym_nil] = {
    .visible = true,
    .named = true,
  },
  [sym_program] = {
    .visible = true,
    .named = true,
  },
  [sym__statement] = {
    .visible = false,
    .named = true,
  },
  [sym_expression_statement] = {
    .visible = true,
    .named = true,
  },
  [sym_print_statement] = {
    .visible = true,
    .named = true,
  },
  [sym__expression] = {
    .visible = false,
    .named = true,
  },
  [sym_binary_expression] = {
    .visible = true,
    .named = true,
  },
  [sym_ternary_expression] = {
    .visible = true,
    .named = true,
  },
  [sym_unary_expression] = {
    .visible = true,
    .named = true,
  },
  [sym_group_expression] = {
    .visible = true,
    .named = true,
  },
  [sym__literal_expression] = {
    .visible = false,
    .named = true,
  },
  [sym_boolean] = {
    .visible = true,
    .named = true,
  },
  [aux_sym_program_repeat1] = {
    .visible = false,
    .named = false,
  },
};

enum {
  field_condition = 1,
  field_else = 2,
  field_expression = 3,
  field_left = 4,
  field_right = 5,
  field_then = 6,
};

static const char * const ts_field_names[] = {
  [0] = NULL,
  [field_condition] = "condition",
  [field_else] = "else",
  [field_expression] = "expression",
  [field_left] = "left",
  [field_right] = "right",
  [field_then] = "then",
};

static const TSFieldMapSlice ts_field_map_slices[PRODUCTION_ID_COUNT] = {
  [1] = {.index = 0, .length = 1},
  [2] = {.index = 1, .length = 1},
  [3] = {.index = 2, .length = 2},
  [4] = {.index = 4, .length = 3},
};

static const TSFieldMapEntry ts_field_map_entries[] = {
  [0] =
    {field_right, 1},
  [1] =
    {field_expression, 1},
  [2] =
    {field_left, 0},
    {field_right, 2},
  [4] =
    {field_condition, 0},
    {field_else, 4},
    {field_then, 2},
};

static const TSSymbol ts_alias_sequences[PRODUCTION_ID_COUNT][MAX_ALIAS_SEQUENCE_LENGTH] = {
  [0] = {0},
};

static const uint16_t ts_non_terminal_alias_map[] = {
  0,
};

static const TSStateId ts_primary_state_ids[STATE_COUNT] = {
  [0] = 0,
  [1] = 1,
  [2] = 2,
  [3] = 3,
  [4] = 4,
  [5] = 5,
  [6] = 6,
  [7] = 7,
  [8] = 8,
  [9] = 9,
  [10] = 10,
  [11] = 11,
  [12] = 12,
  [13] = 13,
  [14] = 14,
  [15] = 15,
  [16] = 16,
  [17] = 17,
  [18] = 18,
  [19] = 19,
  [20] = 20,
  [21] = 21,
  [22] = 22,
  [23] = 23,
  [24] = 24,
  [25] = 25,
  [26] = 26,
  [27] = 27,
  [28] = 28,
  [29] = 29,
};

static bool ts_lex(TSLexer *lexer, TSStateId state) {
  START_LEXER();
  eof = lexer->eof(lexer);
  switch (state) {
    case 0:
      if (eof) ADVANCE(20);
      if (lookahead == '!') ADVANCE(37);
      if (lookahead == '"') ADVANCE(2);
      if (lookahead == '(') ADVANCE(38);
      if (lookahead == ')') ADVANCE(39);
      if (lookahead == '*') ADVANCE(32);
      if (lookahead == '+') ADVANCE(30);
      if (lookahead == ',') ADVANCE(23);
      if (lookahead == '-') ADVANCE(31);
      if (lookahead == '/') ADVANCE(33);
      if (lookahead == ':') ADVANCE(35);
      if (lookahead == ';') ADVANCE(21);
      if (lookahead == '<') ADVANCE(26);
      if (lookahead == '=') ADVANCE(4);
      if (lookahead == '>') ADVANCE(28);
      if (lookahead == '?') ADVANCE(34);
      if (lookahead == 'f') ADVANCE(5);
      if (lookahead == 'n') ADVANCE(9);
      if (lookahead == 'p') ADVANCE(14);
      if (lookahead == 't') ADVANCE(13);
      if (lookahead == '\t' ||
          lookahead == '\n' ||
          lookahead == '\r' ||
          lookahead == ' ') SKIP(0)
      if (('0' <= lookahead && lookahead <= '9')) ADVANCE(40);
      END_STATE();
    case 1:
      if (lookahead == '!') ADVANCE(3);
      if (lookahead == ')') ADVANCE(39);
      if (lookahead == '*') ADVANCE(32);
      if (lookahead == '+') ADVANCE(30);
      if (lookahead == ',') ADVANCE(23);
      if (lookahead == '-') ADVANCE(31);
      if (lookahead == '/') ADVANCE(33);
      if (lookahead == ':') ADVANCE(35);
      if (lookahead == ';') ADVANCE(21);
      if (lookahead == '<') ADVANCE(26);
      if (lookahead == '=') ADVANCE(4);
      if (lookahead == '>') ADVANCE(28);
      if (lookahead == '?') ADVANCE(34);
      if (lookahead == '\t' ||
          lookahead == '\n' ||
          lookahead == '\r' ||
          lookahead == ' ') SKIP(1)
      END_STATE();
    case 2:
      if (lookahead == '"') ADVANCE(42);
      if (lookahead != 0) ADVANCE(2);
      END_STATE();
    case 3:
      if (lookahead == '=') ADVANCE(25);
      END_STATE();
    case 4:
      if (lookahead == '=') ADVANCE(24);
      END_STATE();
    case 5:
      if (lookahead == 'a') ADVANCE(10);
      END_STATE();
    case 6:
      if (lookahead == 'e') ADVANCE(43);
      END_STATE();
    case 7:
      if (lookahead == 'e') ADVANCE(44);
      END_STATE();
    case 8:
      if (lookahead == 'i') ADVANCE(12);
      END_STATE();
    case 9:
      if (lookahead == 'i') ADVANCE(11);
      END_STATE();
    case 10:
      if (lookahead == 'l') ADVANCE(15);
      END_STATE();
    case 11:
      if (lookahead == 'l') ADVANCE(45);
      END_STATE();
    case 12:
      if (lookahead == 'n') ADVANCE(16);
      END_STATE();
    case 13:
      if (lookahead == 'r') ADVANCE(17);
      END_STATE();
    case 14:
      if (lookahead == 'r') ADVANCE(8);
      END_STATE();
    case 15:
      if (lookahead == 's') ADVANCE(7);
      END_STATE();
    case 16:
      if (lookahead == 't') ADVANCE(22);
      END_STATE();
    case 17:
      if (lookahead == 'u') ADVANCE(6);
      END_STATE();
    case 18:
      if (('0' <= lookahead && lookahead <= '9')) ADVANCE(41);
      END_STATE();
    case 19:
      if (eof) ADVANCE(20);
      if (lookahead == '!') ADVANCE(36);
      if (lookahead == '"') ADVANCE(2);
      if (lookahead == '(') ADVANCE(38);
      if (lookahead == '-') ADVANCE(31);
      if (lookahead == 'f') ADVANCE(5);
      if (lookahead == 'n') ADVANCE(9);
      if (lookahead == 'p') ADVANCE(14);
      if (lookahead == 't') ADVANCE(13);
      if (lookahead == '\t' ||
          lookahead == '\n' ||
          lookahead == '\r' ||
          lookahead == ' ') SKIP(19)
      if (('0' <= lookahead && lookahead <= '9')) ADVANCE(40);
      END_STATE();
    case 20:
      ACCEPT_TOKEN(ts_builtin_sym_end);
      END_STATE();
    case 21:
      ACCEPT_TOKEN(anon_sym_SEMI);
      END_STATE();
    case 22:
      ACCEPT_TOKEN(anon_sym_print);
      END_STATE();
    case 23:
      ACCEPT_TOKEN(anon_sym_COMMA);
      END_STATE();
    case 24:
      ACCEPT_TOKEN(anon_sym_EQ_EQ);
      END_STATE();
    case 25:
      ACCEPT_TOKEN(anon_sym_BANG_EQ);
      END_STATE();
    case 26:
      ACCEPT_TOKEN(anon_sym_LT);
      if (lookahead == '=') ADVANCE(27);
      END_STATE();
    case 27:
      ACCEPT_TOKEN(anon_sym_LT_EQ);
      END_STATE();
    case 28:
      ACCEPT_TOKEN(anon_sym_GT);
      if (lookahead == '=') ADVANCE(29);
      END_STATE();
    case 29:
      ACCEPT_TOKEN(anon_sym_GT_EQ);
      END_STATE();
    case 30:
      ACCEPT_TOKEN(anon_sym_PLUS);
      END_STATE();
    case 31:
      ACCEPT_TOKEN(anon_sym_DASH);
      END_STATE();
    case 32:
      ACCEPT_TOKEN(anon_sym_STAR);
      END_STATE();
    case 33:
      ACCEPT_TOKEN(anon_sym_SLASH);
      END_STATE();
    case 34:
      ACCEPT_TOKEN(anon_sym_QMARK);
      END_STATE();
    case 35:
      ACCEPT_TOKEN(anon_sym_COLON);
      END_STATE();
    case 36:
      ACCEPT_TOKEN(anon_sym_BANG);
      END_STATE();
    case 37:
      ACCEPT_TOKEN(anon_sym_BANG);
      if (lookahead == '=') ADVANCE(25);
      END_STATE();
    case 38:
      ACCEPT_TOKEN(anon_sym_LPAREN);
      END_STATE();
    case 39:
      ACCEPT_TOKEN(anon_sym_RPAREN);
      END_STATE();
    case 40:
      ACCEPT_TOKEN(sym_number);
      if (lookahead == '.') ADVANCE(18);
      if (('0' <= lookahead && lookahead <= '9')) ADVANCE(40);
      END_STATE();
    case 41:
      ACCEPT_TOKEN(sym_number);
      if (('0' <= lookahead && lookahead <= '9')) ADVANCE(41);
      END_STATE();
    case 42:
      ACCEPT_TOKEN(sym_string);
      END_STATE();
    case 43:
      ACCEPT_TOKEN(anon_sym_true);
      END_STATE();
    case 44:
      ACCEPT_TOKEN(anon_sym_false);
      END_STATE();
    case 45:
      ACCEPT_TOKEN(sym_nil);
      END_STATE();
    default:
      return false;
  }
}

static const TSLexMode ts_lex_modes[STATE_COUNT] = {
  [0] = {.lex_state = 0},
  [1] = {.lex_state = 19},
  [2] = {.lex_state = 19},
  [3] = {.lex_state = 19},
  [4] = {.lex_state = 1},
  [5] = {.lex_state = 1},
  [6] = {.lex_state = 1},
  [7] = {.lex_state = 1},
  [8] = {.lex_state = 19},
  [9] = {.lex_state = 1},
  [10] = {.lex_state = 1},
  [11] = {.lex_state = 1},
  [12] = {.lex_state = 1},
  [13] = {.lex_state = 19},
  [14] = {.lex_state = 19},
  [15] = {.lex_state = 19},
  [16] = {.lex_state = 19},
  [17] = {.lex_state = 19},
  [18] = {.lex_state = 19},
  [19] = {.lex_state = 19},
  [20] = {.lex_state = 19},
  [21] = {.lex_state = 19},
  [22] = {.lex_state = 1},
  [23] = {.lex_state = 1},
  [24] = {.lex_state = 1},
  [25] = {.lex_state = 1},
  [26] = {.lex_state = 1},
  [27] = {.lex_state = 19},
  [28] = {.lex_state = 19},
  [29] = {.lex_state = 0},
};

static const uint16_t ts_parse_table[LARGE_STATE_COUNT][SYMBOL_COUNT] = {
  [0] = {
    [ts_builtin_sym_end] = ACTIONS(1),
    [anon_sym_SEMI] = ACTIONS(1),
    [anon_sym_print] = ACTIONS(1),
    [anon_sym_COMMA] = ACTIONS(1),
    [anon_sym_EQ_EQ] = ACTIONS(1),
    [anon_sym_BANG_EQ] = ACTIONS(1),
    [anon_sym_LT] = ACTIONS(1),
    [anon_sym_LT_EQ] = ACTIONS(1),
    [anon_sym_GT] = ACTIONS(1),
    [anon_sym_GT_EQ] = ACTIONS(1),
    [anon_sym_PLUS] = ACTIONS(1),
    [anon_sym_DASH] = ACTIONS(1),
    [anon_sym_STAR] = ACTIONS(1),
    [anon_sym_SLASH] = ACTIONS(1),
    [anon_sym_QMARK] = ACTIONS(1),
    [anon_sym_COLON] = ACTIONS(1),
    [anon_sym_BANG] = ACTIONS(1),
    [anon_sym_LPAREN] = ACTIONS(1),
    [anon_sym_RPAREN] = ACTIONS(1),
    [sym_number] = ACTIONS(1),
    [sym_string] = ACTIONS(1),
    [anon_sym_true] = ACTIONS(1),
    [anon_sym_false] = ACTIONS(1),
    [sym_nil] = ACTIONS(1),
  },
  [1] = {
    [sym_program] = STATE(29),
    [sym__statement] = STATE(2),
    [sym_expression_statement] = STATE(2),
    [sym_print_statement] = STATE(2),
    [sym__expression] = STATE(26),
    [sym_binary_expression] = STATE(26),
    [sym_ternary_expression] = STATE(26),
    [sym_unary_expression] = STATE(26),
    [sym_group_expression] = STATE(26),
    [sym__literal_expression] = STATE(26),
    [sym_boolean] = STATE(26),
    [aux_sym_program_repeat1] = STATE(2),
    [ts_builtin_sym_end] = ACTIONS(3),
    [anon_sym_print] = ACTIONS(5),
    [anon_sym_DASH] = ACTIONS(7),
    [anon_sym_BANG] = ACTIONS(7),
    [anon_sym_LPAREN] = ACTIONS(9),
    [sym_number] = ACTIONS(11),
    [sym_string] = ACTIONS(11),
    [anon_sym_true] = ACTIONS(13),
    [anon_sym_false] = ACTIONS(13),
    [sym_nil] = ACTIONS(11),
  },
  [2] = {
    [sym__statement] = STATE(3),
    [sym_expression_statement] = STATE(3),
    [sym_print_statement] = STATE(3),
    [sym__expression] = STATE(26),
    [sym_binary_expression] = STATE(26),
    [sym_ternary_expression] = STATE(26),
    [sym_unary_expression] = STATE(26),
    [sym_group_expression] = STATE(26),
    [sym__literal_expression] = STATE(26),
    [sym_boolean] = STATE(26),
    [aux_sym_program_repeat1] = STATE(3),
    [ts_builtin_sym_end] = ACTIONS(15),
    [anon_sym_print] = ACTIONS(5),
    [anon_sym_DASH] = ACTIONS(7),
    [anon_sym_BANG] = ACTIONS(7),
    [anon_sym_LPAREN] = ACTIONS(9),
    [sym_number] = ACTIONS(11),
    [sym_string] = ACTIONS(11),
    [anon_sym_true] = ACTIONS(13),
    [anon_sym_false] = ACTIONS(13),
    [sym_nil] = ACTIONS(11),
  },
  [3] = {
    [sym__statement] = STATE(3),
    [sym_expression_statement] = STATE(3),
    [sym_print_statement] = STATE(3),
    [sym__expression] = STATE(26),
    [sym_binary_expression] = STATE(26),
    [sym_ternary_expression] = STATE(26),
    [sym_unary_expression] = STATE(26),
    [sym_group_expression] = STATE(26),
    [sym__literal_expression] = STATE(26),
    [sym_boolean] = STATE(26),
    [aux_sym_program_repeat1] = STATE(3),
    [ts_builtin_sym_end] = ACTIONS(17),
    [anon_sym_print] = ACTIONS(19),
    [anon_sym_DASH] = ACTIONS(22),
    [anon_sym_BANG] = ACTIONS(22),
    [anon_sym_LPAREN] = ACTIONS(25),
    [sym_number] = ACTIONS(28),
    [sym_string] = ACTIONS(28),
    [anon_sym_true] = ACTIONS(31),
    [anon_sym_false] = ACTIONS(31),
    [sym_nil] = ACTIONS(28),
  },
};

static const uint16_t ts_small_parse_table[] = {
  [0] = 2,
    ACTIONS(36), 2,
      anon_sym_LT,
      anon_sym_GT,
    ACTIONS(34), 13,
      anon_sym_SEMI,
      anon_sym_COMMA,
      anon_sym_EQ_EQ,
      anon_sym_BANG_EQ,
      anon_sym_LT_EQ,
      anon_sym_GT_EQ,
      anon_sym_PLUS,
      anon_sym_DASH,
      anon_sym_STAR,
      anon_sym_SLASH,
      anon_sym_QMARK,
      anon_sym_COLON,
      anon_sym_RPAREN,
  [20] = 5,
    ACTIONS(40), 2,
      anon_sym_LT,
      anon_sym_GT,
    ACTIONS(42), 2,
      anon_sym_LT_EQ,
      anon_sym_GT_EQ,
    ACTIONS(44), 2,
      anon_sym_PLUS,
      anon_sym_DASH,
    ACTIONS(46), 2,
      anon_sym_STAR,
      anon_sym_SLASH,
    ACTIONS(38), 7,
      anon_sym_SEMI,
      anon_sym_COMMA,
      anon_sym_EQ_EQ,
      anon_sym_BANG_EQ,
      anon_sym_QMARK,
      anon_sym_COLON,
      anon_sym_RPAREN,
  [46] = 2,
    ACTIONS(50), 2,
      anon_sym_LT,
      anon_sym_GT,
    ACTIONS(48), 13,
      anon_sym_SEMI,
      anon_sym_COMMA,
      anon_sym_EQ_EQ,
      anon_sym_BANG_EQ,
      anon_sym_LT_EQ,
      anon_sym_GT_EQ,
      anon_sym_PLUS,
      anon_sym_DASH,
      anon_sym_STAR,
      anon_sym_SLASH,
      anon_sym_QMARK,
      anon_sym_COLON,
      anon_sym_RPAREN,
  [66] = 7,
    ACTIONS(56), 1,
      anon_sym_QMARK,
    ACTIONS(40), 2,
      anon_sym_LT,
      anon_sym_GT,
    ACTIONS(42), 2,
      anon_sym_LT_EQ,
      anon_sym_GT_EQ,
    ACTIONS(44), 2,
      anon_sym_PLUS,
      anon_sym_DASH,
    ACTIONS(46), 2,
      anon_sym_STAR,
      anon_sym_SLASH,
    ACTIONS(54), 2,
      anon_sym_EQ_EQ,
      anon_sym_BANG_EQ,
    ACTIONS(52), 4,
      anon_sym_SEMI,
      anon_sym_COMMA,
      anon_sym_COLON,
      anon_sym_RPAREN,
  [96] = 5,
    ACTIONS(9), 1,
      anon_sym_LPAREN,
    ACTIONS(7), 2,
      anon_sym_DASH,
      anon_sym_BANG,
    ACTIONS(13), 2,
      anon_sym_true,
      anon_sym_false,
    ACTIONS(58), 3,
      sym_number,
      sym_string,
      sym_nil,
    STATE(7), 7,
      sym__expression,
      sym_binary_expression,
      sym_ternary_expression,
      sym_unary_expression,
      sym_group_expression,
      sym__literal_expression,
      sym_boolean,
  [122] = 2,
    ACTIONS(60), 2,
      anon_sym_LT,
      anon_sym_GT,
    ACTIONS(38), 13,
      anon_sym_SEMI,
      anon_sym_COMMA,
      anon_sym_EQ_EQ,
      anon_sym_BANG_EQ,
      anon_sym_LT_EQ,
      anon_sym_GT_EQ,
      anon_sym_PLUS,
      anon_sym_DASH,
      anon_sym_STAR,
      anon_sym_SLASH,
      anon_sym_QMARK,
      anon_sym_COLON,
      anon_sym_RPAREN,
  [142] = 2,
    ACTIONS(64), 2,
      anon_sym_LT,
      anon_sym_GT,
    ACTIONS(62), 13,
      anon_sym_SEMI,
      anon_sym_COMMA,
      anon_sym_EQ_EQ,
      anon_sym_BANG_EQ,
      anon_sym_LT_EQ,
      anon_sym_GT_EQ,
      anon_sym_PLUS,
      anon_sym_DASH,
      anon_sym_STAR,
      anon_sym_SLASH,
      anon_sym_QMARK,
      anon_sym_COLON,
      anon_sym_RPAREN,
  [162] = 3,
    ACTIONS(46), 2,
      anon_sym_STAR,
      anon_sym_SLASH,
    ACTIONS(60), 2,
      anon_sym_LT,
      anon_sym_GT,
    ACTIONS(38), 11,
      anon_sym_SEMI,
      anon_sym_COMMA,
      anon_sym_EQ_EQ,
      anon_sym_BANG_EQ,
      anon_sym_LT_EQ,
      anon_sym_GT_EQ,
      anon_sym_PLUS,
      anon_sym_DASH,
      anon_sym_QMARK,
      anon_sym_COLON,
      anon_sym_RPAREN,
  [184] = 4,
    ACTIONS(44), 2,
      anon_sym_PLUS,
      anon_sym_DASH,
    ACTIONS(46), 2,
      anon_sym_STAR,
      anon_sym_SLASH,
    ACTIONS(60), 2,
      anon_sym_LT,
      anon_sym_GT,
    ACTIONS(38), 9,
      anon_sym_SEMI,
      anon_sym_COMMA,
      anon_sym_EQ_EQ,
      anon_sym_BANG_EQ,
      anon_sym_LT_EQ,
      anon_sym_GT_EQ,
      anon_sym_QMARK,
      anon_sym_COLON,
      anon_sym_RPAREN,
  [208] = 5,
    ACTIONS(9), 1,
      anon_sym_LPAREN,
    ACTIONS(7), 2,
      anon_sym_DASH,
      anon_sym_BANG,
    ACTIONS(13), 2,
      anon_sym_true,
      anon_sym_false,
    ACTIONS(66), 3,
      sym_number,
      sym_string,
      sym_nil,
    STATE(22), 7,
      sym__expression,
      sym_binary_expression,
      sym_ternary_expression,
      sym_unary_expression,
      sym_group_expression,
      sym__literal_expression,
      sym_boolean,
  [234] = 5,
    ACTIONS(9), 1,
      anon_sym_LPAREN,
    ACTIONS(7), 2,
      anon_sym_DASH,
      anon_sym_BANG,
    ACTIONS(13), 2,
      anon_sym_true,
      anon_sym_false,
    ACTIONS(68), 3,
      sym_number,
      sym_string,
      sym_nil,
    STATE(5), 7,
      sym__expression,
      sym_binary_expression,
      sym_ternary_expression,
      sym_unary_expression,
      sym_group_expression,
      sym__literal_expression,
      sym_boolean,
  [260] = 5,
    ACTIONS(9), 1,
      anon_sym_LPAREN,
    ACTIONS(7), 2,
      anon_sym_DASH,
      anon_sym_BANG,
    ACTIONS(13), 2,
      anon_sym_true,
      anon_sym_false,
    ACTIONS(70), 3,
      sym_number,
      sym_string,
      sym_nil,
    STATE(12), 7,
      sym__expression,
      sym_binary_expression,
      sym_ternary_expression,
      sym_unary_expression,
      sym_group_expression,
      sym__literal_expression,
      sym_boolean,
  [286] = 5,
    ACTIONS(9), 1,
      anon_sym_LPAREN,
    ACTIONS(7), 2,
      anon_sym_DASH,
      anon_sym_BANG,
    ACTIONS(13), 2,
      anon_sym_true,
      anon_sym_false,
    ACTIONS(72), 3,
      sym_number,
      sym_string,
      sym_nil,
    STATE(11), 7,
      sym__expression,
      sym_binary_expression,
      sym_ternary_expression,
      sym_unary_expression,
      sym_group_expression,
      sym__literal_expression,
      sym_boolean,
  [312] = 5,
    ACTIONS(9), 1,
      anon_sym_LPAREN,
    ACTIONS(7), 2,
      anon_sym_DASH,
      anon_sym_BANG,
    ACTIONS(13), 2,
      anon_sym_true,
      anon_sym_false,
    ACTIONS(74), 3,
      sym_number,
      sym_string,
      sym_nil,
    STATE(9), 7,
      sym__expression,
      sym_binary_expression,
      sym_ternary_expression,
      sym_unary_expression,
      sym_group_expression,
      sym__literal_expression,
      sym_boolean,
  [338] = 5,
    ACTIONS(9), 1,
      anon_sym_LPAREN,
    ACTIONS(7), 2,
      anon_sym_DASH,
      anon_sym_BANG,
    ACTIONS(13), 2,
      anon_sym_true,
      anon_sym_false,
    ACTIONS(76), 3,
      sym_number,
      sym_string,
      sym_nil,
    STATE(25), 7,
      sym__expression,
      sym_binary_expression,
      sym_ternary_expression,
      sym_unary_expression,
      sym_group_expression,
      sym__literal_expression,
      sym_boolean,
  [364] = 5,
    ACTIONS(9), 1,
      anon_sym_LPAREN,
    ACTIONS(7), 2,
      anon_sym_DASH,
      anon_sym_BANG,
    ACTIONS(13), 2,
      anon_sym_true,
      anon_sym_false,
    ACTIONS(78), 3,
      sym_number,
      sym_string,
      sym_nil,
    STATE(24), 7,
      sym__expression,
      sym_binary_expression,
      sym_ternary_expression,
      sym_unary_expression,
      sym_group_expression,
      sym__literal_expression,
      sym_boolean,
  [390] = 5,
    ACTIONS(9), 1,
      anon_sym_LPAREN,
    ACTIONS(7), 2,
      anon_sym_DASH,
      anon_sym_BANG,
    ACTIONS(13), 2,
      anon_sym_true,
      anon_sym_false,
    ACTIONS(80), 3,
      sym_number,
      sym_string,
      sym_nil,
    STATE(23), 7,
      sym__expression,
      sym_binary_expression,
      sym_ternary_expression,
      sym_unary_expression,
      sym_group_expression,
      sym__literal_expression,
      sym_boolean,
  [416] = 5,
    ACTIONS(9), 1,
      anon_sym_LPAREN,
    ACTIONS(7), 2,
      anon_sym_DASH,
      anon_sym_BANG,
    ACTIONS(13), 2,
      anon_sym_true,
      anon_sym_false,
    ACTIONS(82), 3,
      sym_number,
      sym_string,
      sym_nil,
    STATE(10), 7,
      sym__expression,
      sym_binary_expression,
      sym_ternary_expression,
      sym_unary_expression,
      sym_group_expression,
      sym__literal_expression,
      sym_boolean,
  [442] = 7,
    ACTIONS(56), 1,
      anon_sym_QMARK,
    ACTIONS(40), 2,
      anon_sym_LT,
      anon_sym_GT,
    ACTIONS(42), 2,
      anon_sym_LT_EQ,
      anon_sym_GT_EQ,
    ACTIONS(44), 2,
      anon_sym_PLUS,
      anon_sym_DASH,
    ACTIONS(46), 2,
      anon_sym_STAR,
      anon_sym_SLASH,
    ACTIONS(54), 2,
      anon_sym_EQ_EQ,
      anon_sym_BANG_EQ,
    ACTIONS(38), 4,
      anon_sym_SEMI,
      anon_sym_COMMA,
      anon_sym_COLON,
      anon_sym_RPAREN,
  [472] = 8,
    ACTIONS(56), 1,
      anon_sym_QMARK,
    ACTIONS(84), 1,
      anon_sym_COMMA,
    ACTIONS(86), 1,
      anon_sym_RPAREN,
    ACTIONS(40), 2,
      anon_sym_LT,
      anon_sym_GT,
    ACTIONS(42), 2,
      anon_sym_LT_EQ,
      anon_sym_GT_EQ,
    ACTIONS(44), 2,
      anon_sym_PLUS,
      anon_sym_DASH,
    ACTIONS(46), 2,
      anon_sym_STAR,
      anon_sym_SLASH,
    ACTIONS(54), 2,
      anon_sym_EQ_EQ,
      anon_sym_BANG_EQ,
  [502] = 8,
    ACTIONS(56), 1,
      anon_sym_QMARK,
    ACTIONS(84), 1,
      anon_sym_COMMA,
    ACTIONS(88), 1,
      anon_sym_SEMI,
    ACTIONS(40), 2,
      anon_sym_LT,
      anon_sym_GT,
    ACTIONS(42), 2,
      anon_sym_LT_EQ,
      anon_sym_GT_EQ,
    ACTIONS(44), 2,
      anon_sym_PLUS,
      anon_sym_DASH,
    ACTIONS(46), 2,
      anon_sym_STAR,
      anon_sym_SLASH,
    ACTIONS(54), 2,
      anon_sym_EQ_EQ,
      anon_sym_BANG_EQ,
  [532] = 8,
    ACTIONS(56), 1,
      anon_sym_QMARK,
    ACTIONS(84), 1,
      anon_sym_COMMA,
    ACTIONS(90), 1,
      anon_sym_COLON,
    ACTIONS(40), 2,
      anon_sym_LT,
      anon_sym_GT,
    ACTIONS(42), 2,
      anon_sym_LT_EQ,
      anon_sym_GT_EQ,
    ACTIONS(44), 2,
      anon_sym_PLUS,
      anon_sym_DASH,
    ACTIONS(46), 2,
      anon_sym_STAR,
      anon_sym_SLASH,
    ACTIONS(54), 2,
      anon_sym_EQ_EQ,
      anon_sym_BANG_EQ,
  [562] = 8,
    ACTIONS(56), 1,
      anon_sym_QMARK,
    ACTIONS(84), 1,
      anon_sym_COMMA,
    ACTIONS(92), 1,
      anon_sym_SEMI,
    ACTIONS(40), 2,
      anon_sym_LT,
      anon_sym_GT,
    ACTIONS(42), 2,
      anon_sym_LT_EQ,
      anon_sym_GT_EQ,
    ACTIONS(44), 2,
      anon_sym_PLUS,
      anon_sym_DASH,
    ACTIONS(46), 2,
      anon_sym_STAR,
      anon_sym_SLASH,
    ACTIONS(54), 2,
      anon_sym_EQ_EQ,
      anon_sym_BANG_EQ,
  [592] = 1,
    ACTIONS(94), 10,
      ts_builtin_sym_end,
      anon_sym_print,
      anon_sym_DASH,
      anon_sym_BANG,
      anon_sym_LPAREN,
      sym_number,
      sym_string,
      anon_sym_true,
      anon_sym_false,
      sym_nil,
  [605] = 1,
    ACTIONS(96), 10,
      ts_builtin_sym_end,
      anon_sym_print,
      anon_sym_DASH,
      anon_sym_BANG,
      anon_sym_LPAREN,
      sym_number,
      sym_string,
      anon_sym_true,
      anon_sym_false,
      sym_nil,
  [618] = 1,
    ACTIONS(98), 1,
      ts_builtin_sym_end,
};

static const uint32_t ts_small_parse_table_map[] = {
  [SMALL_STATE(4)] = 0,
  [SMALL_STATE(5)] = 20,
  [SMALL_STATE(6)] = 46,
  [SMALL_STATE(7)] = 66,
  [SMALL_STATE(8)] = 96,
  [SMALL_STATE(9)] = 122,
  [SMALL_STATE(10)] = 142,
  [SMALL_STATE(11)] = 162,
  [SMALL_STATE(12)] = 184,
  [SMALL_STATE(13)] = 208,
  [SMALL_STATE(14)] = 234,
  [SMALL_STATE(15)] = 260,
  [SMALL_STATE(16)] = 286,
  [SMALL_STATE(17)] = 312,
  [SMALL_STATE(18)] = 338,
  [SMALL_STATE(19)] = 364,
  [SMALL_STATE(20)] = 390,
  [SMALL_STATE(21)] = 416,
  [SMALL_STATE(22)] = 442,
  [SMALL_STATE(23)] = 472,
  [SMALL_STATE(24)] = 502,
  [SMALL_STATE(25)] = 532,
  [SMALL_STATE(26)] = 562,
  [SMALL_STATE(27)] = 592,
  [SMALL_STATE(28)] = 605,
  [SMALL_STATE(29)] = 618,
};

static const TSParseActionEntry ts_parse_actions[] = {
  [0] = {.entry = {.count = 0, .reusable = false}},
  [1] = {.entry = {.count = 1, .reusable = false}}, RECOVER(),
  [3] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_program, 0),
  [5] = {.entry = {.count = 1, .reusable = true}}, SHIFT(19),
  [7] = {.entry = {.count = 1, .reusable = true}}, SHIFT(21),
  [9] = {.entry = {.count = 1, .reusable = true}}, SHIFT(20),
  [11] = {.entry = {.count = 1, .reusable = true}}, SHIFT(26),
  [13] = {.entry = {.count = 1, .reusable = true}}, SHIFT(6),
  [15] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_program, 1),
  [17] = {.entry = {.count = 1, .reusable = true}}, REDUCE(aux_sym_program_repeat1, 2),
  [19] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_program_repeat1, 2), SHIFT_REPEAT(19),
  [22] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_program_repeat1, 2), SHIFT_REPEAT(21),
  [25] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_program_repeat1, 2), SHIFT_REPEAT(20),
  [28] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_program_repeat1, 2), SHIFT_REPEAT(26),
  [31] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_program_repeat1, 2), SHIFT_REPEAT(6),
  [34] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_group_expression, 3, .production_id = 2),
  [36] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_group_expression, 3, .production_id = 2),
  [38] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_binary_expression, 3, .production_id = 3),
  [40] = {.entry = {.count = 1, .reusable = false}}, SHIFT(15),
  [42] = {.entry = {.count = 1, .reusable = true}}, SHIFT(15),
  [44] = {.entry = {.count = 1, .reusable = true}}, SHIFT(16),
  [46] = {.entry = {.count = 1, .reusable = true}}, SHIFT(17),
  [48] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_boolean, 1),
  [50] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_boolean, 1),
  [52] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_ternary_expression, 5, .production_id = 4),
  [54] = {.entry = {.count = 1, .reusable = true}}, SHIFT(14),
  [56] = {.entry = {.count = 1, .reusable = true}}, SHIFT(18),
  [58] = {.entry = {.count = 1, .reusable = true}}, SHIFT(7),
  [60] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_binary_expression, 3, .production_id = 3),
  [62] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_unary_expression, 2, .production_id = 1),
  [64] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_unary_expression, 2, .production_id = 1),
  [66] = {.entry = {.count = 1, .reusable = true}}, SHIFT(22),
  [68] = {.entry = {.count = 1, .reusable = true}}, SHIFT(5),
  [70] = {.entry = {.count = 1, .reusable = true}}, SHIFT(12),
  [72] = {.entry = {.count = 1, .reusable = true}}, SHIFT(11),
  [74] = {.entry = {.count = 1, .reusable = true}}, SHIFT(9),
  [76] = {.entry = {.count = 1, .reusable = true}}, SHIFT(25),
  [78] = {.entry = {.count = 1, .reusable = true}}, SHIFT(24),
  [80] = {.entry = {.count = 1, .reusable = true}}, SHIFT(23),
  [82] = {.entry = {.count = 1, .reusable = true}}, SHIFT(10),
  [84] = {.entry = {.count = 1, .reusable = true}}, SHIFT(13),
  [86] = {.entry = {.count = 1, .reusable = true}}, SHIFT(4),
  [88] = {.entry = {.count = 1, .reusable = true}}, SHIFT(27),
  [90] = {.entry = {.count = 1, .reusable = true}}, SHIFT(8),
  [92] = {.entry = {.count = 1, .reusable = true}}, SHIFT(28),
  [94] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_print_statement, 3),
  [96] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_expression_statement, 2),
  [98] = {.entry = {.count = 1, .reusable = true}},  ACCEPT_INPUT(),
};

#ifdef __cplusplus
extern "C" {
#endif
#ifdef _WIN32
#define extern __declspec(dllexport)
#endif

extern const TSLanguage *tree_sitter_Lox(void) {
  static const TSLanguage language = {
    .version = LANGUAGE_VERSION,
    .symbol_count = SYMBOL_COUNT,
    .alias_count = ALIAS_COUNT,
    .token_count = TOKEN_COUNT,
    .external_token_count = EXTERNAL_TOKEN_COUNT,
    .state_count = STATE_COUNT,
    .large_state_count = LARGE_STATE_COUNT,
    .production_id_count = PRODUCTION_ID_COUNT,
    .field_count = FIELD_COUNT,
    .max_alias_sequence_length = MAX_ALIAS_SEQUENCE_LENGTH,
    .parse_table = &ts_parse_table[0][0],
    .small_parse_table = ts_small_parse_table,
    .small_parse_table_map = ts_small_parse_table_map,
    .parse_actions = ts_parse_actions,
    .symbol_names = ts_symbol_names,
    .field_names = ts_field_names,
    .field_map_slices = ts_field_map_slices,
    .field_map_entries = ts_field_map_entries,
    .symbol_metadata = ts_symbol_metadata,
    .public_symbol_map = ts_symbol_map,
    .alias_map = ts_non_terminal_alias_map,
    .alias_sequences = &ts_alias_sequences[0][0],
    .lex_modes = ts_lex_modes,
    .lex_fn = ts_lex,
    .primary_state_ids = ts_primary_state_ids,
  };
  return &language;
}
#ifdef __cplusplus
}
#endif
