.PHONY: clean objects FORCE

BASE_BUILD_DIR := build
CFLAGS := -std=c11 -Wall -Wextra -Wpedantic -Werror

ifdef DEBUG
	CFLAGS += -O0 -g -DDEBUG
	ifdef PRINT_BYTECODE
		CFLAGS += -DDEBUG_PRINT_BYTECODE
	endif
	ifdef TRACE_EXECUTION
		CFLAGS += -DDEBUG_TRACE_EXECUTION
	endif
	BUILD_DIR := $(BASE_BUILD_DIR)/debug
else
	CFLAGS += -O3 -flto
	BUILD_DIR := $(BASE_BUILD_DIR)/release
endif

clox: $(BUILD_DIR)/clox
	ln -sf $< $@
	@ printf "\x1b[32m$< linked to clox\x1b[0m\n"

SOURCES := $(wildcard *.c)
OBJECTS := $(addprefix $(BUILD_DIR)/, $(patsubst %.c, %.o, $(SOURCES)))

$(BUILD_DIR)/clox: $(OBJECTS)
	mkdir -p $(dir $@)
	$(CC) $(CFLAGS) $^ -o $@

HEADERS := $(wildcard *.h)

FORCE:

$(BUILD_DIR)/.cflags: FORCE
	@mkdir -p $(dir $@)
	@printf '%s\n' '$(CC) $(CFLAGS)' | cmp -s - $@ || printf '%s\n' '$(CC) $(CFLAGS)' > $@

$(BUILD_DIR)/%.o: %.c $(HEADERS) $(BUILD_DIR)/.cflags
	mkdir -p $(dir $@)
	$(CC) -c $(CFLAGS) -o $@ $<

clean:
	rm -rf $(BUILD_DIR) clox

objects: $(OBJECTS)

compile_commands.json: Makefile $(SOURCES) $(HEADERS) $(BUILD_DIR)/.cflags
	$(MAKE) clean DEBUG=1
	bear -- $(MAKE) -k objects DEBUG=1 PRINT_BYTECODE=1 TRACE_EXECUTION=1
