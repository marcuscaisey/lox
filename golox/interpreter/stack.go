package interpreter

import (
	"fmt"
	"iter"    //nolint:gci
	"strings" //nolint:gci
)

type stack[E any] []E

func newStack[E any]() *stack[E] {
	return &stack[E]{}
}

func (s *stack[E]) Push(v E) {
	*s = append(*s, v)
}

func (s *stack[E]) Pop() E {
	if len(*s) == 0 {
		panic("pop from empty stack")
	}
	v := (*s)[len(*s)-1]
	*s = (*s)[:len(*s)-1]
	return v
}

func (s *stack[E]) Peek() E {
	if len(*s) == 0 {
		panic("peek of empty stack")
	}
	return (*s)[len(*s)-1]
}

func (s *stack[E]) Len() int {
	return len(*s)
}

func (s *stack[E]) Clear() {
	*s = (*s)[:0]
}

func (s *stack[E]) Backward() iter.Seq2[int, E] {
	return func(yield func(int, E) bool) {
		for i := s.Len() - 1; i >= 0; i-- {
			if !yield(i, (*s)[i]) {
				return
			}
		}
	}
}

func (s *stack[E]) String() string {
	var b strings.Builder
	fmt.Fprint(&b, "stack([")
	for i, v := range *s {
		fmt.Fprintf(&b, "%v", v)
		if i < len(*s)-1 {
			fmt.Fprint(&b, ", ")
		}
	}
	fmt.Fprint(&b, "])")
	return b.String()
}
