package interpreter

import "time"

var builtinsByName = map[string]loxObject{
	"clock": newBuiltinLoxFunction("clock", nil, func([]loxObject) loxObject {
		return loxNumber(time.Now().UnixNano()) / loxNumber(time.Second)
	}),
	"type": newBuiltinLoxFunction("type", []string{"object"}, func(args []loxObject) loxObject {
		return loxString(args[0].Type())
	}),
}
