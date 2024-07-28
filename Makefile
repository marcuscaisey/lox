.PHONY: golox test update_tests

BUILD_DIR = ${PWD}/build
GOLOX_BUILD_PATH = ${BUILD_DIR}/golox

golox:
	go build -o ${GOLOX_BUILD_PATH} github.com/marcuscaisey/lox/golox

extra_test_args =
ifdef RUN
	extra_test_args = -run ${RUN}
endif

test_golox: golox
	go run gotest.tools/gotestsum ./test -interpreter=${GOLOX_BUILD_PATH} ${extra_test_args}

update_tests: golox
	go run gotest.tools/gotestsum ./test -interpreter=${GOLOX_BUILD_PATH} -update ${extra_test_args}
