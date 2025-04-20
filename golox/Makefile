.PHONY: golox test update_tests

BUILD_PATH = ${PWD}/build/golox

golox:
	go build -o ${BUILD_PATH} .

extra_test_args =
ifdef RUN
	extra_test_args = -run ${RUN}
endif

test: golox
	go test ../test -pwd=${PWD} -interpreter=${BUILD_PATH} ${extra_test_args}

update_tests: golox
	go test ../test -pwd=${PWD} -interpreter=${BUILD_PATH} -update ${extra_test_args}
