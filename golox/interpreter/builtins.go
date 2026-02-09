package interpreter

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
)

var builtinFunctions = map[string]*loxFunction{
	"clock": newBuiltinLoxFunction("clock", nil, func([]loxValue) loxValue {
		return loxNumber(time.Now().UnixNano()) / loxNumber(time.Second)
	}),
	"sleep": newBuiltinLoxFunction("sleep", []string{"duration"}, func(args []loxValue) loxValue {
		durationNumber, ok := args[0].(loxNumber)
		if !ok {
			return newErrorMsgf("expected sleep argument to be a %m, got %m", loxTypeNumber, args[0].Type())
		}
		durationDuration := time.Duration(durationNumber * loxNumber(time.Second))
		if durationDuration < 0 {
			return newErrorMsgf("expected sleep argument (%s) to be non-negative", durationNumber)
		}
		time.Sleep(durationDuration)
		return loxNil{}
	}),
	"type": newBuiltinLoxFunction("type", []string{"value"}, func(args []loxValue) loxValue {
		return loxString(args[0].Type())
	}),
	"parseNumber": newBuiltinLoxFunction("parseNumber", []string{"str"}, func(args []loxValue) loxValue {
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
	"getenv": newBuiltinLoxFunction("getenv", []string{"name"}, func(args []loxValue) loxValue {
		name, ok := args[0].(loxString)
		if !ok {
			return newErrorMsgf("expected getenv argument to be a %m, got %m", loxTypeString, args[0].Type())
		}
		value := os.Getenv(name.String())
		return loxString(value)
	}),
	"string": newBuiltinLoxFunction("string", []string{"value"}, func(args []loxValue) loxValue {
		return loxString(args[0].String())
	}),
	"error": newBuiltinLoxFunction("error", []string{"msg"}, func(args []loxValue) loxValue {
		return newErrorMsg(args[0].String())
	}),
	"printerr": newBuiltinLoxFunction("printerr", []string{"msg"}, func(args []loxValue) loxValue {
		fmt.Fprintln(os.Stderr, args[0].String())
		return loxNil{}
	}),
	"exit": newBuiltinLoxFunction("exit", []string{"code"}, func(args []loxValue) loxValue {
		codeNumber, ok := args[0].(loxNumber)
		if !ok {
			return newErrorMsgf("expected exit argument to be a %m, got %m", loxTypeNumber, args[0].Type())
		}
		if math.Floor(float64(codeNumber)) != float64(codeNumber) {
			return newErrorMsgf("expected exit argument (%s) to be an integer", codeNumber)
		}
		codeInt := int(codeNumber)
		os.Exit(codeInt)
		return loxNil{}
	}),
}
