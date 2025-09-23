#!/usr/bin/env bash

jloxify="$1"
golox="$2"
repo="$3"
if [[ "$jloxify" == "" || "$golox" == "" || "$repo" == "" ]]; then
  echo "Usage: test.sh <jloxify> <golox> <repo>"
  exit 1
fi

cd "$repo" || exit 1
if [[ ! -f .deps ]]; then
  make get
  touch .deps
fi

# Our Lox flavour allows multiplying numbers and strings
rm -f test/operator/multiply_num_nonnum.lox
rm -f test/operator/multiply_nonnum_num.lox
# Our Lox flavour allows setting properites on classes
rm -f test/field/get_on_class.lox
rm -f test/field/set_on_class.lox
# Expecting different parser errors than golox raises
rm -f test/while/fun_in_body.lox
rm -f test/for/fun_in_body.lox
rm -f test/for/statement_condition.lox
rm -f test/for/statement_initializer.lox
rm -f test/function/missing_comma_in_parameters.lox
rm -f test/function/body_must_be_block.lox
rm -f test/unexpected_character.lox
rm -f test/if/fun_in_then.lox
rm -f test/if/fun_in_else.lox
# golox error doesn't have usual format
rm -f test/number/decimal_point_at_eof.lox
# golox doesn't support inheritance yet
rm -rf test/super
rm -rf test/inheritance
rm -f test/class/inherit_self.lox
rm -f test/class/inherited_method.lox
rm -f test/class/local_inherit_other.lox
rm -f test/class/local_inherit_self.lox
rm -f test/regression/394.lox

dart tool/bin/test.dart jlox -i ~/scratch/lox/build/jloxify -a ~/scratch/lox/build/golox
