package interpreter

import (
	"bytes"
	"fmt"
	"strings"
	"unicode"

	"github.com/mattn/go-runewidth"

	"github.com/marcuscaisey/lox/lox/ansi"
	"github.com/marcuscaisey/lox/lox/stack"
	"github.com/marcuscaisey/lox/lox/token"
)

type callStack struct {
	frames      *stack.Stack[*stackFrame]
	calledFuncs *stack.Stack[string]
}

// stackFrame points either to a function call or where an error occurred.
type stackFrame struct {
	Function string // Name of the function being executed, or empty if not in a function
	Location token.Position
}

func newCallStack() *callStack {
	callStack := &callStack{
		frames:      stack.New[*stackFrame](),
		calledFuncs: stack.New[string](),
	}
	callStack.calledFuncs.Push("")
	return callStack
}

func (cs *callStack) Push(function string, location token.Position) {
	cs.frames.Push(&stackFrame{
		Function: cs.calledFuncs.Peek(),
		Location: location,
	})
	cs.calledFuncs.Push(function)
}

func (cs *callStack) Pop() {
	cs.frames.Pop()
	cs.calledFuncs.Pop()
}

func (cs *callStack) Len() int {
	return cs.frames.Len()
}

func (cs *callStack) Clear() {
	cs.frames.Clear()
	cs.calledFuncs.Clear()
	cs.calledFuncs.Push("")
}

func (cs *callStack) StackTrace() string {
	var b strings.Builder
	ansi.Fprintln(&b, "${BOLD}Stack Trace (most recent call first):${RESET_BOLD}")
	locations := make([]string, cs.Len())
	locationWidth := 0
	functions := make([]string, cs.Len())
	functionWidth := 0
	lines := make([]string, cs.Len())
	for i, frame := range cs.frames.Backward() {
		locations[i] = fmt.Sprintf("%m", frame.Location)
		locationWidth = max(locationWidth, runewidth.StringWidth(locations[i]))
		function := ""
		if frame.Function != "" {
			function = fmt.Sprintf("in %s", frame.Function)
		}
		functions[i] = function
		functionWidth = max(functionWidth, runewidth.StringWidth(functions[i]))
		trimmedLine := string(bytes.TrimLeftFunc(frame.Location.File.Line(frame.Location.Line), unicode.IsSpace))
		lines[i] = ansi.Sprint("${FAINT}", trimmedLine, "${RESET_BOLD}")
	}
	for i := cs.Len() - 1; i >= 0; i-- {
		location := runewidth.FillRight(locations[i], locationWidth)
		function := runewidth.FillRight(functions[i], functionWidth)
		fmt.Fprint(&b, "  ", location, " ", function, " ", lines[i])
		if i > 0 {
			fmt.Fprintln(&b)
		}
	}
	return b.String()
}
