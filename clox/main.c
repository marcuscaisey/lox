#include <errno.h>
#include <stdbool.h>
#include <stddef.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "vm.h"

// Reads `filename` and returns its contents. If the operation fails, `NULL` is returned and the
// global variable `errno` is set to indicate the error.
static char *read_file(const char *filename)
{
    FILE *file = fopen(filename, "rb");
    if (file == NULL)
        return NULL;
    char *file_contents = NULL;
    if (fseek(file, 0, SEEK_END))
        goto out_close_file;
    long file_size = ftell(file);
    if (file_size == -1)
        goto out_close_file;
    if (fseek(file, 0, SEEK_SET))
        goto out_close_file;
    file_contents = malloc(file_size + 1);
    if (file_contents == NULL)
        goto out_close_file;
    size_t bytes_read = fread(file_contents, sizeof(*file_contents), file_size, file);
    if (bytes_read < (size_t)file_size && ferror(file))
        goto out_free_file_contents;
    file_contents[bytes_read] = '\0';
    goto out_close_file;

out_free_file_contents:
    free(file_contents);
    file_contents = NULL;
out_close_file:
    fclose(file);
    return file_contents;
}

// Runs `filename` and returns an appropriate exit status.
static int run_file(const char *filename)
{
    vm vm;
    vm_init(&vm);

    char *source = read_file(filename);
    if (source == NULL) {
        fprintf(stderr, "clox: reading %s: %s\n", filename, strerror(errno));
        return 1;
    }

    bool success = vm_interpret(&vm, source);

    vm_deinit(&vm);
    free(source);

    return success ? 0 : 1;
}

int main(int argc, char *argv[])
{
    if (argc != 2) {
        fprintf(stderr, "Usage: clox [<script>]\n");
        return 2;
    }
    char *filename = argv[1];
    return run_file(filename);
}
