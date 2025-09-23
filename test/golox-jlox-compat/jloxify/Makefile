.PHONY: jloxify

MAKEFILE_DIR := $(dir $(abspath $(lastword ${MAKEFILE_LIST})))
ROOT_DIR := $(realpath ${MAKEFILE_DIR}/../../..)
BUILD_DIR := ${ROOT_DIR}/build
BUILD_PATH := ${BUILD_DIR}/jloxify

jloxify:
	go build -o ${BUILD_PATH} .
