.PHONY: test

MAKEFILE_DIR := $(dir $(abspath $(lastword ${MAKEFILE_LIST})))
ROOT_DIR := $(realpath ${MAKEFILE_DIR}/../..)
BUILD_DIR := ${ROOT_DIR}/build

CRAFTING_INTERPRETERS_PATH := ${BUILD_DIR}/craftinginterpreters

test:
	$(MAKE) -C jloxify
	$(MAKE) -C ${ROOT_DIR}/golox
	-[ ! -d ${CRAFTING_INTERPRETERS_PATH} ] && git clone https://github.com/munificent/craftinginterpreters.git --depth 1 ${CRAFTING_INTERPRETERS_PATH}
	./test.sh ${BUILD_DIR}/jloxify ${BUILD_DIR}/golox ${CRAFTING_INTERPRETERS_PATH}
