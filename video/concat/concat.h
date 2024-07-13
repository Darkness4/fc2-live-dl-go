#ifndef CONCAT_H
#define CONCAT_H

#include <stddef.h>

typedef void *go_ctx;
typedef void *go_span;

/**
 * Start a trace span for the input process.
 *
 * Externally defined in the Go code.
 *
 * @param ctx The Go context.
 * @param index The index of the input file.
 * @param input_file The input file path.
 */
extern go_span goTraceProcessInputStart(go_ctx ctx, size_t index,
                                        char *input_file);

/**
 * End and free a trace span for the input process.
 *
 * Externally defined in the Go code.
 *
 * @param ctx The Go context.
 * @param index The index of the input file.
 * @param input_file The input file path.
 */
extern void goTraceProcessInputEnd(go_span span);

/**
 * Concat audio and video streams. Streams must be aligned and format must be
 * identical. Remux at the same time.
 *
 * @param ctx The Go context.
 * @param input_files The input files path.
 * @param input_files_count Number of files to be treated.
 * @param output_file The output file name.
 * @param audio_only Only extract audio.
 *
 * @return 0 if the conversion was successful, a negative value on error.
 */
int concat(void *ctx, const char *output_file, size_t input_files_count,
           const char *input_files[], int audio_only);

#endif /* CONCAT_H */
