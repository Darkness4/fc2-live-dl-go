// +build dontbuild

#include "concat.h"

int main(int argc, char *argv[]) {
  const char *input_files[] = {"input.mp4"};
  concat(NULL, "output.mp4", 1, input_files, 0);
  return 0;
}
