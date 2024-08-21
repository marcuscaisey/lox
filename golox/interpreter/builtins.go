package interpreter

import (
	"time"

	"github.com/marcuscaisey/lox/golox/lox"
)

var builtinsByName = map[string]loxObject{
	"clock": newBuiltinLoxFunction("clock", nil, func(*Interpreter, []loxObject) loxObject {
		return loxNumber(time.Now().UnixNano()) / loxNumber(time.Second)
	}),
	"type": newBuiltinLoxFunction("type", []string{"object"}, func(_ *Interpreter, args []loxObject) loxObject {
		return loxString(args[0].Type())
	}),
	"error": newBuiltinLoxFunction("error", []string{"message"}, func(i *Interpreter, args []loxObject) loxObject {
		lastFrame := i.callStack.Peek()
		panic(lox.NewError(lastFrame.CallStart, lastFrame.CallEnd, "%s", args[0].String()))
	}),
}
