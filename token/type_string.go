// Code generated by "stringer -type Type -linecomment"; DO NOT EDIT.

package token

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[unknown-0]
	_ = x[keywordsStart-1]
	_ = x[Print-2]
	_ = x[Var-3]
	_ = x[True-4]
	_ = x[False-5]
	_ = x[Nil-6]
	_ = x[If-7]
	_ = x[Else-8]
	_ = x[And-9]
	_ = x[Or-10]
	_ = x[While-11]
	_ = x[For-12]
	_ = x[Function-13]
	_ = x[Return-14]
	_ = x[Class-15]
	_ = x[This-16]
	_ = x[Super-17]
	_ = x[keywordsEnd-18]
	_ = x[Semicolon-19]
	_ = x[Comma-20]
	_ = x[Dot-21]
	_ = x[literalsStart-22]
	_ = x[Ident-23]
	_ = x[String-24]
	_ = x[Number-25]
	_ = x[literalsEnd-26]
	_ = x[Assign-27]
	_ = x[Plus-28]
	_ = x[Minus-29]
	_ = x[Asterisk-30]
	_ = x[Slash-31]
	_ = x[Less-32]
	_ = x[LessEqual-33]
	_ = x[Greater-34]
	_ = x[GreaterEqual-35]
	_ = x[Equal-36]
	_ = x[NotEqual-37]
	_ = x[Bang-38]
	_ = x[OpenParen-39]
	_ = x[CloseParen-40]
	_ = x[OpenBrace-41]
	_ = x[CloseBrace-42]
	_ = x[EOF-43]
}

const _Type_name = "unknownkeywordsStartprintvartruefalsenilifelseandorwhileforfunreturnclassthissuperkeywordsEnd;,.literalsStartidentifierstringnumberliteralsEnd=+-*/<<=>>===!=!(){}EOF"

var _Type_index = [...]uint8{0, 7, 20, 25, 28, 32, 37, 40, 42, 46, 49, 51, 56, 59, 62, 68, 73, 77, 82, 93, 94, 95, 96, 109, 119, 125, 131, 142, 143, 144, 145, 146, 147, 148, 150, 151, 153, 155, 157, 158, 159, 160, 161, 162, 165}

func (i Type) String() string {
	if i >= Type(len(_Type_index)-1) {
		return "Type(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _Type_name[_Type_index[i]:_Type_index[i+1]]
}
