// +build test

#include "remux.h"

#include "libavutil/common.h"

#include <stdio.h>

int test_remux() {
  // Convert input.mpeg to output.mp4
  int ret = remux("input.ts", "output.mp4", 0);
  if (ret != AVERROR_EOF && ret != 0) {
    fprintf(stderr, "Error converting file: %d\n", ret);
    return ret;
  }
  printf("File converted successfully\n");
  return 0;
}

int test_remux_no_video() {
  // Convert input.mpeg to output.mp4
  int ret = remux("input.ts", "output.m4a", 1);
  if (ret != AVERROR_EOF && ret != 0) {
    fprintf(stderr, "Error converting file: %d\n", ret);
    return ret;
  }
  printf("File converted successfully\n");
  return 0;
}

int main() {
  int ret = test_remux();
  if (ret < 0)
    return ret;

  ret = test_remux_no_video();
  if (ret < 0)
    return ret;

  return 0;
}
