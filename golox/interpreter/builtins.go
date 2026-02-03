package interpreter

import (
	"strconv"
	"strings"
	"time"
)

var builtIns = map[string]loxValue{
	"clock": newBuiltInLoxFunction("clock", nil, func([]loxValue) loxValue {
		return loxNumber(time.Now().UnixNano()) / loxNumber(time.Second)
	}),
	"type": newBuiltInLoxFunction("type", []string{"value"}, func(args []loxValue) loxValue {
		return loxString(args[0].Type())
	}),
	"parseNumber": newBuiltInLoxFunction("parseNumber", []string{"str"}, func(args []loxValue) loxValue {
		str, ok := args[0].(loxString)
		if !ok {
			return newErrorMsgf("expected parseNumber argument to be a %m, got %m", loxTypeString, args[0].Type())
		}
		f, err := strconv.ParseFloat(strings.TrimSpace(str.String()), 64)
		if err != nil {
			return newErrorMsgf("%q could not be parsed as a %m", str, loxTypeNumber)
		}
		return loxNumber(f)
	}),
	"error": newBuiltInLoxFunction("error", []string{"msg"}, func(args []loxValue) loxValue {
		return newErrorMsg(args[0].String())
	}),
}
