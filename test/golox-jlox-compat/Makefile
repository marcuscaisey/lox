.PHONY: test

MAKEFILE_DIR := $(dir $(abspath $(lastword ${MAKEFILE_LIST})))
ROOT_DIR := $(realpath ${MAKEFILE_DIR}/../..)
BUILD_DIR := ${ROOT_DIR}/build
GOLOX_BUILD_PATH := ${BUILD_DIR}/golox
JLOXIFY_BUILD_PATH := ${BUILD_DIR}/jloxify

CRAFTING_INTERPRETERS_PATH := ${BUILD_DIR}/craftinginterpreters

test:
	go build -o ${JLOXIFY_BUILD_PATH} ./jloxify
	go build -o ${GOLOX_BUILD_PATH} ${ROOT_DIR}/golox
	-[ ! -d ${CRAFTING_INTERPRETERS_PATH} ] && git clone https://github.com/munificent/craftinginterpreters.git --depth 1 ${CRAFTING_INTERPRETERS_PATH}
	./test.sh ${BUILD_DIR}/jloxify ${GOLOX_BUILD_PATH} ${CRAFTING_INTERPRETERS_PATH}
