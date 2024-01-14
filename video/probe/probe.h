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

#endif /* PROBE_H */
