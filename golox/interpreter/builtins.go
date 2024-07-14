package interpreter

import "time"

var builtinFns = map[string]loxObject{
	"clock": loxBuiltinFunction{
		name: "clock",
		fn: func([]loxObject) loxObject {
			return loxNumber(float64(time.Now().UnixNano()) / float64(time.Second))
		},
	},
}
