// Package stack implements a generic LIFO stack.
package stack

import (
	"fmt"
	"iter"
	"strings"
)

// Stack is a generic LIFO stack.
type Stack[E any] []E

// New creates a new stack.
func New[E any]() *Stack[E] {
	return &Stack[E]{}
}

// Push pushes a value onto the stack.
func (s *Stack[E]) Push(v E) {
	*s = append(*s, v)
}

// Pop pops a value from the stack and returns it.
// If the stack is empty, it panics.
func (s *Stack[E]) Pop() E {
	if len(*s) == 0 {
		panic("pop from empty stack")
	}
	v := (*s)[len(*s)-1]
	*s = (*s)[:len(*s)-1]
	return v
}

// Peek returns the top value of the stack without removing it.
// If the stack is empty, it panics.
func (s *Stack[E]) Peek() E {
	if len(*s) == 0 {
		panic("peek of empty stack")
	}
	return (*s)[len(*s)-1]
}

// Len returns the number of elements in the stack.
func (s *Stack[E]) Len() int {
	return len(*s)
}

// Clear removes all elements from the stack.
func (s *Stack[E]) Clear() {
	*s = (*s)[:0]
}

// Backward returns an iterator over index-value pairs in the stack, traversing it backward with descending indices.
func (s *Stack[E]) Backward() iter.Seq2[int, E] {
	return func(yield func(int, E) bool) {
		for i := s.Len() - 1; i >= 0; i-- {
			if !yield(i, (*s)[i]) {
				return
			}
		}
	}
}

func (s *Stack[E]) String() string {
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
