package interpreter

import (
	"time"
)

var builtins = map[string]loxObject{
	"clock": newBuiltinLoxFunction("clock", nil, func([]loxObject) loxObject {
		return loxNumber(time.Now().UnixNano()) / loxNumber(time.Second)
	}),
	"type": newBuiltinLoxFunction("type", []string{"object"}, func(args []loxObject) loxObject {
		return loxString(args[0].Type())
	}),
	"error": newBuiltinLoxFunction("error", []string{"msg"}, func(args []loxObject) loxObject {
		return errorMsg(args[0].String())
	}),
}
