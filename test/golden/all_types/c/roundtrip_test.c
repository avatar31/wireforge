/*
 * roundtrip_test.c — Runtime tests for generated C wire-protocol code.
 *
 * Tests cover:
 *   - Scalar roundtrip (marshal → unmarshal → field equality)
 *   - Variable-length field roundtrip (string, bytes, nested struct)
 *   - Wire frame header layout and Big-Endian encoding
 *   - Error cases (truncated buffer, NULL pointer)
 *   - Free / double-free safety
 *   - Dynamic payload size computation
 */

#include <assert.h>
#include <math.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "messages.h"

/* ---------------------------------------------------------------------------
 * Assertion helpers
 * --------------------------------------------------------------------------*/

#define TEST(name)                          \
    do {                                    \
        printf("  %-60s", #name "...");     \
        fflush(stdout);                     \
    } while (0)

#define PASS()  do { printf("PASS\n"); } while (0)
#define FAIL(msg)                                                           \
    do {                                                                    \
        fprintf(stderr, "FAIL\n  %s:%d: %s\n", __FILE__, __LINE__, msg);   \
        exit(1);                                                            \
    } while (0)

#define ASSERT(cond)       do { if (!(cond))  FAIL("assertion failed: " #cond); } while (0)
#define ASSERT_EQ_INT(a,b) do { if ((a)!=(b)) { fprintf(stderr, "FAIL\n  %s:%d: expected %lld, got %lld\n", __FILE__, __LINE__, (long long)(b), (long long)(a)); exit(1); } } while (0)
#define ASSERT_EQ_FLT(a,b) do { if (fabs((double)(a)-(double)(b)) > 1e-6) FAIL("float mismatch"); } while (0)
#define ASSERT_EQ_STR(a,b) do { if (strcmp((a),(b)) != 0) FAIL("string mismatch"); } while (0)
#define ASSERT_MEM_EQ(a,b,n) do { if (memcmp((a),(b),(n)) != 0) FAIL("memory mismatch"); } while (0)
#define ASSERT_BYTE_ARRAY(a,b) do { if ((a).len != (b).len || ((a).len > 0 && memcmp((a).data, (b).data, (a).len) != 0)) FAIL("byte array mismatch"); } while (0)
#define ASSERT_STRING(a,b) do { if ((a).len != (b).len || ((a).len > 0 && memcmp((a).data, (b).data, (a).len) != 0)) FAIL("string mismatch"); } while (0)

/**---------------------------------------------------------------------------
 * Helpers
 * ---------------------------------------------------------------------------*/

 static void initialize_only_scalar_types_msg_t(only_scalar_types_msg_t *msg)
{
    memset(msg, 0, sizeof(*msg));

    only_scalar_types_msg_t_set_val_uint64(msg, 0xDEADBEEFCAFEBABEULL);
    only_scalar_types_msg_t_set_val_int64(msg, -9223372036854775807LL);
    only_scalar_types_msg_t_set_val_double(msg, 3.141592653589793);
    only_scalar_types_msg_t_set_val_uint32(msg, 0xFFFFFFFF);
    only_scalar_types_msg_t_set_val_int32(msg, -2147483648);
    only_scalar_types_msg_t_set_val_float(msg, 1.23456789f);
    only_scalar_types_msg_t_set_val_uint16(msg, 65535);
    only_scalar_types_msg_t_set_val_int16(msg, -32768);
    only_scalar_types_msg_t_set_val_uint8(msg, 255);
    only_scalar_types_msg_t_set_val_int8(msg, -128);
    only_scalar_types_msg_t_set_val_bool(msg, 1);
}

static void compare_only_scalar_types_msg_t(const only_scalar_types_msg_t *a, const only_scalar_types_msg_t *b)
{
    ASSERT_EQ_INT(a->val_uint64, b->val_uint64);
    ASSERT_EQ_INT(a->val_int64, b->val_int64);
    ASSERT_EQ_FLT(a->val_double, b->val_double);
    ASSERT_EQ_INT(a->val_uint32, b->val_uint32);
    ASSERT_EQ_INT(a->val_int32, b->val_int32);
    ASSERT_EQ_FLT(a->val_float, b->val_float);
    ASSERT_EQ_INT(a->val_uint16, b->val_uint16);
    ASSERT_EQ_INT(a->val_int16, b->val_int16);
    ASSERT_EQ_INT(a->val_uint8, b->val_uint8);
    ASSERT_EQ_INT(a->val_int8, b->val_int8);
    ASSERT_EQ_INT(a->val_bool, b->val_bool);
}

static int initialize_only_variable_types_msg_t(only_variable_types_msg_t *msg)
{
    memset(msg, 0, sizeof(*msg));

    only_variable_types_msg_t_set_name(msg, "hello world", 11);
    uint8_t b[] = {0xDE, 0xAD, 0xBE, 0xEF};
    only_variable_types_msg_t_set_data(msg, b, 4);

    only_scalar_types_msg_t inner = {0};
    only_scalar_types_msg_t_set_val_uint64(&inner, 42ULL);
    only_variable_types_msg_t_set_nested(msg, &inner);

    dynamic_array_t tags = {0};
    dynamic_array_init(&tags, TAG_STRING, sizeof (string_t), 1, 4, 1, 1);
    string_t str1 = {.data = (char *)"alpha", .len = 5};
    ASSERT(dynamic_array_set(&tags, 0, 0, 0, &str1) == DYN_ARR_OK);
    string_t str2 = {.data = (char *)"beta", .len = 4};
    ASSERT(dynamic_array_set(&tags, 1, 0, 0, &str2) == DYN_ARR_OK);
    string_t str3 = {.data = (char *)"gamma", .len = 5};
    ASSERT(dynamic_array_set(&tags, 2, 0, 0, &str3) == DYN_ARR_OK);
    string_t str4 = {.data = (char *)"xyz", .len = 3};
    ASSERT(dynamic_array_set(&tags, 3, 0, 0, &str4) == DYN_ARR_OK);
    only_variable_types_msg_t_set_tags(msg, &tags);
    dynamic_array_destroy(&tags);

    dynamic_array_t matrix = {0};
    dynamic_array_init(&matrix, TAG_INT32, sizeof (int32_t), 2, 3, 3, 1);
    ASSERT(dynamic_array_set(&matrix, 0, 0, 0, &(int32_t){1}) == DYN_ARR_OK);
    ASSERT(dynamic_array_set(&matrix, 0, 1, 0, &(int32_t){2}) == DYN_ARR_OK);
    ASSERT(dynamic_array_set(&matrix, 0, 2, 0, &(int32_t){3}) == DYN_ARR_OK);
    ASSERT(dynamic_array_set(&matrix, 1, 0, 0, &(int32_t){4}) == DYN_ARR_OK);
    ASSERT(dynamic_array_set(&matrix, 1, 1, 0, &(int32_t){5}) == DYN_ARR_OK);
    ASSERT(dynamic_array_set(&matrix, 1, 2, 0, &(int32_t){6}) == DYN_ARR_OK);
    ASSERT(dynamic_array_set(&matrix, 2, 0, 0, &(int32_t){7}) == DYN_ARR_OK);
    ASSERT(dynamic_array_set(&matrix, 2, 1, 0, &(int32_t){8}) == DYN_ARR_OK);
    ASSERT(dynamic_array_set(&matrix, 2, 2, 0, &(int32_t){9}) == DYN_ARR_OK);
    only_variable_types_msg_t_set_matrix(msg, &matrix);
    dynamic_array_destroy(&matrix);

    dynamic_array_t byte_array = {0};
    dynamic_array_init(&byte_array, TAG_BYTES, sizeof (byte_array_t), 1, 2, 1, 1);
    static uint8_t buf1[] = {0x01, 0x02};
    byte_array_t data1 = {.data = buf1, .len = 2};
    ASSERT(dynamic_array_set(&byte_array, 0, 0, 0, &data1) == DYN_ARR_OK);
    static uint8_t buf2[] = {0x03, 0x04, 0x05};
    byte_array_t data2 = {.data = buf2, .len = 3};
    ASSERT(dynamic_array_set(&byte_array, 1, 0, 0, &data2) == DYN_ARR_OK);
    only_variable_types_msg_t_set_byte_array(msg, &byte_array);
    dynamic_array_destroy(&byte_array);

    return 183; /* expected dynamic payload size for this test case, from roundtrip_test.go */
}

static void dynamic_array_compare(uint16_t ele_type, const dynamic_array_t *a,
                            const dynamic_array_t *b)
{
    if (!a || !b)
        FAIL("dynamic_array_equal: NULL pointer");

    if (a->num_dims != b->num_dims ||
        a->x != b->x ||
        a->y != b->y ||
        a->z != b->z)
        FAIL("array dimension mismatch");

    switch (ele_type) {
        case TAG_UINT8:
            for (size_t _flat = 0; _flat < a->x * a->y * a->z; _flat++) {
            size_t _xi = (a->y * a->z > 0) ? _flat / (a->y * a->z) : 0;
            size_t _yj = (a->z > 0) ? (_flat / a->z) % a->y : 0;
            size_t _zk = _flat % (a->z > 0 ? a->z : 1);
                uint8_t *pa = (uint8_t *)dynamic_array_get_ptr(a, _xi, _yj, _zk);
                uint8_t *pb = (uint8_t *)dynamic_array_get_ptr(b, _xi, _yj, _zk);
                if (!pa || !pb || *pa != *pb) FAIL("array item mismatch");
            }
            break;
        case TAG_INT8:
            for (size_t _flat = 0; _flat < a->x * a->y * a->z; _flat++) {
            size_t _xi = (a->y * a->z > 0) ? _flat / (a->y * a->z) : 0;
            size_t _yj = (a->z > 0) ? (_flat / a->z) % a->y : 0;
            size_t _zk = _flat % (a->z > 0 ? a->z : 1);
                int8_t *pa = (int8_t *)dynamic_array_get_ptr(a, _xi, _yj, _zk);
                int8_t *pb = (int8_t *)dynamic_array_get_ptr(b, _xi, _yj, _zk);
                if (!pa || !pb || *pa != *pb) FAIL("array item mismatch");
            }
            break;
        case TAG_BOOL:
            for (size_t _flat = 0; _flat < a->x * a->y * a->z; _flat++) {
            size_t _xi = (a->y * a->z > 0) ? _flat / (a->y * a->z) : 0;
            size_t _yj = (a->z > 0) ? (_flat / a->z) % a->y : 0;
            size_t _zk = _flat % (a->z > 0 ? a->z : 1);
                uint8_t *pa = (uint8_t *)dynamic_array_get_ptr(a, _xi, _yj, _zk);
                uint8_t *pb = (uint8_t *)dynamic_array_get_ptr(b, _xi, _yj, _zk);
                if (!pa || !pb || *pa != *pb) FAIL("array item mismatch");
            }
            break;
        case TAG_UINT16:
            for (size_t _flat = 0; _flat < a->x * a->y * a->z; _flat++) {
            size_t _xi = (a->y * a->z > 0) ? _flat / (a->y * a->z) : 0;
            size_t _yj = (a->z > 0) ? (_flat / a->z) % a->y : 0;
            size_t _zk = _flat % (a->z > 0 ? a->z : 1);
                uint16_t *pa = (uint16_t *)dynamic_array_get_ptr(a, _xi, _yj, _zk);
                uint16_t *pb = (uint16_t *)dynamic_array_get_ptr(b, _xi, _yj, _zk);
                if (!pa || !pb || *pa != *pb) FAIL("array item mismatch");
            }
            break;
        case TAG_INT16:
            for (size_t _flat = 0; _flat < a->x * a->y * a->z; _flat++) {
            size_t _xi = (a->y * a->z > 0) ? _flat / (a->y * a->z) : 0;
            size_t _yj = (a->z > 0) ? (_flat / a->z) % a->y : 0;
            size_t _zk = _flat % (a->z > 0 ? a->z : 1);
                int16_t *pa = (int16_t *)dynamic_array_get_ptr(a, _xi, _yj, _zk);
                int16_t *pb = (int16_t *)dynamic_array_get_ptr(b, _xi, _yj, _zk);
                if (!pa || !pb || *pa != *pb) FAIL("array item mismatch");
            }
            break;
        case TAG_UINT32:
            for (size_t _flat = 0; _flat < a->x * a->y * a->z; _flat++) {
            size_t _xi = (a->y * a->z > 0) ? _flat / (a->y * a->z) : 0;
            size_t _yj = (a->z > 0) ? (_flat / a->z) % a->y : 0;
            size_t _zk = _flat % (a->z > 0 ? a->z : 1);
                uint32_t *pa = (uint32_t *)dynamic_array_get_ptr(a, _xi, _yj, _zk);
                uint32_t *pb = (uint32_t *)dynamic_array_get_ptr(b, _xi, _yj, _zk);
                if (!pa || !pb || *pa != *pb) FAIL("array item mismatch");
            }
            break;
        case TAG_INT32:
            for (size_t _flat = 0; _flat < a->x * a->y * a->z; _flat++) {
            size_t _xi = (a->y * a->z > 0) ? _flat / (a->y * a->z) : 0;
            size_t _yj = (a->z > 0) ? (_flat / a->z) % a->y : 0;
            size_t _zk = _flat % (a->z > 0 ? a->z : 1);
                int32_t *pa = (int32_t *)dynamic_array_get_ptr(a, _xi, _yj, _zk);
                int32_t *pb = (int32_t *)dynamic_array_get_ptr(b, _xi, _yj, _zk);
                if (!pa || !pb || *pa != *pb) FAIL("array item mismatch");
            }
            break;
        case TAG_FLOAT32:
            for (size_t _flat = 0; _flat < a->x * a->y * a->z; _flat++) {
            size_t _xi = (a->y * a->z > 0) ? _flat / (a->y * a->z) : 0;
            size_t _yj = (a->z > 0) ? (_flat / a->z) % a->y : 0;
            size_t _zk = _flat % (a->z > 0 ? a->z : 1);
                float *pa = (float *)dynamic_array_get_ptr(a, _xi, _yj, _zk);
                float *pb = (float *)dynamic_array_get_ptr(b, _xi, _yj, _zk);
                if (!pa || !pb || fabs((double)(*pa) - (double)(*pb)) > 1e-6) FAIL("array item mismatch");
            }
            break;
        case TAG_UINT64:
            for (size_t _flat = 0; _flat < a->x * a->y * a->z; _flat++) {
            size_t _xi = (a->y * a->z > 0) ? _flat / (a->y * a->z) : 0;
            size_t _yj = (a->z > 0) ? (_flat / a->z) % a->y : 0;
            size_t _zk = _flat % (a->z > 0 ? a->z : 1);
                uint64_t *pa = (uint64_t *)dynamic_array_get_ptr(a, _xi, _yj, _zk);
                uint64_t *pb = (uint64_t *)dynamic_array_get_ptr(b, _xi, _yj, _zk);
                if (!pa || !pb || *pa != *pb) FAIL("array item mismatch");
            }
            break;
        case TAG_INT64:
            for (size_t _flat = 0; _flat < a->x * a->y * a->z; _flat++) {
            size_t _xi = (a->y * a->z > 0) ? _flat / (a->y * a->z) : 0;
            size_t _yj = (a->z > 0) ? (_flat / a->z) % a->y : 0;
            size_t _zk = _flat % (a->z > 0 ? a->z : 1);
                int64_t *pa = (int64_t *)dynamic_array_get_ptr(a, _xi, _yj, _zk);
                int64_t *pb = (int64_t *)dynamic_array_get_ptr(b, _xi, _yj, _zk);
                if (!pa || !pb || *pa != *pb) FAIL("array item mismatch");
            }
            break;
        case TAG_FLOAT64:
            for (size_t _flat = 0; _flat < a->x * a->y * a->z; _flat++) {
            size_t _xi = (a->y * a->z > 0) ? _flat / (a->y * a->z) : 0;
            size_t _yj = (a->z > 0) ? (_flat / a->z) % a->y : 0;
            size_t _zk = _flat % (a->z > 0 ? a->z : 1);
                double *pa = (double *)dynamic_array_get_ptr(a, _xi, _yj, _zk);
                double *pb = (double *)dynamic_array_get_ptr(b, _xi, _yj, _zk);
                if (!pa || !pb || fabs(*pa - *pb) > 1e-12) FAIL("array item mismatch");
            }
            break;
        case TAG_STRING:
            for (size_t _flat = 0; _flat < a->x * a->y * a->z; _flat++) {
            size_t _xi = (a->y * a->z > 0) ? _flat / (a->y * a->z) : 0;
            size_t _yj = (a->z > 0) ? (_flat / a->z) % a->y : 0;
            size_t _zk = _flat % (a->z > 0 ? a->z : 1);
                string_t *sa = (string_t *)dynamic_array_get_ptr(a, _xi, _yj, _zk);
                string_t *sb = (string_t *)dynamic_array_get_ptr(b, _xi, _yj, _zk);
                ASSERT_STRING(*sa, *sb);
            }
            break;
        case TAG_BYTES:
            for (size_t _flat = 0; _flat < a->x * a->y * a->z; _flat++) {
            size_t _xi = (a->y * a->z > 0) ? _flat / (a->y * a->z) : 0;
            size_t _yj = (a->z > 0) ? (_flat / a->z) % a->y : 0;
            size_t _zk = _flat % (a->z > 0 ? a->z : 1);
                byte_array_t *ba = (byte_array_t *)dynamic_array_get_ptr(a, _xi, _yj, _zk);
                byte_array_t *bb = (byte_array_t *)dynamic_array_get_ptr(b, _xi, _yj, _zk);
                ASSERT_BYTE_ARRAY(*ba, *bb);
            }
            break;
        case TAG_ONLY_SCALAR_TYPES_MSG:
            for (size_t _flat = 0; _flat < a->x * a->y * a->z; _flat++) {
            size_t _xi = (a->y * a->z > 0) ? _flat / (a->y * a->z) : 0;
            size_t _yj = (a->z > 0) ? (_flat / a->z) % a->y : 0;
            size_t _zk = _flat % (a->z > 0 ? a->z : 1);
                only_scalar_types_msg_t *sa = (only_scalar_types_msg_t *)dynamic_array_get_ptr(a, _xi, _yj, _zk);
                only_scalar_types_msg_t *sb = (only_scalar_types_msg_t *)dynamic_array_get_ptr(b, _xi, _yj, _zk);
                compare_only_scalar_types_msg_t(sa, sb);
            }
            break;
        case TAG_ONLY_VARIABLE_TYPES_MSG:
            for (size_t _flat = 0; _flat < a->x * a->y * a->z; _flat++) {
            size_t _xi = (a->y * a->z > 0) ? _flat / (a->y * a->z) : 0;
            size_t _yj = (a->z > 0) ? (_flat / a->z) % a->y : 0;
            size_t _zk = _flat % (a->z > 0 ? a->z : 1);
                only_variable_types_msg_t *va =
                    (only_variable_types_msg_t *)dynamic_array_get_ptr(a, _xi, _yj, _zk);
                only_variable_types_msg_t *vb =
                    (only_variable_types_msg_t *)dynamic_array_get_ptr(b, _xi, _yj, _zk);
                if (!va || !vb) FAIL("nested array item is NULL");
                ASSERT_EQ_INT(va->name.len, vb->name.len);
                ASSERT_EQ_INT(va->data.len, vb->data.len);
            }
            break;
        default:
            FAIL("unsupported dynamic array type");
    }
}

/* ---------------------------------------------------------------------------
 * Helper: compare_all_types_fields_msg_t
 * --------------------------------------------------------------------------*/
static void compare_all_types_fields_msg_t(const all_types_fields_msg_t *a,
                                            const all_types_fields_msg_t *b)
{
    ASSERT_EQ_INT(a->val_uint64, b->val_uint64);
    ASSERT_EQ_INT(a->val_int64,  b->val_int64);
    ASSERT_EQ_FLT(a->val_double, b->val_double);
    ASSERT_EQ_INT(a->val_uint32, b->val_uint32);
    ASSERT_EQ_INT(a->val_int32,  b->val_int32);
    ASSERT_EQ_FLT(a->val_float,  b->val_float);
    ASSERT_EQ_INT(a->val_uint16, b->val_uint16);
    ASSERT_EQ_INT(a->val_int16,  b->val_int16);
    ASSERT_EQ_INT(a->val_uint8,  b->val_uint8);
    ASSERT_EQ_INT(a->val_int8,   b->val_int8);
    ASSERT_EQ_INT(a->val_bool,   b->val_bool);
    ASSERT_EQ_INT(a->name.len,   b->name.len);
    if (a->name.len > 0) ASSERT_EQ_STR(a->name.data, b->name.data);
    ASSERT_EQ_INT(a->data.len,   b->data.len);
    if (a->data.len > 0) ASSERT_MEM_EQ(a->data.data, b->data.data, a->data.len);
    if (a->nested == NULL) {
        ASSERT(b->nested == NULL);
    } else {
        ASSERT(b->nested != NULL);
        compare_only_scalar_types_msg_t(a->nested, b->nested);
    }
    dynamic_array_compare(TAG_STRING, &a->tags,       &b->tags);
    dynamic_array_compare(TAG_INT32,  &a->matrix,     &b->matrix);
    dynamic_array_compare(TAG_BYTES,  &a->byte_array, &b->byte_array);
}

/* Match Go test:
 *   AllTypesFieldsMsg{ ValUint64:0x0102030405060708, ValInt64:-1234567890123456789,
 *     ValDouble:Pi, ValUint32:0xABCDEF01, ValInt32:-1000000, ValFloat:1.234,
 *     Name:"hello world", Data:{DE,AD,BE,EF},
 *     Nested:{ValUint64:42}, Tags:["alpha","beta","gamma","xyz"],
 *     Matrix:[[1..9]], ByteArray:[[01,02],[03,04,05]],
 *     ValUint16:0x1234, ValInt16:-100, ValUint8:200, ValInt8:-50, ValBool:true }
 * Returns expected dynamic payload size (183).
 */
static int initialize_all_types_fields_msg_t(all_types_fields_msg_t *msg)
{
    memset(msg, 0, sizeof(*msg));

    all_types_fields_msg_t_set_val_uint64(msg, 0x0102030405060708ULL);
    all_types_fields_msg_t_set_val_int64(msg,  -1234567890123456789LL);
    all_types_fields_msg_t_set_val_double(msg, 3.141592653589793);
    all_types_fields_msg_t_set_val_uint32(msg, 0xABCDEF01U);
    all_types_fields_msg_t_set_val_int32(msg,  -1000000);
    all_types_fields_msg_t_set_val_float(msg,  1.234f);
    all_types_fields_msg_t_set_name(msg, "hello world", 11);

    uint8_t raw[] = {0xDE, 0xAD, 0xBE, 0xEF};
    all_types_fields_msg_t_set_data(msg, raw, 4);

    only_scalar_types_msg_t inner = {0};
    only_scalar_types_msg_t_set_val_uint64(&inner, 42ULL);
    all_types_fields_msg_t_set_nested(msg, &inner);

    {
        dynamic_array_t tags = {0};
        dynamic_array_init(&tags, TAG_STRING, sizeof(string_t), 1, 4, 1, 1);
        string_t s0={.data=(char*)"alpha",.len=5}, s1={.data=(char*)"beta", .len=4};
        string_t s2={.data=(char*)"gamma",.len=5}, s3={.data=(char*)"xyz",  .len=3};
        ASSERT(dynamic_array_set(&tags, 0, 0, 0, &s0) == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&tags, 1, 0, 0, &s1) == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&tags, 2, 0, 0, &s2) == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&tags, 3, 0, 0, &s3) == DYN_ARR_OK);
        all_types_fields_msg_t_set_tags(msg, &tags);
        dynamic_array_destroy(&tags);
    }
    {
        dynamic_array_t matrix = {0};
        dynamic_array_init(&matrix, TAG_INT32, sizeof(int32_t), 2, 3, 3, 1);
        int32_t v[9] = {1,2,3,4,5,6,7,8,9};
        for (size_t r = 0; r < 3; r++)
            for (size_t c = 0; c < 3; c++)
                ASSERT(dynamic_array_set(&matrix, r, c, 0, &v[r*3+c]) == DYN_ARR_OK);
        all_types_fields_msg_t_set_matrix(msg, &matrix);
        dynamic_array_destroy(&matrix);
    }
    {
        dynamic_array_t ba = {0};
        dynamic_array_init(&ba, TAG_BYTES, sizeof(byte_array_t), 1, 2, 1, 1);
        static uint8_t d0[]={0x01,0x02}, d1[]={0x03,0x04,0x05};
        byte_array_t ba0={.data=d0,.len=2}, ba1={.data=d1,.len=3};
        ASSERT(dynamic_array_set(&ba, 0, 0, 0, &ba0) == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&ba, 1, 0, 0, &ba1) == DYN_ARR_OK);
        all_types_fields_msg_t_set_byte_array(msg, &ba);
        dynamic_array_destroy(&ba);
    }

    all_types_fields_msg_t_set_val_uint16(msg, 0x1234);
    all_types_fields_msg_t_set_val_int16(msg,  -100);
    all_types_fields_msg_t_set_val_uint8(msg,  200);
    all_types_fields_msg_t_set_val_int8(msg,   (int8_t)-50);
    all_types_fields_msg_t_set_val_bool(msg,   1);

    return 183; /* verified against Go roundtrip_test.go */
}

/* ---------------------------------------------------------------------------
 * Helper: compare_recursive_nested_msg_t  (recursive)
 * --------------------------------------------------------------------------*/
static void compare_recursive_nested_msg_t(const recursive_nested_msg_t *a,
                                             const recursive_nested_msg_t *b)
{
    if (a == NULL && b == NULL) return;
    if (a == NULL || b == NULL) FAIL("recursive_nested: one side is NULL");
    ASSERT_EQ_INT(a->name.len, b->name.len);
    if (a->name.len > 0) ASSERT_EQ_STR(a->name.data, b->name.data);
    compare_recursive_nested_msg_t(a->nested, b->nested);
}

/* Match Go test:
 *   RecursiveNestedMsg{ Name:"level1",
 *     Nested:{ Name:"level2", Nested:{ Name:"level3" }}}
 *
 * MEMORY NOTE: set_nested does a shallow copy, so level2/level3 local structs
 * share name.data pointers with msg. Only call recursive_nested_msg_t_free(msg)
 * — do NOT separately free the intermediate level2/level3 vars.
 * Returns expected dynamic size (50).
 */
static int initialize_recursive_nested_msg_t(recursive_nested_msg_t *msg)
{
    memset(msg, 0, sizeof(*msg));
    recursive_nested_msg_t_set_name(msg, "level1", 6);

    recursive_nested_msg_t level3 = {0};
    recursive_nested_msg_t_set_name(&level3, "level3", 6);

    recursive_nested_msg_t level2 = {0};
    recursive_nested_msg_t_set_name(&level2, "level2", 6);
    recursive_nested_msg_t_set_nested(&level2, &level3);

    recursive_nested_msg_t_set_nested(msg, &level2);
    /* dyn = len("level1")=6 + Size(level2)=(8+8+6+Size(level3))
     *     Size(level3) = 8+8+6 = 22
     *     Size(level2) = 8+8+28 = 44
     *     dyn = 6 + 44 = 50 */
    return 50;
}

/* ---------------------------------------------------------------------------
 * Helper: compare_all_types_of_arrays_msg_t
 * --------------------------------------------------------------------------*/
static void compare_all_types_of_arrays_msg_t(const all_types_of_arrays_msg_t *a,
                                               const all_types_of_arrays_msg_t *b)
{
    /* 1D */
    dynamic_array_compare(TAG_BOOL,                    &a->arr1_d_bool,   &b->arr1_d_bool);
    dynamic_array_compare(TAG_BYTES,                   &a->arr1_d_bytes,  &b->arr1_d_bytes);
    dynamic_array_compare(TAG_FLOAT64,                 &a->arr1_d_double, &b->arr1_d_double);
    dynamic_array_compare(TAG_FLOAT32,                 &a->arr1_d_float,  &b->arr1_d_float);
    dynamic_array_compare(TAG_INT16,                   &a->arr1_d_int16,  &b->arr1_d_int16);
    dynamic_array_compare(TAG_INT32,                   &a->arr1_d_int32,  &b->arr1_d_int32);
    dynamic_array_compare(TAG_INT64,                   &a->arr1_d_int64,  &b->arr1_d_int64);
    dynamic_array_compare(TAG_INT8,                    &a->arr1_d_int8,   &b->arr1_d_int8);
    dynamic_array_compare(TAG_ONLY_VARIABLE_TYPES_MSG, &a->arr1_d_nested, &b->arr1_d_nested);
    dynamic_array_compare(TAG_ONLY_SCALAR_TYPES_MSG,   &a->arr1_d_object, &b->arr1_d_object);
    dynamic_array_compare(TAG_STRING,                  &a->arr1_d_string, &b->arr1_d_string);
    dynamic_array_compare(TAG_UINT16,                  &a->arr1_d_uint16, &b->arr1_d_uint16);
    dynamic_array_compare(TAG_UINT32,                  &a->arr1_d_uint32, &b->arr1_d_uint32);
    dynamic_array_compare(TAG_UINT64,                  &a->arr1_d_uint64, &b->arr1_d_uint64);
    dynamic_array_compare(TAG_UINT8,                   &a->arr1_d_uint8,  &b->arr1_d_uint8);
    /* 2D */
    dynamic_array_compare(TAG_BOOL,                    &a->arr2_d_bool,   &b->arr2_d_bool);
    dynamic_array_compare(TAG_BYTES,                   &a->arr2_d_bytes,  &b->arr2_d_bytes);
    dynamic_array_compare(TAG_FLOAT64,                 &a->arr2_d_double, &b->arr2_d_double);
    dynamic_array_compare(TAG_FLOAT32,                 &a->arr2_d_float,  &b->arr2_d_float);
    dynamic_array_compare(TAG_INT16,                   &a->arr2_d_int16,  &b->arr2_d_int16);
    dynamic_array_compare(TAG_INT32,                   &a->arr2_d_int32,  &b->arr2_d_int32);
    dynamic_array_compare(TAG_INT64,                   &a->arr2_d_int64,  &b->arr2_d_int64);
    dynamic_array_compare(TAG_INT8,                    &a->arr2_d_int8,   &b->arr2_d_int8);
    dynamic_array_compare(TAG_ONLY_VARIABLE_TYPES_MSG, &a->arr2_d_nested, &b->arr2_d_nested);
    dynamic_array_compare(TAG_ONLY_SCALAR_TYPES_MSG,   &a->arr2_d_object, &b->arr2_d_object);
    dynamic_array_compare(TAG_STRING,                  &a->arr2_d_string, &b->arr2_d_string);
    dynamic_array_compare(TAG_UINT16,                  &a->arr2_d_uint16, &b->arr2_d_uint16);
    dynamic_array_compare(TAG_UINT32,                  &a->arr2_d_uint32, &b->arr2_d_uint32);
    dynamic_array_compare(TAG_UINT64,                  &a->arr2_d_uint64, &b->arr2_d_uint64);
    dynamic_array_compare(TAG_UINT8,                   &a->arr2_d_uint8,  &b->arr2_d_uint8);
    /* 3D */
    dynamic_array_compare(TAG_BOOL,                    &a->arr3_d_bool,   &b->arr3_d_bool);
    dynamic_array_compare(TAG_BYTES,                   &a->arr3_d_bytes,  &b->arr3_d_bytes);
    dynamic_array_compare(TAG_FLOAT64,                 &a->arr3_d_double, &b->arr3_d_double);
    dynamic_array_compare(TAG_FLOAT32,                 &a->arr3_d_float,  &b->arr3_d_float);
    dynamic_array_compare(TAG_INT16,                   &a->arr3_d_int16,  &b->arr3_d_int16);
    dynamic_array_compare(TAG_INT32,                   &a->arr3_d_int32,  &b->arr3_d_int32);
    dynamic_array_compare(TAG_INT64,                   &a->arr3_d_int64,  &b->arr3_d_int64);
    dynamic_array_compare(TAG_INT8,                    &a->arr3_d_int8,   &b->arr3_d_int8);
    dynamic_array_compare(TAG_ONLY_VARIABLE_TYPES_MSG, &a->arr3_d_nested, &b->arr3_d_nested);
    dynamic_array_compare(TAG_ONLY_SCALAR_TYPES_MSG,   &a->arr3_d_object, &b->arr3_d_object);
    dynamic_array_compare(TAG_STRING,                  &a->arr3_d_string, &b->arr3_d_string);
    dynamic_array_compare(TAG_UINT16,                  &a->arr3_d_uint16, &b->arr3_d_uint16);
    dynamic_array_compare(TAG_UINT32,                  &a->arr3_d_uint32, &b->arr3_d_uint32);
    dynamic_array_compare(TAG_UINT64,                  &a->arr3_d_uint64, &b->arr3_d_uint64);
    dynamic_array_compare(TAG_UINT8,                   &a->arr3_d_uint8,  &b->arr3_d_uint8);
}

/* ---------------------------------------------------------------------------
 * Helper: initialize_all_types_of_arrays_msg_t
 *
 * Uses the same values as Go roundtrip_test.go for all numeric types.
 * 2D/3D string arrays use uniform inner sizes (not jagged) because
 * dynamic_array_t requires a fixed layout. nested arrays use zero-init
 * OnlyVariableTypesMsg to keep free() semantics ASAN-clean.
 *
 * Returns expected dynamic payload size (1844).
 * --------------------------------------------------------------------------*/
static int initialize_all_types_of_arrays_msg_t(all_types_of_arrays_msg_t *msg)
{
    memset(msg, 0, sizeof(*msg));

    /* ===== 1D arrays ===== */

    /* arr1_d_bool: {true, false, true} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_BOOL, sizeof(uint8_t), 1, 3, 1, 1);
        ASSERT(dynamic_array_set(&a, 0,0,0, &(uint8_t){1}) == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 1,0,0, &(uint8_t){0}) == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 2,0,0, &(uint8_t){1}) == DYN_ARR_OK);
        all_types_of_arrays_msg_t_set_arr1_d_bool(msg, &a);
        dynamic_array_destroy(&a);
    }
    /* arr1_d_uint8: {0, 128, 255} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_UINT8, sizeof(uint8_t), 1, 3, 1, 1);
        ASSERT(dynamic_array_set(&a, 0,0,0, &(uint8_t){0})   == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 1,0,0, &(uint8_t){128}) == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 2,0,0, &(uint8_t){255}) == DYN_ARR_OK);
        all_types_of_arrays_msg_t_set_arr1_d_uint8(msg, &a);
        dynamic_array_destroy(&a);
    }
    /* arr1_d_int8: {-128, 0, 127} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_INT8, sizeof(int8_t), 1, 3, 1, 1);
        ASSERT(dynamic_array_set(&a, 0,0,0, &(int8_t){-128}) == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 1,0,0, &(int8_t){0})    == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 2,0,0, &(int8_t){127})  == DYN_ARR_OK);
        all_types_of_arrays_msg_t_set_arr1_d_int8(msg, &a);
        dynamic_array_destroy(&a);
    }
    /* arr1_d_uint16: {0, 1000, 65535} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_UINT16, sizeof(uint16_t), 1, 3, 1, 1);
        ASSERT(dynamic_array_set(&a, 0,0,0, &(uint16_t){0})     == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 1,0,0, &(uint16_t){1000})  == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 2,0,0, &(uint16_t){65535}) == DYN_ARR_OK);
        all_types_of_arrays_msg_t_set_arr1_d_uint16(msg, &a);
        dynamic_array_destroy(&a);
    }
    /* arr1_d_int16: {-32768, 0, 32767} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_INT16, sizeof(int16_t), 1, 3, 1, 1);
        ASSERT(dynamic_array_set(&a, 0,0,0, &(int16_t){-32768}) == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 1,0,0, &(int16_t){0})      == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 2,0,0, &(int16_t){32767})  == DYN_ARR_OK);
        all_types_of_arrays_msg_t_set_arr1_d_int16(msg, &a);
        dynamic_array_destroy(&a);
    }
    /* arr1_d_uint32: {0, 1, 4294967295} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_UINT32, sizeof(uint32_t), 1, 3, 1, 1);
        ASSERT(dynamic_array_set(&a, 0,0,0, &(uint32_t){0})           == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 1,0,0, &(uint32_t){1})           == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 2,0,0, &(uint32_t){4294967295U}) == DYN_ARR_OK);
        all_types_of_arrays_msg_t_set_arr1_d_uint32(msg, &a);
        dynamic_array_destroy(&a);
    }
    /* arr1_d_int32: {-2147483648, 0, 2147483647} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_INT32, sizeof(int32_t), 1, 3, 1, 1);
        ASSERT(dynamic_array_set(&a, 0,0,0, &(int32_t){-2147483648}) == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 1,0,0, &(int32_t){0})           == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 2,0,0, &(int32_t){2147483647})  == DYN_ARR_OK);
        all_types_of_arrays_msg_t_set_arr1_d_int32(msg, &a);
        dynamic_array_destroy(&a);
    }
    /* arr1_d_float: {-1.5, 0.0, 1.5} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_FLOAT32, sizeof(float), 1, 3, 1, 1);
        ASSERT(dynamic_array_set(&a, 0,0,0, &(float){-1.5f}) == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 1,0,0, &(float){0.0f})  == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 2,0,0, &(float){1.5f})  == DYN_ARR_OK);
        all_types_of_arrays_msg_t_set_arr1_d_float(msg, &a);
        dynamic_array_destroy(&a);
    }
    /* arr1_d_uint64: {0, 1, UINT64_MAX} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_UINT64, sizeof(uint64_t), 1, 3, 1, 1);
        ASSERT(dynamic_array_set(&a, 0,0,0, &(uint64_t){0})                      == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 1,0,0, &(uint64_t){1})                      == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 2,0,0, &(uint64_t){18446744073709551615ULL}) == DYN_ARR_OK);
        all_types_of_arrays_msg_t_set_arr1_d_uint64(msg, &a);
        dynamic_array_destroy(&a);
    }
    /* arr1_d_int64: {INT64_MIN, 0, INT64_MAX} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_INT64, sizeof(int64_t), 1, 3, 1, 1);
        ASSERT(dynamic_array_set(&a, 0,0,0, &(int64_t){-9223372036854775807LL-1}) == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 1,0,0, &(int64_t){0})                        == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 2,0,0, &(int64_t){9223372036854775807LL})    == DYN_ARR_OK);
        all_types_of_arrays_msg_t_set_arr1_d_int64(msg, &a);
        dynamic_array_destroy(&a);
    }
    /* arr1_d_double: {-Pi, 0.0, Pi} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_FLOAT64, sizeof(double), 1, 3, 1, 1);
        ASSERT(dynamic_array_set(&a, 0,0,0, &(double){-3.141592653589793}) == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 1,0,0, &(double){0.0})                == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 2,0,0, &(double){3.141592653589793})  == DYN_ARR_OK);
        all_types_of_arrays_msg_t_set_arr1_d_double(msg, &a);
        dynamic_array_destroy(&a);
    }

    /* arr1_d_string: {"hello", "world", "日本語"} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_STRING, sizeof(string_t), 1, 3, 1, 1);
        string_t s0 = {.data=(char*)"hello",.len=5};
        string_t s1 = {.data=(char*)"world",.len=5};
        /* 日本語 in UTF-8: E6 97 A5 E6 9C AC E8 AA 9E = 9 bytes */
        string_t s2 = {.data=(char*)"\xe6\x97\xa5\xe6\x9c\xac\xe8\xaa\x9e",.len=9};
        ASSERT(dynamic_array_set(&a, 0,0,0, &s0) == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 1,0,0, &s1) == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 2,0,0, &s2) == DYN_ARR_OK);
        all_types_of_arrays_msg_t_set_arr1_d_string(msg, &a);
        dynamic_array_destroy(&a);
    }
    /* arr1_d_bytes: {{0x01, 0x02}, {0xFF}, {0xFF}} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_BYTES, sizeof(byte_array_t), 1, 3, 1, 1);
        static uint8_t d0[]={0x01,0x02}, d2[]={0xFF};
        byte_array_t ba0={.data=d0,.len=2}, ba1={.data=d2,.len=1}, ba2={.data=d2,.len=1};
        ASSERT(dynamic_array_set(&a, 0,0,0, &ba0) == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 1,0,0, &ba1) == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 2,0,0, &ba2) == DYN_ARR_OK);
        all_types_of_arrays_msg_t_set_arr1_d_bytes(msg, &a);
        dynamic_array_destroy(&a);
    }
    /* arr1_d_nested: {{Name: "something"}, {Name: "something"}, {Name: "something"}} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_ONLY_VARIABLE_TYPES_MSG,
                           sizeof(only_variable_types_msg_t), 1, 3, 1, 1);
        only_variable_types_msg_t z1 = {0};
        only_variable_types_msg_t_set_name(&z1, "something", 9);
        ASSERT(dynamic_array_set(&a, 0,0,0, &z1) == DYN_ARR_OK);
        only_variable_types_msg_t z2 = {0};
        only_variable_types_msg_t_set_name(&z2, "something", 9);
        ASSERT(dynamic_array_set(&a, 1,0,0, &z2) == DYN_ARR_OK);
        only_variable_types_msg_t z3 = {0};
        only_variable_types_msg_t_set_name(&z3, "something", 9);
        ASSERT(dynamic_array_set(&a, 2,0,0, &z3) == DYN_ARR_OK);
        all_types_of_arrays_msg_t_set_arr1_d_nested(msg, &a);
        dynamic_array_destroy(&a);
    }
    /* arr1_d_object: {{ValUint64: 100}, {ValUint64: 100}, {ValUint64: 100}} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_ONLY_SCALAR_TYPES_MSG,
                           sizeof(only_scalar_types_msg_t), 1, 3, 1, 1);
        only_scalar_types_msg_t o = {0};
        only_scalar_types_msg_t_set_val_uint64(&o, 100ULL);
        ASSERT(dynamic_array_set(&a, 0,0,0, &o) == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 1,0,0, &o) == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 2,0,0, &o) == DYN_ARR_OK);
        all_types_of_arrays_msg_t_set_arr1_d_object(msg, &a);
        dynamic_array_destroy(&a);
    }

    /* ===== 2D arrays ===== */

    /* arr2_d_bool: {{true,false},{false,true}} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_BOOL, sizeof(uint8_t), 2, 2, 2, 1);
        ASSERT(dynamic_array_set(&a, 0,0,0, &(uint8_t){1}) == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 0,1,0, &(uint8_t){0}) == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 1,0,0, &(uint8_t){0}) == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 1,1,0, &(uint8_t){1}) == DYN_ARR_OK);
        all_types_of_arrays_msg_t_set_arr2_d_bool(msg, &a);
        dynamic_array_destroy(&a);
    }
    /* arr2_d_uint8: {{0,128},{255,64}} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_UINT8, sizeof(uint8_t), 2, 2, 2, 1);
        ASSERT(dynamic_array_set(&a, 0,0,0, &(uint8_t){0})   == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 0,1,0, &(uint8_t){128}) == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 1,0,0, &(uint8_t){255}) == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 1,1,0, &(uint8_t){64})  == DYN_ARR_OK);
        all_types_of_arrays_msg_t_set_arr2_d_uint8(msg, &a);
        dynamic_array_destroy(&a);
    }
    /* arr2_d_int8: {{-128,0},{127,-64}} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_INT8, sizeof(int8_t), 2, 2, 2, 1);
        ASSERT(dynamic_array_set(&a, 0,0,0, &(int8_t){-128}) == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 0,1,0, &(int8_t){0})    == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 1,0,0, &(int8_t){127})  == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 1,1,0, &(int8_t){-64})  == DYN_ARR_OK);
        all_types_of_arrays_msg_t_set_arr2_d_int8(msg, &a);
        dynamic_array_destroy(&a);
    }
    /* arr2_d_uint16: {{0,1000},{65535,500}} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_UINT16, sizeof(uint16_t), 2, 2, 2, 1);
        ASSERT(dynamic_array_set(&a, 0,0,0, &(uint16_t){0})     == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 0,1,0, &(uint16_t){1000})  == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 1,0,0, &(uint16_t){65535}) == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 1,1,0, &(uint16_t){500})   == DYN_ARR_OK);
        all_types_of_arrays_msg_t_set_arr2_d_uint16(msg, &a);
        dynamic_array_destroy(&a);
    }
    /* arr2_d_int16: {{-32768,0},{32767,-500}} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_INT16, sizeof(int16_t), 2, 2, 2, 1);
        ASSERT(dynamic_array_set(&a, 0,0,0, &(int16_t){-32768}) == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 0,1,0, &(int16_t){0})      == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 1,0,0, &(int16_t){32767})  == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 1,1,0, &(int16_t){-500})   == DYN_ARR_OK);
        all_types_of_arrays_msg_t_set_arr2_d_int16(msg, &a);
        dynamic_array_destroy(&a);
    }
    /* arr2_d_uint32: {{0,1},{4294967295,100}} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_UINT32, sizeof(uint32_t), 2, 2, 2, 1);
        ASSERT(dynamic_array_set(&a, 0,0,0, &(uint32_t){0})           == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 0,1,0, &(uint32_t){1})           == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 1,0,0, &(uint32_t){4294967295U}) == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 1,1,0, &(uint32_t){100})         == DYN_ARR_OK);
        all_types_of_arrays_msg_t_set_arr2_d_uint32(msg, &a);
        dynamic_array_destroy(&a);
    }
    /* arr2_d_int32: {{-2147483648,0},{2147483647,-100}} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_INT32, sizeof(int32_t), 2, 2, 2, 1);
        ASSERT(dynamic_array_set(&a, 0,0,0, &(int32_t){-2147483648}) == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 0,1,0, &(int32_t){0})           == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 1,0,0, &(int32_t){2147483647})  == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 1,1,0, &(int32_t){-100})        == DYN_ARR_OK);
        all_types_of_arrays_msg_t_set_arr2_d_int32(msg, &a);
        dynamic_array_destroy(&a);
    }
    /* arr2_d_uint64: {{0,1},{UINT64_MAX,1000}} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_UINT64, sizeof(uint64_t), 2, 2, 2, 1);
        ASSERT(dynamic_array_set(&a, 0,0,0, &(uint64_t){0})                      == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 0,1,0, &(uint64_t){1})                      == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 1,0,0, &(uint64_t){18446744073709551615ULL}) == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 1,1,0, &(uint64_t){1000})                   == DYN_ARR_OK);
        all_types_of_arrays_msg_t_set_arr2_d_uint64(msg, &a);
        dynamic_array_destroy(&a);
    }
    /* arr2_d_int64: {{INT64_MIN,0},{INT64_MAX,-1000}} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_INT64, sizeof(int64_t), 2, 2, 2, 1);
        ASSERT(dynamic_array_set(&a, 0,0,0, &(int64_t){-9223372036854775807LL-1}) == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 0,1,0, &(int64_t){0})                        == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 1,0,0, &(int64_t){9223372036854775807LL})    == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 1,1,0, &(int64_t){-1000})                    == DYN_ARR_OK);
        all_types_of_arrays_msg_t_set_arr2_d_int64(msg, &a);
        dynamic_array_destroy(&a);
    }
    /* arr2_d_float: {{-1.5,0.0},{1.5,-2.5}} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_FLOAT32, sizeof(float), 2, 2, 2, 1);
        ASSERT(dynamic_array_set(&a, 0,0,0, &(float){-1.5f}) == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 0,1,0, &(float){0.0f})  == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 1,0,0, &(float){1.5f})  == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 1,1,0, &(float){-2.5f}) == DYN_ARR_OK);
        all_types_of_arrays_msg_t_set_arr2_d_float(msg, &a);
        dynamic_array_destroy(&a);
    }
    /* arr2_d_double: {{-Pi,0.0},{Pi,-E}} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_FLOAT64, sizeof(double), 2, 2, 2, 1);
        ASSERT(dynamic_array_set(&a, 0,0,0, &(double){-3.141592653589793}) == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 0,1,0, &(double){0.0})                == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 1,0,0, &(double){3.141592653589793})  == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 1,1,0, &(double){-2.718281828459045}) == DYN_ARR_OK);
        all_types_of_arrays_msg_t_set_arr2_d_double(msg, &a);
        dynamic_array_destroy(&a);
    }

    /* arr2_d_string: {{"a", "b"}, {"c", "d"}} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_STRING, sizeof(string_t), 2, 2, 2, 1);
        string_t sa={.data=(char*)"a",.len=1}, sb={.data=(char*)"b",.len=1};
        string_t sc={.data=(char*)"c",.len=1}, sd={.data=(char*)"d",.len=1};
        ASSERT(dynamic_array_set(&a, 0,0,0, &sa) == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 0,1,0, &sb) == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 1,0,0, &sc) == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 1,1,0, &sd) == DYN_ARR_OK);
        all_types_of_arrays_msg_t_set_arr2_d_string(msg, &a);
        dynamic_array_destroy(&a);
    }
    /* arr2_d_bytes: {{{0x01},{0x02}},{{0x03},{0x04,0x05}}} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_BYTES, sizeof(byte_array_t), 2, 2, 2, 1);
        static uint8_t d01[]={0x01}, d02[]={0x02}, d03[]={0x03}, d04[]={0x04,0x05};
        byte_array_t ba01={.data=d01,.len=1}, ba02={.data=d02,.len=1};
        byte_array_t ba03={.data=d03,.len=1}, ba0405={.data=d04,.len=2};
        ASSERT(dynamic_array_set(&a, 0,0,0, &ba01)   == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 0,1,0, &ba02)   == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 1,0,0, &ba03)   == DYN_ARR_OK);
        ASSERT(dynamic_array_set(&a, 1,1,0, &ba0405) == DYN_ARR_OK);
        all_types_of_arrays_msg_t_set_arr2_d_bytes(msg, &a);
        dynamic_array_destroy(&a);
    }
    /* arr2_d_nested: [][]*OnlyVariableTypesMsg{{{Name: "nested1"}, {Name: "nested2"}}, {{Name: "nested3"}, {Name: "nested4"}}} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_ONLY_VARIABLE_TYPES_MSG, sizeof(only_variable_types_msg_t), 2, 2, 2, 1);
        only_variable_types_msg_t z1 = {0}, z2 = {0}, z3 = {0}, z4 = {0};
        only_variable_types_msg_t_set_name(&z1, "nested1", 7);
        ASSERT(dynamic_array_set(&a, 0,0,0, &z1) == DYN_ARR_OK);
        only_variable_types_msg_t_set_name(&z2, "nested2", 7);
        ASSERT(dynamic_array_set(&a, 0,1,0, &z2) == DYN_ARR_OK);
        only_variable_types_msg_t_set_name(&z3, "nested3", 7);
        ASSERT(dynamic_array_set(&a, 1,0,0, &z3) == DYN_ARR_OK);
        only_variable_types_msg_t_set_name(&z4, "nested4", 7);
        ASSERT(dynamic_array_set(&a, 1,1,0, &z4) == DYN_ARR_OK);
        all_types_of_arrays_msg_t_set_arr2_d_nested(msg, &a);
        dynamic_array_destroy(&a);
    }
    /* arr2_d_object: [][]*OnlyScalarTypesMsg{{{ValUint64: 1}, {ValUint64: 2}}, {{ValUint64: 3}, {ValUint64: 4}}} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_ONLY_SCALAR_TYPES_MSG,
                           sizeof(only_scalar_types_msg_t), 2, 2, 2, 1);
        only_scalar_types_msg_t o1={0}, o2={0}, o3={0}, o4={0};
        only_scalar_types_msg_t_set_val_uint64(&o1, 1ULL);
        ASSERT(dynamic_array_set(&a, 0,0,0, &o1) == DYN_ARR_OK);
        only_scalar_types_msg_t_set_val_uint64(&o2, 2ULL);
        ASSERT(dynamic_array_set(&a, 0,1,0, &o2) == DYN_ARR_OK);
        only_scalar_types_msg_t_set_val_uint64(&o3, 3ULL);
        ASSERT(dynamic_array_set(&a, 1,0,0, &o3) == DYN_ARR_OK);
        only_scalar_types_msg_t_set_val_uint64(&o4, 4ULL);
        ASSERT(dynamic_array_set(&a, 1,1,0, &o4) == DYN_ARR_OK);
        all_types_of_arrays_msg_t_set_arr2_d_object(msg, &a);
        dynamic_array_destroy(&a);
    }

    /* ===== 3D arrays ===== */

    /* Macro to iterate 2x2x2 and set values from flat array */
#define SET_3D_2x2x2(arr_ptr, vals) \
    do { \
        for (size_t _x=0;_x<2;_x++) \
            for (size_t _y=0;_y<2;_y++) \
                for (size_t _z=0;_z<2;_z++) \
                    ASSERT(dynamic_array_set((arr_ptr),_x,_y,_z, \
                        &(vals)[_x*4+_y*2+_z]) == DYN_ARR_OK); \
    } while(0)

    /* arr3_d_bool: {{{true,false},{false,true}},{{true,true},{false,false}}} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_BOOL, sizeof(uint8_t), 3, 2, 2, 2);
        uint8_t v[8]={1,0,0,1,1,1,0,0};
        SET_3D_2x2x2(&a, v);
        all_types_of_arrays_msg_t_set_arr3_d_bool(msg, &a);
        dynamic_array_destroy(&a);
    }
    /* arr3_d_uint8: {{{0,128},{255,64}},{{1,2},{3,4}}} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_UINT8, sizeof(uint8_t), 3, 2, 2, 2);
        uint8_t v[8]={0,128,255,64,1,2,3,4};
        SET_3D_2x2x2(&a, v);
        all_types_of_arrays_msg_t_set_arr3_d_uint8(msg, &a);
        dynamic_array_destroy(&a);
    }
    /* arr3_d_int8: {{{-128,0},{127,-64}},{{-1,-2},{-3,-4}}} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_INT8, sizeof(int8_t), 3, 2, 2, 2);
        int8_t v[8]={-128,0,127,-64,-1,-2,-3,-4};
        SET_3D_2x2x2(&a, v);
        all_types_of_arrays_msg_t_set_arr3_d_int8(msg, &a);
        dynamic_array_destroy(&a);
    }
    /* arr3_d_uint16: {{{0,1000},{65535,500}},{{1,2},{3,4}}} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_UINT16, sizeof(uint16_t), 3, 2, 2, 2);
        uint16_t v[8]={0,1000,65535,500,1,2,3,4};
        SET_3D_2x2x2(&a, v);
        all_types_of_arrays_msg_t_set_arr3_d_uint16(msg, &a);
        dynamic_array_destroy(&a);
    }
    /* arr3_d_int16: {{{-32768,0},{32767,-500}},{{-1,-2},{-3,-4}}} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_INT16, sizeof(int16_t), 3, 2, 2, 2);
        int16_t v[8]={-32768,0,32767,-500,-1,-2,-3,-4};
        SET_3D_2x2x2(&a, v);
        all_types_of_arrays_msg_t_set_arr3_d_int16(msg, &a);
        dynamic_array_destroy(&a);
    }
    /* arr3_d_uint32: {{{0,1},{4294967295,100}},{{1,2},{3,4}}} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_UINT32, sizeof(uint32_t), 3, 2, 2, 2);
        uint32_t v[8]={0,1,4294967295U,100,1,2,3,4};
        SET_3D_2x2x2(&a, v);
        all_types_of_arrays_msg_t_set_arr3_d_uint32(msg, &a);
        dynamic_array_destroy(&a);
    }
    /* arr3_d_int32: {{{-2147483648,0},{2147483647,-100}},{{-1,-2},{-3,-4}}} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_INT32, sizeof(int32_t), 3, 2, 2, 2);
        int32_t v[8]={-2147483648,0,2147483647,-100,-1,-2,-3,-4};
        SET_3D_2x2x2(&a, v);
        all_types_of_arrays_msg_t_set_arr3_d_int32(msg, &a);
        dynamic_array_destroy(&a);
    }
    /* arr3_d_uint64: {{{0,1},{UINT64_MAX,1000}},{{1,2},{3,4}}} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_UINT64, sizeof(uint64_t), 3, 2, 2, 2);
        uint64_t v[8]={0,1,18446744073709551615ULL,1000,1,2,3,4};
        SET_3D_2x2x2(&a, v);
        all_types_of_arrays_msg_t_set_arr3_d_uint64(msg, &a);
        dynamic_array_destroy(&a);
    }
    /* arr3_d_int64: {{{INT64_MIN,0},{INT64_MAX,-1000}},{{-1,-2},{-3,-4}}} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_INT64, sizeof(int64_t), 3, 2, 2, 2);
        int64_t v[8]={-9223372036854775807LL-1,0, 9223372036854775807LL,-1000, -1,-2,-3,-4};
        SET_3D_2x2x2(&a, v);
        all_types_of_arrays_msg_t_set_arr3_d_int64(msg, &a);
        dynamic_array_destroy(&a);
    }
    /* arr3_d_float: {{{-1.5,0.0},{1.5,-2.5}},{{-3.5,4.5},{5.5,-6.5}}} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_FLOAT32, sizeof(float), 3, 2, 2, 2);
        float v[8]={-1.5f,0.0f,1.5f,-2.5f,-3.5f,4.5f,5.5f,-6.5f};
        SET_3D_2x2x2(&a, v);
        all_types_of_arrays_msg_t_set_arr3_d_float(msg, &a);
        dynamic_array_destroy(&a);
    }
    /* arr3_d_double: {{{-Pi,0.0},{Pi,-E}},{{-1.0,2.0},{3.0,-4.0}}} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_FLOAT64, sizeof(double), 3, 2, 2, 2);
        double v[8]={-3.141592653589793,0.0, 3.141592653589793,-2.718281828459045,
                     -1.0,2.0, 3.0,-4.0};
        SET_3D_2x2x2(&a, v);
        all_types_of_arrays_msg_t_set_arr3_d_double(msg, &a);
        dynamic_array_destroy(&a);
    }

    /* arr3_d_string: {{{"a", "b"}, {"c", "d"}}, {{"e", "f"}, {"g", "h"}}} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_STRING, sizeof(string_t), 3, 2, 2, 2);
        string_t s1 = {.data=(char*)"a", .len=1};
        ASSERT(dynamic_array_set(&a,0,0,0,&s1) == DYN_ARR_OK);
        string_t s2 = {.data=(char*)"b", .len=1};
        ASSERT(dynamic_array_set(&a,0,0,1,&s2) == DYN_ARR_OK);
        string_t s3 = {.data=(char*)"c", .len=1};
        ASSERT(dynamic_array_set(&a,0,1,0,&s3) == DYN_ARR_OK);
        string_t s4 = {.data=(char*)"d", .len=1};
        ASSERT(dynamic_array_set(&a,0,1,1,&s4) == DYN_ARR_OK);
        string_t s5 = {.data=(char*)"e", .len=1};
        ASSERT(dynamic_array_set(&a,1,0,0,&s5) == DYN_ARR_OK);
        string_t s6 = {.data=(char*)"f", .len=1};
        ASSERT(dynamic_array_set(&a,1,0,1,&s6) == DYN_ARR_OK);
        string_t s7 = {.data=(char*)"g", .len=1};
        ASSERT(dynamic_array_set(&a,1,1,0,&s7) == DYN_ARR_OK);
        string_t s8 = {.data=(char*)"h", .len=1};
        ASSERT(dynamic_array_set(&a,1,1,1,&s8) == DYN_ARR_OK);

        all_types_of_arrays_msg_t_set_arr3_d_string(msg, &a);
        dynamic_array_destroy(&a);
    }
    /* arr3_d_bytes: {{{{0x01}, {0x02}}, {{0x03}, {0x04, 0x05}}}, {{{0x06}, {0x07}}, {{0x07, 0x08}, {0x09}}}} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_BYTES, sizeof(byte_array_t), 3, 2, 2, 2);
        byte_array_t ba1 = {.data=(uint8_t[]){0x01}, .len=1};
        ASSERT(dynamic_array_set(&a,0,0,0,&ba1) == DYN_ARR_OK);
        byte_array_t ba2 = {.data=(uint8_t[]){0x02}, .len=1};
        ASSERT(dynamic_array_set(&a,0,0,1,&ba2) == DYN_ARR_OK);
        byte_array_t ba3 = {.data=(uint8_t[]){0x03}, .len=1};
        ASSERT(dynamic_array_set(&a,0,1,0,&ba3) == DYN_ARR_OK);
        byte_array_t ba4 = {.data=(uint8_t[]){0x04, 0x05}, .len=2};
        ASSERT(dynamic_array_set(&a,0,1,1,&ba4) == DYN_ARR_OK);
        byte_array_t ba5 = {.data=(uint8_t[]){0x06}, .len=1};
        ASSERT(dynamic_array_set(&a,1,0,0,&ba5) == DYN_ARR_OK);
        byte_array_t ba6 = {.data=(uint8_t[]){0x07}, .len=1};
        ASSERT(dynamic_array_set(&a,1,0,1,&ba6) == DYN_ARR_OK);
        byte_array_t ba7 = {.data=(uint8_t[]){0x07, 0x08}, .len=2};
        ASSERT(dynamic_array_set(&a,1,1,0,&ba7) == DYN_ARR_OK);
        byte_array_t ba8 = {.data=(uint8_t[]){0x09}, .len=1};
        ASSERT(dynamic_array_set(&a,1,1,1,&ba8) == DYN_ARR_OK);

        all_types_of_arrays_msg_t_set_arr3_d_bytes(msg, &a);
        dynamic_array_destroy(&a);
    }
    /* arr3_d_nested: [][][]*OnlyVariableTypesMsg{{{{Name: "nested1"}, {Name: "nested2"}}, {{Name: "nested3"}, {Name: "nested4"}}},{{{Name: "nested5"}, {Name: "nested6"}}, {{Name: "nested7"}, {Name: "nested8"}}}} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_ONLY_VARIABLE_TYPES_MSG, sizeof(only_variable_types_msg_t), 3, 2, 2, 2);
        only_variable_types_msg_t z1 = {0};
        only_variable_types_msg_t_set_name(&z1, "nested1", 7);
        ASSERT(dynamic_array_set(&a, 0,0,0, &z1) == DYN_ARR_OK);
        only_variable_types_msg_t z2 = {0};
        only_variable_types_msg_t_set_name(&z2, "nested2", 7);
        ASSERT(dynamic_array_set(&a, 0,0,1, &z2) == DYN_ARR_OK);
        only_variable_types_msg_t z3 = {0};
        only_variable_types_msg_t_set_name(&z3, "nested3", 7);
        ASSERT(dynamic_array_set(&a, 0,1,0, &z3) == DYN_ARR_OK);
        only_variable_types_msg_t z4 = {0};
        only_variable_types_msg_t_set_name(&z4, "nested4", 7);
        ASSERT(dynamic_array_set(&a, 0,1,1, &z4) == DYN_ARR_OK);
        only_variable_types_msg_t z5 = {0};
        only_variable_types_msg_t_set_name(&z5, "nested5", 7);
        ASSERT(dynamic_array_set(&a, 1,0,0, &z5) == DYN_ARR_OK);
        only_variable_types_msg_t z6 = {0};
        only_variable_types_msg_t_set_name(&z6, "nested6", 7);
        ASSERT(dynamic_array_set(&a, 1,0,1, &z6) == DYN_ARR_OK);
        only_variable_types_msg_t z7 = {0};
        only_variable_types_msg_t_set_name(&z7, "nested7", 7);
        ASSERT(dynamic_array_set(&a, 1,1,0, &z7) == DYN_ARR_OK);
        only_variable_types_msg_t z8 = {0};
        only_variable_types_msg_t_set_name(&z8, "nested8", 7);
        ASSERT(dynamic_array_set(&a, 1,1,1, &z8) == DYN_ARR_OK);

        all_types_of_arrays_msg_t_set_arr3_d_nested(msg, &a);
        dynamic_array_destroy(&a);
    }
    /* arr3_d_object: [][][]*OnlyScalarTypesMsg{{{{ValUint64: 1}, {ValUint64: 2}},{{ValUint64: 3}, {ValUint64: 4}}}, {{{ValUint64: 5}, {ValUint64: 6}}{{ValUint64: 7}, {ValUint64: 8}}}} */
    {
        dynamic_array_t a = {0};
        dynamic_array_init(&a, TAG_ONLY_SCALAR_TYPES_MSG, sizeof(only_scalar_types_msg_t), 3, 2, 2, 2);
        only_scalar_types_msg_t o1={0}, o2={0}, o3={0}, o4={0}, o5={0}, o6={0}, o7={0}, o8={0};
        only_scalar_types_msg_t_set_val_uint64(&o1, 1ULL);
        ASSERT(dynamic_array_set(&a, 0,0,0, &o1) == DYN_ARR_OK);
        only_scalar_types_msg_t_set_val_uint64(&o2, 2ULL);
        ASSERT(dynamic_array_set(&a, 1,0,0, &o2) == DYN_ARR_OK);
        only_scalar_types_msg_t_set_val_uint64(&o3, 3ULL);
        ASSERT(dynamic_array_set(&a, 0,1,0, &o3) == DYN_ARR_OK);
        only_scalar_types_msg_t_set_val_uint64(&o4, 4ULL);
        ASSERT(dynamic_array_set(&a, 1,1,0, &o4) == DYN_ARR_OK);
        only_scalar_types_msg_t_set_val_uint64(&o5, 5ULL);
        ASSERT(dynamic_array_set(&a, 0,0,1, &o5) == DYN_ARR_OK);
        only_scalar_types_msg_t_set_val_uint64(&o6, 6ULL);
        ASSERT(dynamic_array_set(&a, 1,0,1, &o6) == DYN_ARR_OK);
        only_scalar_types_msg_t_set_val_uint64(&o7, 7ULL);
        ASSERT(dynamic_array_set(&a, 0,1,1, &o7) == DYN_ARR_OK);
        only_scalar_types_msg_t_set_val_uint64(&o8, 8ULL);
        ASSERT(dynamic_array_set(&a, 1,1,1, &o8) == DYN_ARR_OK);

        all_types_of_arrays_msg_t_set_arr3_d_object(msg, &a);
        dynamic_array_destroy(&a);
    }

#undef SET_3D_2x2x2
    return 2876; // From roundtrip_test.go
}

/* ---------------------------------------------------------------------------
 * Test: Wire frame header byte layout
 * --------------------------------------------------------------------------*/
static void test_wire_frame_header_big_endian(void)
{
    TEST(wire_frame_header_big_endian);

    only_scalar_types_msg_t src = {0};
    only_scalar_types_msg_t_set_val_uint64(&src, 0x0102030405060708ULL);

    uint8_t *buf = NULL;
    int total = only_scalar_types_msg_t_marshal(&src, &buf);
    ASSERT_EQ_INT(total, WIRE_FRAME_HEADER_SIZE + ONLY_SCALAR_TYPES_MSG_FIXED_SIZE);

    /* Bytes [0:2] = type ID in Big-Endian */
    ASSERT_EQ_INT(buf[0], 0x00);
    ASSERT_EQ_INT(buf[1], ONLY_SCALAR_TYPES_MSG_TYPE_ID);

    /* Bytes [2:4] = fixed payload len in Big-Endian */
    ASSERT_EQ_INT(buf[2], 0x00);
    ASSERT_EQ_INT(buf[3], ONLY_SCALAR_TYPES_MSG_FIXED_SIZE);

    /* uint64 at fixed-payload offset 0 → wire offset 8, Big-Endian */
    const uint8_t *fixed = buf + WIRE_FRAME_HEADER_SIZE;
    ASSERT_EQ_INT(fixed[0], 0x01);
    ASSERT_EQ_INT(fixed[1], 0x02);
    ASSERT_EQ_INT(fixed[7], 0x08);

    free(buf);
    PASS();
}

static void test_wire_frame_type_ids(void)
{
    TEST(wire_frame_type_ids);

    only_scalar_types_msg_t   s = {0};
    only_variable_types_msg_t v = {0};
    all_types_fields_msg_t    a = {0};
    recursive_nested_msg_t    r = {0};
    all_types_of_arrays_msg_t x = {0};

    uint8_t *b1 = NULL, *b2 = NULL, *b3 = NULL, *b4 = NULL, *b5 = NULL;
    ASSERT(only_scalar_types_msg_t_marshal(&s, &b1) > 0);
    ASSERT(only_variable_types_msg_t_marshal(&v, &b2) > 0);
    ASSERT(all_types_fields_msg_t_marshal(&a, &b3) > 0);
    ASSERT(recursive_nested_msg_t_marshal(&r, &b4) > 0);
    ASSERT(all_types_of_arrays_msg_t_marshal(&x, &b5) > 0);

    ASSERT_EQ_INT(get_message_type(b1), 1);
    ASSERT_EQ_INT(get_message_type(b2), 2);
    ASSERT_EQ_INT(get_message_type(b3), 3);
    ASSERT_EQ_INT(get_message_type(b4), 4);
    ASSERT_EQ_INT(get_message_type(b5), 5);

    free(b1); free(b2); free(b3); free(b4); free(b5);
    PASS();
}

/* ---------------------------------------------------------------------------
 * Test: OnlyScalarTypesMsg
 * --------------------------------------------------------------------------*/

static void test_only_scalar_types_msg_t_size(void)
{
    TEST(only_scalar_types_msg_t_size);

    only_scalar_types_msg_t msg = {0};
    initialize_only_scalar_types_msg_t(&msg);

    size_t total = only_scalar_types_msg_t_size(&msg);
    ASSERT_EQ_INT((int)total, WIRE_FRAME_HEADER_SIZE + ONLY_SCALAR_TYPES_MSG_FIXED_SIZE);

    only_scalar_types_msg_t_free(&msg);
    PASS();
}

static void test_only_scalar_types_msg_t_dynamic_payload_size(void)
{
    TEST(only_scalar_types_msg_t_dynamic_payload_size);

    only_scalar_types_msg_t msg = {0};
    initialize_only_scalar_types_msg_t(&msg);

    size_t dyn = only_scalar_types_msg_t_dynamic_payload_size(&msg);
    ASSERT_EQ_INT((int)dyn, 0);

    only_scalar_types_msg_t_free(&msg);
    PASS();
}

static void test_only_scalar_types_msg_t_marshal(void)
{
    TEST(only_scalar_types_msg_t_marshal);

    only_scalar_types_msg_t msg = {0};
    initialize_only_scalar_types_msg_t(&msg);

    uint8_t *buf = NULL;
    int total = only_scalar_types_msg_t_marshal(&msg, &buf);
    ASSERT(total > 0);
    ASSERT(buf != NULL);

    uint16_t type_id     = get_message_type(buf);
    uint16_t fixed_len   = get_message_fixed_payload_length(buf);
    uint32_t overall_len = get_message_overall_payload_length(buf);

    ASSERT_EQ_INT(type_id,   ONLY_SCALAR_TYPES_MSG_TYPE_ID);
    ASSERT_EQ_INT(fixed_len, ONLY_SCALAR_TYPES_MSG_FIXED_SIZE);
    ASSERT_EQ_INT((int)total, WIRE_FRAME_HEADER_SIZE + (int)overall_len);

    only_scalar_types_msg_t_free(&msg);
    free(buf);
    PASS();
}

static void test_only_scalar_types_msg_t_unmarshal(void)
{
    TEST(only_scalar_types_msg_t_unmarshal);

    only_scalar_types_msg_t src = {0};
    initialize_only_scalar_types_msg_t(&src);

    uint8_t *buf = NULL;
    int total = only_scalar_types_msg_t_marshal(&src, &buf);
    ASSERT(total > 0);
    ASSERT(buf != NULL);

    uint16_t fixed_len   = get_message_fixed_payload_length(buf);
    uint32_t overall_len = get_message_overall_payload_length(buf);

    only_scalar_types_msg_t dst = {0};
    int rc = only_scalar_types_msg_t_unmarshal(
        buf + WIRE_FRAME_HEADER_SIZE, fixed_len, overall_len, &dst);
    ASSERT_EQ_INT(rc, 0);

    compare_only_scalar_types_msg_t(&src, &dst);

    only_scalar_types_msg_t_free(&dst);
    only_scalar_types_msg_t_free(&src);
    free(buf);
    PASS();
}

static void test_only_scalar_types_msg_t_roundtrip_empty_fields(void)
{
    TEST(scalar_roundtrip_zero_values);

    only_scalar_types_msg_t src = {0};
    uint8_t *buf = NULL;
    int total = only_scalar_types_msg_t_marshal(&src, &buf);
    ASSERT(total > 0);

    only_scalar_types_msg_t dst = {0};
    uint16_t fixed_len   = get_message_fixed_payload_length(buf);
    uint32_t overall_len = get_message_overall_payload_length(buf);
    int rc = only_scalar_types_msg_t_unmarshal(
        buf + WIRE_FRAME_HEADER_SIZE, fixed_len, overall_len, &dst);
    ASSERT_EQ_INT(rc, 0);

    ASSERT_EQ_INT(dst.val_uint64, 0);
    ASSERT_EQ_INT(dst.val_int64,  0);
    ASSERT_EQ_INT(dst.val_uint32, 0);
    ASSERT_EQ_INT(dst.val_int32,  0);
    ASSERT_EQ_INT(dst.val_uint16, 0);
    ASSERT_EQ_INT(dst.val_bool,   0);

    only_scalar_types_msg_t_free(&dst);
    free(buf);
    PASS();
}

/* ---------------------------------------------------------------------------
 * Test: OnlyVariableTypesMsg
 * --------------------------------------------------------------------------*/

static void test_only_variable_types_msg_t_size(void)
{
    TEST(only_variable_types_msg_t_size);

    only_variable_types_msg_t msg = {0};
    int expected_dynamic_size = initialize_only_variable_types_msg_t(&msg);

    size_t total = only_variable_types_msg_t_size(&msg);
    ASSERT_EQ_INT((int)total, WIRE_FRAME_HEADER_SIZE + ONLY_VARIABLE_TYPES_MSG_FIXED_SIZE + expected_dynamic_size);

    only_variable_types_msg_t_free(&msg);
    PASS();
}

static void test_only_variable_types_msg_t_dynamic_payload_size(void)
{
    TEST(only_variable_types_msg_t_dynamic_payload_size);

    only_variable_types_msg_t msg = {0};
    int expected_dynamic_size = initialize_only_variable_types_msg_t(&msg);

    size_t dyn = only_variable_types_msg_t_dynamic_payload_size(&msg);
    ASSERT_EQ_INT((int)dyn, expected_dynamic_size);

    only_variable_types_msg_t_free(&msg);
    PASS();
}

static void test_only_variable_types_msg_t_marshal(void)
{
    TEST(only_variable_types_msg_t_marshal);

    only_variable_types_msg_t msg = {0};
    int expected_dynamic_size = initialize_only_variable_types_msg_t(&msg);

    uint8_t *buf = NULL;
    int total = only_variable_types_msg_t_marshal(&msg, &buf);
    ASSERT(total == WIRE_FRAME_HEADER_SIZE + ONLY_VARIABLE_TYPES_MSG_FIXED_SIZE + expected_dynamic_size);
    ASSERT(buf != NULL);

    uint16_t type_id     = get_message_type(buf);
    uint16_t fixed_len   = get_message_fixed_payload_length(buf);
    uint32_t overall_len = get_message_overall_payload_length(buf);

    ASSERT_EQ_INT(type_id,   ONLY_VARIABLE_TYPES_MSG_TYPE_ID);
    ASSERT_EQ_INT(fixed_len, ONLY_VARIABLE_TYPES_MSG_FIXED_SIZE);
    ASSERT_EQ_INT((int)overall_len, ONLY_VARIABLE_TYPES_MSG_FIXED_SIZE + expected_dynamic_size);

    only_variable_types_msg_t_free(&msg);
    free(buf);
    PASS();
}

static void test_only_variable_types_msg_t_unmarshal(void)
{
    TEST(only_variable_types_msg_t_unmarshal);

    only_variable_types_msg_t src = {0};
    int expected_dynamic_size = initialize_only_variable_types_msg_t(&src);

    uint8_t *buf = NULL;
    int total = only_variable_types_msg_t_marshal(&src, &buf);
    ASSERT(total == WIRE_FRAME_HEADER_SIZE + ONLY_VARIABLE_TYPES_MSG_FIXED_SIZE + expected_dynamic_size);
    ASSERT(buf != NULL);

    uint16_t fixed_len   = get_message_fixed_payload_length(buf);
    uint32_t overall_len = get_message_overall_payload_length(buf);

    only_variable_types_msg_t dst = {0};
    int rc = only_variable_types_msg_t_unmarshal(
        buf + WIRE_FRAME_HEADER_SIZE, fixed_len, overall_len, &dst);
    ASSERT_EQ_INT(rc, 0);

    ASSERT_EQ_INT(dst.name.len, 11);
    ASSERT_EQ_STR(dst.name.data, "hello world");
    ASSERT_EQ_INT(dst.data.len, 4);
    compare_only_scalar_types_msg_t(src.nested, dst.nested);
    
    dynamic_array_compare(TAG_STRING, &src.tags, &dst.tags);
    dynamic_array_compare(TAG_INT32, &src.matrix, &dst.matrix);
    dynamic_array_compare(TAG_BYTES, &src.byte_array, &dst.byte_array);

    only_variable_types_msg_t_free(&dst);
    only_variable_types_msg_t_free(&src);
    free(buf);
    PASS();
}

static void test_only_variable_types_msg_t_roundtrip_empty_fields(void)
{
    TEST(variable_roundtrip_empty_fields);

    only_variable_types_msg_t src = {0};
    uint8_t *buf = NULL;
    int total = only_variable_types_msg_t_marshal(&src, &buf);
    ASSERT(total > 0);

    uint16_t fixed_len   = get_message_fixed_payload_length(buf);
    uint32_t overall_len = get_message_overall_payload_length(buf);

    only_variable_types_msg_t dst = {0};
    int rc = only_variable_types_msg_t_unmarshal(
        buf + WIRE_FRAME_HEADER_SIZE, fixed_len, overall_len, &dst);
    ASSERT_EQ_INT(rc, 0);

    ASSERT_EQ_INT(dst.name.len, 0);
    ASSERT_EQ_INT(dst.data.len, 0);
    ASSERT(dst.nested == NULL);

    only_variable_types_msg_t_free(&dst);
    free(buf);
    PASS();
}

/* ---------------------------------------------------------------------------
 * Test: AllTypesFieldsMsg
 * --------------------------------------------------------------------------*/

static void test_all_types_fields_msg_t_size(void)
{
    TEST(all_types_fields_msg_t_size);

    all_types_fields_msg_t msg = {0};
    int expected_dyn = initialize_all_types_fields_msg_t(&msg);

    size_t total = all_types_fields_msg_t_size(&msg);
    ASSERT_EQ_INT((int)total, WIRE_FRAME_HEADER_SIZE + ALL_TYPES_FIELDS_MSG_FIXED_SIZE + expected_dyn);
    
    all_types_fields_msg_t_free(&msg);
    PASS();
}

static void test_all_types_fields_msg_t_dynamic_payload_size(void)
{
    TEST(all_types_fields_msg_t_dynamic_payload_size);

    all_types_fields_msg_t msg = {0};
    int expected_dyn = initialize_all_types_fields_msg_t(&msg);

    size_t dyn = all_types_fields_msg_t_dynamic_payload_size(&msg);
    ASSERT_EQ_INT((int)dyn, expected_dyn);

    all_types_fields_msg_t_free(&msg);
    PASS();
}

static void test_all_types_fields_msg_t_marshal(void)
{
    TEST(all_types_fields_msg_t_marshal);

    all_types_fields_msg_t msg = {0};
    int expected_dyn = initialize_all_types_fields_msg_t(&msg);

    uint8_t *buf = NULL;
    int total = all_types_fields_msg_t_marshal(&msg, &buf);
    ASSERT(total == WIRE_FRAME_HEADER_SIZE + ALL_TYPES_FIELDS_MSG_FIXED_SIZE + expected_dyn);
    ASSERT(buf != NULL);

    uint16_t type_id     = get_message_type(buf);
    uint16_t fixed_len   = get_message_fixed_payload_length(buf);
    uint32_t overall_len = get_message_overall_payload_length(buf);

    ASSERT_EQ_INT(type_id,          ALL_TYPES_FIELDS_MSG_TYPE_ID);
    ASSERT_EQ_INT(fixed_len,        ALL_TYPES_FIELDS_MSG_FIXED_SIZE);
    ASSERT_EQ_INT((int)overall_len, ALL_TYPES_FIELDS_MSG_FIXED_SIZE + expected_dyn);
    
    all_types_fields_msg_t_free(&msg);
    free(buf);
    PASS();
}

static void test_all_types_fields_msg_t_unmarshal(void)
{
    TEST(all_types_fields_msg_t_unmarshal);

    all_types_fields_msg_t src = {0};
    int expected_dyn = initialize_all_types_fields_msg_t(&src);
    
    uint8_t *buf = NULL;
    int total = all_types_fields_msg_t_marshal(&src, &buf);
    ASSERT(total == WIRE_FRAME_HEADER_SIZE + ALL_TYPES_FIELDS_MSG_FIXED_SIZE + expected_dyn);
    ASSERT(buf != NULL);

    uint16_t fixed_len   = get_message_fixed_payload_length(buf);
    uint32_t overall_len = get_message_overall_payload_length(buf);
    
    all_types_fields_msg_t dst = {0};
    int rc = all_types_fields_msg_t_unmarshal(
        buf + WIRE_FRAME_HEADER_SIZE, fixed_len, overall_len, &dst);
    ASSERT_EQ_INT(rc, 0);

    compare_all_types_fields_msg_t(&src, &dst);

    all_types_fields_msg_t_free(&dst);
    all_types_fields_msg_t_free(&src);
    free(buf);
    PASS();
}

static void test_all_types_fields_msg_t_roundtrip_empty_fields(void)
{
    TEST(all_types_fields_msg_t_roundtrip_empty_fields);
    
    all_types_fields_msg_t src = {0};
    uint8_t *buf = NULL;
    ASSERT(all_types_fields_msg_t_marshal(&src, &buf) > 0);
    
    uint16_t fixed_len   = get_message_fixed_payload_length(buf);
    uint32_t overall_len = get_message_overall_payload_length(buf);
    
    all_types_fields_msg_t dst = {0};
    int rc = all_types_fields_msg_t_unmarshal(
        buf + WIRE_FRAME_HEADER_SIZE, fixed_len, overall_len, &dst);
    ASSERT_EQ_INT(rc, 0);

    ASSERT_EQ_INT(dst.name.len,   0);
    ASSERT_EQ_INT(dst.data.len,   0);
    ASSERT(dst.nested == NULL);
    ASSERT_EQ_INT(dst.val_uint64, 0);
    ASSERT_EQ_INT(dst.val_bool,   0);

    all_types_fields_msg_t_free(&dst);
    free(buf);
    PASS();
}

/* ---------------------------------------------------------------------------
 * Test: RecursiveNestedMsg
 * --------------------------------------------------------------------------*/

static void test_recursive_nested_msg_t_size(void)
{
    TEST(recursive_nested_msg_t_size);

    recursive_nested_msg_t msg = {0};
    int expected_dyn = initialize_recursive_nested_msg_t(&msg);

    size_t total = recursive_nested_msg_t_size(&msg);
    ASSERT_EQ_INT((int)total, WIRE_FRAME_HEADER_SIZE + RECURSIVE_NESTED_MSG_FIXED_SIZE + expected_dyn);

    recursive_nested_msg_t_free(&msg);
    PASS();
}

static void test_recursive_nested_msg_t_dynamic_payload_size(void)
{
    TEST(recursive_nested_msg_t_dynamic_payload_size);

    recursive_nested_msg_t msg = {0};
    int expected_dyn = initialize_recursive_nested_msg_t(&msg);

    size_t dyn = recursive_nested_msg_t_dynamic_payload_size(&msg);
    ASSERT_EQ_INT((int)dyn, expected_dyn);

    recursive_nested_msg_t_free(&msg);
    PASS();
}

static void test_recursive_nested_msg_t_marshal(void)
{
    TEST(recursive_nested_msg_t_marshal);

    recursive_nested_msg_t msg = {0};
    int expected_dyn = initialize_recursive_nested_msg_t(&msg);

    uint8_t *buf = NULL;
    int total = recursive_nested_msg_t_marshal(&msg, &buf);
    ASSERT(total == WIRE_FRAME_HEADER_SIZE + RECURSIVE_NESTED_MSG_FIXED_SIZE + expected_dyn);
    ASSERT(buf != NULL);

    uint16_t type_id     = get_message_type(buf);
    uint16_t fixed_len   = get_message_fixed_payload_length(buf);
    uint32_t overall_len = get_message_overall_payload_length(buf);

    ASSERT_EQ_INT(type_id,          RECURSIVE_NESTED_MSG_TYPE_ID);
    ASSERT_EQ_INT(fixed_len,        RECURSIVE_NESTED_MSG_FIXED_SIZE);
    ASSERT_EQ_INT((int)overall_len, RECURSIVE_NESTED_MSG_FIXED_SIZE + expected_dyn);

    recursive_nested_msg_t_free(&msg);
    free(buf);
    PASS();
}

static void test_recursive_nested_msg_t_unmarshal(void)
{
    TEST(recursive_nested_msg_t_unmarshal);

    recursive_nested_msg_t src = {0};
    int expected_dyn = initialize_recursive_nested_msg_t(&src);

    uint8_t *buf = NULL;
    int total = recursive_nested_msg_t_marshal(&src, &buf);
    ASSERT(total == WIRE_FRAME_HEADER_SIZE + RECURSIVE_NESTED_MSG_FIXED_SIZE + expected_dyn);
    ASSERT(buf != NULL);

    uint16_t fixed_len   = get_message_fixed_payload_length(buf);
    uint32_t overall_len = get_message_overall_payload_length(buf);

    recursive_nested_msg_t dst = {0};
    int rc = recursive_nested_msg_t_unmarshal(
        buf + WIRE_FRAME_HEADER_SIZE, fixed_len, overall_len, &dst);
    ASSERT_EQ_INT(rc, 0);

    compare_recursive_nested_msg_t(&src, &dst);

    recursive_nested_msg_t_free(&dst);
    recursive_nested_msg_t_free(&src);
    free(buf);
    PASS();
}

static void test_recursive_nested_msg_t_roundtrip_empty_fields(void)
{
    TEST(recursive_nested_msg_t_roundtrip_single_level);

    recursive_nested_msg_t src = {0};
    recursive_nested_msg_t_set_name(&src, "leaf", 4);

    uint8_t *buf = NULL;
    int total = recursive_nested_msg_t_marshal(&src, &buf);
    ASSERT(total > 0);

    uint16_t fixed_len   = get_message_fixed_payload_length(buf);
    uint32_t overall_len = get_message_overall_payload_length(buf);

    recursive_nested_msg_t dst = {0};
    int rc = recursive_nested_msg_t_unmarshal(
        buf + WIRE_FRAME_HEADER_SIZE, fixed_len, overall_len, &dst);
    ASSERT_EQ_INT(rc, 0);

    ASSERT_EQ_INT(dst.name.len, 4);
    ASSERT_EQ_STR(dst.name.data, "leaf");
    ASSERT(dst.nested == NULL);

    recursive_nested_msg_t_free(&dst);
    recursive_nested_msg_t_free(&src);
    free(buf);
    PASS();
}

/* ---------------------------------------------------------------------------
 * Test: AllTypesOfArraysMsg
 * --------------------------------------------------------------------------*/

static void test_all_types_of_arrays_msg_t_size(void)
{
    TEST(all_types_of_arrays_msg_t_size);

    all_types_of_arrays_msg_t msg = {0};
    int expected_dyn = initialize_all_types_of_arrays_msg_t(&msg);

    size_t total = all_types_of_arrays_msg_t_size(&msg);
    ASSERT_EQ_INT((int)total, WIRE_FRAME_HEADER_SIZE + ALL_TYPES_OF_ARRAYS_MSG_FIXED_SIZE + expected_dyn);

    all_types_of_arrays_msg_t_free(&msg);
    PASS();
}

static void test_all_types_of_arrays_msg_t_dynamic_payload_size(void)
{
    TEST(all_types_of_arrays_msg_t_dynamic_payload_size);

    all_types_of_arrays_msg_t msg = {0};
    int expected_dyn = initialize_all_types_of_arrays_msg_t(&msg);

    size_t dyn = all_types_of_arrays_msg_t_dynamic_payload_size(&msg);
    ASSERT_EQ_INT((int)dyn, expected_dyn);

    all_types_of_arrays_msg_t_free(&msg);
    PASS();
}

static void test_all_types_of_arrays_msg_t_marshal(void)
{
    TEST(all_types_of_arrays_msg_t_marshal);

    all_types_of_arrays_msg_t msg = {0};
    int expected_dyn = initialize_all_types_of_arrays_msg_t(&msg);

    uint8_t *buf = NULL;
    int total = all_types_of_arrays_msg_t_marshal(&msg, &buf);
    ASSERT(total == WIRE_FRAME_HEADER_SIZE + ALL_TYPES_OF_ARRAYS_MSG_FIXED_SIZE + expected_dyn);
    ASSERT(buf != NULL);

    uint16_t type_id     = get_message_type(buf);
    uint16_t fixed_len   = get_message_fixed_payload_length(buf);
    uint32_t overall_len = get_message_overall_payload_length(buf);

    ASSERT_EQ_INT(type_id,          ALL_TYPES_OF_ARRAYS_MSG_TYPE_ID);
    ASSERT_EQ_INT(fixed_len,        ALL_TYPES_OF_ARRAYS_MSG_FIXED_SIZE);
    ASSERT_EQ_INT((int)overall_len, ALL_TYPES_OF_ARRAYS_MSG_FIXED_SIZE + expected_dyn);

    all_types_of_arrays_msg_t_free(&msg);
    free(buf);
    PASS();
}

static void test_all_types_of_arrays_msg_t_unmarshal(void)
{
    TEST(all_types_of_arrays_msg_t_unmarshal);

    all_types_of_arrays_msg_t src = {0};
    int expected_dyn = initialize_all_types_of_arrays_msg_t(&src);

    uint8_t *buf = NULL;
    int total = all_types_of_arrays_msg_t_marshal(&src, &buf);
    ASSERT(total == WIRE_FRAME_HEADER_SIZE + ALL_TYPES_OF_ARRAYS_MSG_FIXED_SIZE + expected_dyn);
    ASSERT(buf != NULL);

    uint16_t fixed_len   = get_message_fixed_payload_length(buf);
    uint32_t overall_len = get_message_overall_payload_length(buf);

    all_types_of_arrays_msg_t dst = {0};
    int rc = all_types_of_arrays_msg_t_unmarshal(
        buf + WIRE_FRAME_HEADER_SIZE, fixed_len, overall_len, &dst);
    ASSERT_EQ_INT(rc, 0);

    compare_all_types_of_arrays_msg_t(&src, &dst);

    all_types_of_arrays_msg_t_free(&dst);
    all_types_of_arrays_msg_t_free(&src);
    free(buf);
    PASS();
}

static void test_all_types_of_arrays_msg_t_roundtrip_empty_fields(void)
{
    TEST(all_types_of_arrays_msg_t_roundtrip_empty_fields);

    all_types_of_arrays_msg_t src = {0};
    uint8_t *buf = NULL;
    int total = all_types_of_arrays_msg_t_marshal(&src, &buf);
    ASSERT(total > 0);

    uint16_t fixed_len   = get_message_fixed_payload_length(buf);
    uint32_t overall_len = get_message_overall_payload_length(buf);

    all_types_of_arrays_msg_t dst = {0};
    int rc = all_types_of_arrays_msg_t_unmarshal(
        buf + WIRE_FRAME_HEADER_SIZE, fixed_len, overall_len, &dst);
    ASSERT_EQ_INT(rc, 0);

    ASSERT_EQ_INT((int)dst.arr1_d_bool.x,   0);
    ASSERT_EQ_INT((int)dst.arr1_d_uint8.x,  0);
    ASSERT_EQ_INT((int)dst.arr1_d_string.x, 0);
    ASSERT_EQ_INT((int)dst.arr2_d_bool.x,   0);
    ASSERT_EQ_INT((int)dst.arr3_d_bool.x,   0);

    all_types_of_arrays_msg_t_free(&dst);
    free(buf);
    PASS();
}

/* ---------------------------------------------------------------------------
 * Test: Free safety — NULL pointer is a no-op
 * --------------------------------------------------------------------------*/
static void test_free_null_safety(void)
{
    TEST(free_null_safety);

    /* all _free() functions must accept NULL without crashing */
    only_scalar_types_msg_t_free(NULL);
    only_variable_types_msg_t_free(NULL);
    all_types_fields_msg_t_free(NULL);
    recursive_nested_msg_t_free(NULL);
    all_types_of_arrays_msg_t_free(NULL);

    PASS();
}

static void test_free_double_free_safety(void)
{
    TEST(free_double_free_safety);

    only_variable_types_msg_t msg = {0};
    only_variable_types_msg_t_set_name(&msg, "test", 4);

    only_variable_types_msg_t_free(&msg);
    /* after free, name.data should be NULL — second free must not crash */
    only_variable_types_msg_t_free(&msg);

    PASS();
}

/* ---------------------------------------------------------------------------
 * Test: Error cases — truncated input
 * --------------------------------------------------------------------------*/
static void test_unmarshal_truncated_fixed_block(void)
{
    TEST(unmarshal_truncated_fixed_block);

    only_scalar_types_msg_t src = {0};
    only_scalar_types_msg_t_set_val_uint64(&src, 42ULL);

    uint8_t *buf = NULL;
    int total = only_scalar_types_msg_t_marshal(&src, &buf);
    ASSERT(total > 0);

    uint16_t fixed_len   = get_message_fixed_payload_length(buf);
    uint32_t overall_len = get_message_overall_payload_length(buf);

    /* Pass a shorter-than-declared fixed block: overall_len = fixed_len - 4 */
    only_scalar_types_msg_t dst = {0};
    int rc = only_scalar_types_msg_t_unmarshal(
        buf + WIRE_FRAME_HEADER_SIZE, fixed_len, overall_len - 4, &dst);
    ASSERT_EQ_INT(rc, -1);

    free(buf);
    PASS();
}

static void test_marshal_null_pointer(void)
{
    TEST(marshal_null_pointer);

    uint8_t *buf = NULL;
    int rc = only_scalar_types_msg_t_marshal(NULL, &buf);
    ASSERT_EQ_INT(rc, -1);
    ASSERT(buf == NULL);

    PASS();
}

/* ---------------------------------------------------------------------------
 * Test: Padding bytes are zeroed in fixed block
 * --------------------------------------------------------------------------*/
static void test_padding_bytes_zeroed(void)
{
    TEST(padding_bytes_zeroed);

    /* OnlyScalarTypesMsg has 1 padding byte at fixed-block offset 47 */
    only_scalar_types_msg_t msg = {0};
    only_scalar_types_msg_t_set_val_bool(&msg, 1);

    uint8_t *buf = NULL;
    ASSERT(only_scalar_types_msg_t_marshal(&msg, &buf) > 0);

    int pad_wire_offset = WIRE_FRAME_HEADER_SIZE + 47;
    ASSERT_EQ_INT(buf[pad_wire_offset], 0);

    free(buf);
    PASS();
}


/* ---------------------------------------------------------------------------
 * main
 * --------------------------------------------------------------------------*/
int main(void)
{
    printf("=== C Roundtrip Tests ===\n\n");

    printf("\n--- Wire frame header ---\n");
    test_wire_frame_header_big_endian();
    test_wire_frame_type_ids();

    printf("\n--- Scalar message ---\n");
    test_only_scalar_types_msg_t_dynamic_payload_size();
    test_only_scalar_types_msg_t_size();
    test_only_scalar_types_msg_t_marshal();
    test_only_scalar_types_msg_t_unmarshal();
    test_only_scalar_types_msg_t_roundtrip_empty_fields();

    printf("\n--- Variable-length message ---\n");
    test_only_variable_types_msg_t_dynamic_payload_size();
    test_only_variable_types_msg_t_size();
    test_only_variable_types_msg_t_marshal();
    test_only_variable_types_msg_t_unmarshal();
    test_only_variable_types_msg_t_roundtrip_empty_fields();

    printf("\n--- AllTypesFields message ---\n");
    test_all_types_fields_msg_t_dynamic_payload_size();
    test_all_types_fields_msg_t_size();
    test_all_types_fields_msg_t_marshal();
    test_all_types_fields_msg_t_unmarshal();
    test_all_types_fields_msg_t_roundtrip_empty_fields();

    printf("\n--- RecursiveNested message ---\n");
    test_recursive_nested_msg_t_dynamic_payload_size();
    test_recursive_nested_msg_t_size();
    test_recursive_nested_msg_t_marshal();
    test_recursive_nested_msg_t_unmarshal();
    test_recursive_nested_msg_t_roundtrip_empty_fields();

    printf("\n--- AllTypesOfArrays message ---\n");
    test_all_types_of_arrays_msg_t_dynamic_payload_size();
    test_all_types_of_arrays_msg_t_size();
    test_all_types_of_arrays_msg_t_marshal();
    test_all_types_of_arrays_msg_t_unmarshal();
    test_all_types_of_arrays_msg_t_roundtrip_empty_fields();

    printf("\n--- Memory safety ---\n");
    test_free_null_safety();
    test_free_double_free_safety();

    printf("\n--- Error handling ---\n");
    test_unmarshal_truncated_fixed_block();
    test_marshal_null_pointer();

    printf("\n--- Layout correctness ---\n");
    test_padding_bytes_zeroed();

    printf("\n=== All C tests PASSED ===\n");
    return 0;
}
