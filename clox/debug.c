#include "debug.h"

#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "bytecode.h"
#include "line_number.h"
#include "value.h"

void disassemble(const struct bytecode_chunk *chunk, const char *name)
{
    printf("== %s ==\n", name);
    for (uint8_t *ip = chunk->instructions; ip < chunk->instructions + chunk->len;)
        ip = disassemble_instruction(chunk, ip);
}

uint8_t *disassemble_instruction(const struct bytecode_chunk *chunk, uint8_t *ip)
{
    int offset = ip - chunk->instructions;
    int offset_width = 4;
    printf("%0*d ", offset_width, offset);

    int line_width = 4;
    line_number line = chunk->offset_lines[offset];
    if (offset > 0 && line == chunk->offset_lines[offset - 1])
        printf("%*s ", line_width, "|");
    else
        printf("%*d ", line_width, line);

    // Width of longest name of instruction which accepts an operand
    int name_width = strlen(instruction_name(OP_DEFINE_GLOBAL_FINAL));
    int operand_width = 4;

    enum opcode opcode = bytecode_read_opcode(&ip);
    const char *instruction = instruction_name(opcode);
    switch (opcode) {
    case OP_CONSTANT: {
        constant_index index = bytecode_read_constant_index(&ip);
        char *value = value_string(chunk->constants[index]);
        printf("%-*s %*d '%s'\n", name_width, instruction, operand_width, index, value);
        free(value);
        return ip;
    }

    case OP_POP_N:
    case OP_CLOSE_UPVALUE_N: {
        pop_count count = bytecode_read_pop_count(&ip);
        printf("%-*s %*d\n", name_width, instruction, operand_width, count);
        return ip;
    }

    case OP_DEFINE_GLOBAL:
    case OP_DEFINE_GLOBAL_FINAL:
    case OP_GET_GLOBAL:
    case OP_SET_GLOBAL: {
        global_index index = bytecode_read_global_index(&ip);
        printf("%-*s %*d\n", name_width, instruction, operand_width, index);
        return ip;
    }

    case OP_DEFINE_LOCAL_FINAL: {
        final_local_name_index index = bytecode_read_final_local_name_index(&ip);
        printf("%-*s %*d '%s'\n", name_width, instruction, operand_width, index,
               chunk->final_local_names[index]);
        return ip;
    }

    case OP_GET_UPVALUE:
    case OP_SET_UPVALUE: {
        upvalue_index index = bytecode_read_upvalue_index(&ip);
        printf("%-*s %*d\n", name_width, instruction, operand_width, index);
        return ip;
    }

    case OP_GET_LOCAL:
    case OP_SET_LOCAL: {
        stack_index index = bytecode_read_stack_index(&ip);
        printf("%-*s %*d\n", name_width, instruction, operand_width, index);
        return ip;
    }

    case OP_JUMP:
    case OP_JUMP_BACK:
    case OP_JUMP_IF_TRUE:
    case OP_JUMP_IF_FALSE: {
        jump_offset jump_offset = bytecode_read_jump_offset(&ip);
        int target_offset =
            ip - chunk->instructions + (opcode == OP_JUMP_BACK ? -1 : 1) * jump_offset;
        printf("%-*s %*d -> %0*d\n", name_width, instruction, operand_width, jump_offset,
               offset_width, target_offset);
        return ip;
    }

    case OP_CALL: {
        argument_count count = bytecode_read_argument_count(&ip);
        printf("%-*s %*d\n", name_width, instruction, operand_width, count);
        return ip;
    }

    case OP_CLOSURE: {
        constant_index function_index = bytecode_read_constant_index(&ip);
        char *function_string = value_string(chunk->constants[function_index]);
        printf("%-*s %*d %s\n", name_width, instruction, operand_width, function_index,
               function_string);
        free(function_string);
        upvalue_count count = bytecode_read_upvalue_count(&ip);
        for (upvalue_count i = 0; i < count; i++) {
            printf("%0*d %*s %-*s %*s ", offset_width, offset, line_width, "", name_width, "|",
                   operand_width, "");
            bool is_stack_index = *ip++;
            if (is_stack_index) {
                stack_index index = bytecode_read_stack_index(&ip);
                printf("local %d\n", index);
            } else {
                upvalue_index index = bytecode_read_upvalue_index(&ip);
                printf("upvalue %d\n", index);
            }
        }
        return ip;
    }

    case OP_NIL:
    case OP_TRUE:
    case OP_FALSE:
    case OP_POP:
    case OP_DUP:
    case OP_DEFINE_LOCAL:
    case OP_EQUAL:
    case OP_NOT_EQUAL:
    case OP_LESS:
    case OP_LESS_EQUAL:
    case OP_GREATER:
    case OP_GREATER_EQUAL:
    case OP_ADD:
    case OP_SUBTRACT:
    case OP_MULTIPLY:
    case OP_DIVIDE:
    case OP_MODULO:
    case OP_NOT:
    case OP_NEGATE:
    case OP_PRINT:
    case OP_CLOSE_UPVALUE:
    case OP_RETURN:
        printf("%s\n", instruction);
        return ip;
    }

    printf("Unknown opcode %d\n", opcode);
    return ip;
}
