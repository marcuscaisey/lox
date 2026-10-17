.PHONY: clean objects FORCE test/skipped_tests.txt

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


clox: $(BUILD_DIR)/clox $(BASE_BUILD_DIR)/.makeflags
	ln -sf $< $@
	@printf "\x1b[32m$< linked to clox\x1b[0m\n"

$(BASE_BUILD_DIR)/.makeflags: FORCE
	@mkdir -p $(dir $@)
	@printf '%s\n' '$(DEBUG)' | cmp -s - $@ || printf '%s\n' '$(DEBUG)' > $@

SOURCES := $(wildcard *.c)
OBJECTS := $(addprefix $(BUILD_DIR)/, $(patsubst %.c, %.o, $(SOURCES)))

$(BUILD_DIR)/clox: $(OBJECTS)
	@mkdir -p $(dir $@)
	$(CC) $(CFLAGS) $^ -o $@

HEADERS := $(wildcard *.h)

FORCE:

$(BUILD_DIR)/.cflags: FORCE
	@mkdir -p $(dir $@)
	@printf '%s\n' '$(CC) $(CFLAGS)' | cmp -s - $@ || printf '%s\n' '$(CC) $(CFLAGS)' > $@

$(BUILD_DIR)/%.o: %.c $(HEADERS) $(BUILD_DIR)/.cflags
	@mkdir -p $(dir $@)
	$(CC) -c $(CFLAGS) -o $@ $<

clean:
	rm -rf $(BUILD_DIR) clox

objects: $(OBJECTS)

compile_commands.json: Makefile $(SOURCES) $(HEADERS) $(BUILD_DIR)/.cflags
	$(MAKE) clean DEBUG=1
	bear -- $(MAKE) -k objects DEBUG=1 PRINT_BYTECODE=1 TRACE_EXECUTION=1

test/skipped_tests.txt:
	# The jq filter selects only failing tests whose names are not a prefix of another
	@go test ./test -json | jq --raw-output --slurp ' \
		[.[] | select(.Action == "fail" and .Test != null)] | . as $$failures \
		| .[] | select( \
			. as $$parent | all($$failures[]; .Test | startswith($$parent.Test + "/") | not) \
		) \
		| .Test \
	' | sort > $@
