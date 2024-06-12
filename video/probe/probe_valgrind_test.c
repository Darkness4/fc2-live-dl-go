// +build dontbuild

#include "probe.h"

#include <stdio.h>
#include <string.h>

int main(int argc, char *argv[]) {
  if (argc < 2) {
    fprintf(stderr, "Usage: %s <test>\n", argv[0]);
    return 1;
  }

  if (strncmp(argv[1], "probe", 5) == 0) {
    const char *input_files[] = {"input.mp4"};
    probe(1, input_files, 0);
  } else if (strncmp(argv[1], "contains_video_or_audio", 23) == 0) {
    contains_video_or_audio("input.mp4");
  } else if (strncmp(argv[1], "is_mpegts_or_aac", 16) == 0) {
    is_mpegts_or_aac("input.mp4");
  } else {
    fprintf(stderr, "Unknown test: %s\n", argv[1]);
    return 1;
  }
  return 0;
}
