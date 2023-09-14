package scanner

type Token int

type Scanner struct {
	src string
}

func New(src string) *Scanner {
	return &Scanner{
		src: src,
	}
}

func (s *Scanner) ScanTokens() []Token {
	return nil
}
