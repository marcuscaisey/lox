.PHONY: clean clean-all

BASE_BUILD_DIR := build
CFLAGS := -std=c99 -Wall -Wextra -Wpedantic -Wswitch-enum -Werror

ifdef DEBUG
	CFLAGS += -O0 -g
	BUILD_DIR := $(BASE_BUILD_DIR)/debug
else
	CFLAGS += -O3
	BUILD_DIR := $(BASE_BUILD_DIR)/release
endif

SOURCES := $(wildcard *.c)
OBJECTS := $(addprefix $(BUILD_DIR)/, $(patsubst %.c, %.o, $(SOURCES)))

$(BUILD_DIR)/clox: $(OBJECTS)
	mkdir -p $(dir $@)
	$(CC) $(CFLAGS) $^ -o $@
	@ printf "\x1b[32mclox output to $@\x1b[0m\n"

HEADERS := $(wildcard *.h)

$(BUILD_DIR)/%.o: %.c $(HEADERS)
	mkdir -p $(dir $@)
	$(CC) -c $(CFLAGS) -o $@ $<

clean:
	rm -rf $(BUILD_DIR)

clean-all:
	rm -rf $(BASE_BUILD_DIR)

compile_commands.json: Makefile $(SOURCES) $(HEADERS)
	$(MAKE) clean
	bear -- make
