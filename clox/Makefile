.PHONY: clean objects

BASE_BUILD_DIR := build
CFLAGS := -std=c11 -Wall -Wextra -Wpedantic -Werror

ifdef DEBUG
	CFLAGS += -O0 -g -DDEBUG
	BUILD_DIR := $(BASE_BUILD_DIR)/debug
else
	CFLAGS += -O3 -flto
	BUILD_DIR := $(BASE_BUILD_DIR)/release
endif

SOURCES := $(wildcard *.c)
OBJECTS := $(addprefix $(BUILD_DIR)/, $(patsubst %.c, %.o, $(SOURCES)))

clox: $(BUILD_DIR)/clox
	ln -sf $< $@
	@ printf "\x1b[32m$< linked to clox\x1b[0m\n"

$(BUILD_DIR)/clox: $(OBJECTS)
	mkdir -p $(dir $@)
	$(CC) $(CFLAGS) $^ -o $@

HEADERS := $(wildcard *.h)

$(BUILD_DIR)/%.o: %.c $(HEADERS)
	mkdir -p $(dir $@)
	$(CC) -c $(CFLAGS) -o $@ $<

clean:
	rm -rf $(BUILD_DIR) clox

objects: $(OBJECTS)

compile_commands.json: Makefile $(SOURCES) $(HEADERS)
	$(MAKE) clean DEBUG=1
	bear -- $(MAKE) -k objects DEBUG=1
