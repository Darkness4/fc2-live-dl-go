#ifndef CONCAT_H
#define CONCAT_H

#include <stddef.h>

/**
 * Concat audio and video streams. Streams must be aligned and format must be
 * identical. Remux at the same time.
 *
 * @param input_files The input files path.
 * @param input_files_count Number of files to be treated.
 * @param output_file The output file name.
 * @param audio_only Only extract audio.
 *
 * @return 0 if the conversion was successful, a negative value on error.
 */
int concat(const char *output_file, size_t input_files_count,
           const char *input_files[], int audio_only);

#endif /* CONCAT_H */
