package interpreter

import (
	"time"
)

var builtIns = map[string]loxObject{
	"clock": newBuiltInLoxFunction("clock", nil, func([]loxObject) loxObject {
		return loxNumber(time.Now().UnixNano()) / loxNumber(time.Second)
	}),
	"type": newBuiltInLoxFunction("type", []string{"object"}, func(args []loxObject) loxObject {
		return loxString(args[0].Type())
	}),
	"error": newBuiltInLoxFunction("error", []string{"msg"}, func(args []loxObject) loxObject {
		return errorMsg(args[0].String())
	}),
}
