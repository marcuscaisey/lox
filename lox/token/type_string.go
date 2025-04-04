// Code generated by "stringer -type Type"; DO NOT EDIT.

package token

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[Illegal-0]
	_ = x[EOF-1]
	_ = x[keywordsStart-2]
	_ = x[Print-3]
	_ = x[Var-4]
	_ = x[True-5]
	_ = x[False-6]
	_ = x[Nil-7]
	_ = x[If-8]
	_ = x[Else-9]
	_ = x[And-10]
	_ = x[Or-11]
	_ = x[While-12]
	_ = x[For-13]
	_ = x[Break-14]
	_ = x[Continue-15]
	_ = x[Fun-16]
	_ = x[Return-17]
	_ = x[Class-18]
	_ = x[This-19]
	_ = x[Super-20]
	_ = x[Static-21]
	_ = x[Get-22]
	_ = x[Set-23]
	_ = x[keywordsEnd-24]
	_ = x[Ident-25]
	_ = x[String-26]
	_ = x[Number-27]
	_ = x[Comment-28]
	_ = x[Semicolon-29]
	_ = x[Comma-30]
	_ = x[Dot-31]
	_ = x[Equal-32]
	_ = x[Plus-33]
	_ = x[Minus-34]
	_ = x[Asterisk-35]
	_ = x[Slash-36]
	_ = x[Percent-37]
	_ = x[Less-38]
	_ = x[LessEqual-39]
	_ = x[Greater-40]
	_ = x[GreaterEqual-41]
	_ = x[EqualEqual-42]
	_ = x[BangEqual-43]
	_ = x[Bang-44]
	_ = x[Question-45]
	_ = x[Colon-46]
	_ = x[LeftParen-47]
	_ = x[RightParen-48]
	_ = x[LeftBrace-49]
	_ = x[RightBrace-50]
	_ = x[typesEnd-51]
}

const _Type_name = "IllegalEOFkeywordsStartPrintVarTrueFalseNilIfElseAndOrWhileForBreakContinueFunReturnClassThisSuperStaticGetSetkeywordsEndIdentStringNumberCommentSemicolonCommaDotEqualPlusMinusAsteriskSlashPercentLessLessEqualGreaterGreaterEqualEqualEqualBangEqualBangQuestionColonLeftParenRightParenLeftBraceRightBracetypesEnd"

var _Type_index = [...]uint16{0, 7, 10, 23, 28, 31, 35, 40, 43, 45, 49, 52, 54, 59, 62, 67, 75, 78, 84, 89, 93, 98, 104, 107, 110, 121, 126, 132, 138, 145, 154, 159, 162, 167, 171, 176, 184, 189, 196, 200, 209, 216, 228, 238, 247, 251, 259, 264, 273, 283, 292, 302, 310}

func (i Type) String() string {
	if i < 0 || i >= Type(len(_Type_index)-1) {
		return "Type(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _Type_name[_Type_index[i]:_Type_index[i+1]]
}
