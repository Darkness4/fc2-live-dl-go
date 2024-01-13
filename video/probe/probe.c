#include <libavformat/avformat.h>
#include <libavutil/avutil.h>
#include <libavutil/log.h>
#include <stdio.h>

int probe(size_t input_files_count, const char *input_files[]) {
  if (input_files_count == 0) {
    return 0;
  }

  AVFormatContext *ifmt_ctx = NULL;
  int ret;

  // For each input
  for (size_t input_idx = 0; input_idx < input_files_count; input_idx++) {
    const char *input_file = input_files[input_idx];
    int stream_index = 0;

    if ((ret = avformat_open_input(&ifmt_ctx, input_file, 0, 0)) < 0) {
      fprintf(stderr, "Could not open input file '%s': %s, skipping...\n",
              input_file, av_err2str(ret));
      continue;
    }

    // Retrieve input stream information
    if ((ret = avformat_find_stream_info(ifmt_ctx, 0)) < 0) {
      fprintf(stderr,
              "Failed to retrieve input stream information: %s, skipping...\n",
              av_err2str(ret));
      continue;
    }

    av_dump_format(ifmt_ctx, 0, input_file, 0);

    avformat_close_input(&ifmt_ctx);
  } // for each inputs.

end:
  if (ifmt_ctx)
    avformat_close_input(&ifmt_ctx);

  if (ret < 0) {
    if (ret != AVERROR_EOF) {
      fprintf(stderr, "Error occurred: %s\n", av_err2str(ret));
    }
    return ret;
  }

  return 0;
}
