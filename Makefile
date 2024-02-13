BUILD_DIR := ${PWD}/build

golox:
	go build -o ${BUILD_DIR}/golox .

test: golox
	go run github.com/rakyll/gotest ./test -interpreter ${BUILD_DIR}/golox

update_tests: golox
	go run github.com/rakyll/gotest ./test -interpreter ${BUILD_DIR}/golox -update
