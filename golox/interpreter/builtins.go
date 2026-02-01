package interpreter

import (
	"time"
)

var builtIns = map[string]loxValue{
	"clock": newBuiltInLoxFunction("clock", nil, func([]loxValue) loxValue {
		return loxNumber(time.Now().UnixNano()) / loxNumber(time.Second)
	}),
	"type": newBuiltInLoxFunction("type", []string{"value"}, func(args []loxValue) loxValue {
		return loxString(args[0].Type())
	}),
	"error": newBuiltInLoxFunction("error", []string{"msg"}, func(args []loxValue) loxValue {
		return errorMsg(args[0].String())
	}),
}
