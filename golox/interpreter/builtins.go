package interpreter

import "time"

var builtins = []*loxBuiltinFunction{
	newLoxBuiltinFunction("clock", nil, func([]loxObject) loxObject {
		return loxNumber(time.Now().UnixNano()) / loxNumber(time.Second)
	}),
}
