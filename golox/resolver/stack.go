package resolver

type stack[T any] []T

func newStack[T any]() *stack[T] {
	return &stack[T]{}
}

func (s *stack[T]) Push(v T) {
	*s = append(*s, v)
}

func (s *stack[T]) Pop() T {
	if len(*s) == 0 {
		panic("pop from empty stack")
	}
	v := (*s)[len(*s)-1]
	*s = (*s)[:len(*s)-1]
	return v
}

func (s *stack[T]) Peek() T {
	if len(*s) == 0 {
		panic("peek of empty stack")
	}
	return (*s)[len(*s)-1]
}

func (s *stack[T]) Len() int {
	return len(*s)
}

func (s *stack[T]) Index(i int) T {
	return (*s)[i]
}
