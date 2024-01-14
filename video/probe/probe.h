#ifndef PROBE_H
#define PROBE_H

#include <stddef.h>

/**
 * Probe the video.
 *
 * @param input_files The input files path.
 *
 * @return 0 if the video could be probed, a negative value on error.
 */
int probe(size_t input_files_count, const char *input_files[], int quiet);

/**
 * Check if a file is a video.
 *
 * @param input_file The input file path.
 * @param is_video If the file is a video, returns 1.
 *
 * @return Errors code.
 */
int is_video(const char *input_file, int *is_video);

#endif /* PROBE_H */
