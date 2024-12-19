package interpreter

import (
	"time"

	"github.com/marcuscaisey/lox/lox"
)

var builtins = map[string]loxObject{
	lox.BuiltinClock: newBuiltinLoxFunction(lox.BuiltinClock, nil, func([]loxObject) loxObject {
		return loxNumber(time.Now().UnixNano()) / loxNumber(time.Second)
	}),
	lox.BuiltinType: newBuiltinLoxFunction(lox.BuiltinType, []string{"object"}, func(args []loxObject) loxObject {
		return loxString(args[0].Type())
	}),
	lox.BuiltinError: newBuiltinLoxFunction(lox.BuiltinError, []string{"msg"}, func(args []loxObject) loxObject {
		return errorMsg(args[0].String())
	}),
}
