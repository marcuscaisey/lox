package interpreter

import "time"

var builtins = map[string]loxObject{
	"clock": loxBuiltinFunction{
		name: "clock",
		fn: func([]loxObject) loxObject {
			return loxNumber(float64(time.Now().UnixNano()) / float64(time.Second))
		},
	},
}
