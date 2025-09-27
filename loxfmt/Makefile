.PHONY: loxfmt test update_tests

MAKEFILE_DIR := $(dir $(abspath $(lastword ${MAKEFILE_LIST})))
ROOT_DIR := $(realpath ${MAKEFILE_DIR}/..)
BUILD_DIR := ${ROOT_DIR}/build
BUILD_PATH := ${BUILD_DIR}/loxfmt

loxfmt:
	go build -o ${BUILD_PATH} .

extra_test_args =
ifdef RUN
	extra_test_args = -run ${RUN}
endif

test: loxfmt
	go test ../test -pwd=${PWD} -formatter=${BUILD_PATH} ${extra_test_args}

update_tests: loxfmt
	go test ../test -pwd=${PWD} -formatter=${BUILD_PATH} -update ${extra_test_args}
