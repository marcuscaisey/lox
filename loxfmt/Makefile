.PHONY: loxfmt test update_tests

BUILD_PATH = ${PWD}/build/loxfmt

loxfmt:
	go build -o ${BUILD_PATH} .

extra_test_args =
ifdef RUN
	extra_test_args = -run ${RUN}
endif

test: loxfmt
	go run gotest.tools/gotestsum ../test -formatter=${BUILD_PATH} ${extra_test_args}

update_tests: loxfmt
	go run gotest.tools/gotestsum ../test -formatter=${BUILD_PATH} -update ${extra_test_args}
