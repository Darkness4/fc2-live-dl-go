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

struct contains_video_or_audio_ret {
  /// If the file contains a video or audio, returns 1.
  int contains_video_or_audio;
  /// Errors code.
  int err;
};

/**
 * Check if a file contains a video or audio stream.
 *
 * @param input_file The input file path.
 *
 * @return Returns a contains_video_or_audio_ret struct.
 */
struct contains_video_or_audio_ret
contains_video_or_audio(const char *input_file);

#endif /* PROBE_H */
