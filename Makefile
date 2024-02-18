BUILD_DIR = ${PWD}/build

golox:
	go build -o ${BUILD_DIR}/golox .

extra_test_args =
ifdef RUN
	extra_test_args = -run ${RUN}
endif
test: golox
	go run github.com/rakyll/gotest ./test -interpreter ${BUILD_DIR}/golox ${extra_test_args}

update_tests: golox
	go run github.com/rakyll/gotest ./test -interpreter ${BUILD_DIR}/golox -update
