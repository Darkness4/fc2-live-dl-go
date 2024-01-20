#include "probe.h"

#include <libavformat/avformat.h>
#include <libavutil/avutil.h>
#include <libavutil/log.h>
#include <stdio.h>

int probe(size_t input_files_count, const char *input_files[], int quiet) {
  if (input_files_count == 0) {
    return 0;
  }

  if (quiet == 1) {
    av_log_set_level(AV_LOG_ERROR);
  } else {
    av_log_set_level(AV_LOG_INFO);
  }

  AVFormatContext *ifmt_ctx = NULL;
  int ret;

  // For each input
  for (size_t input_idx = 0; input_idx < input_files_count; input_idx++) {
    const char *input_file = input_files[input_idx];
    if ((ret = avformat_open_input(&ifmt_ctx, input_file, 0, 0)) < 0) {
      fprintf(stderr, "Could not open input file '%s': %s, skipping...\n",
              input_file, av_err2str(ret));
      goto end;
    }

    // Retrieve input stream information
    if ((ret = avformat_find_stream_info(ifmt_ctx, 0)) < 0) {
      fprintf(stderr,
              "Failed to retrieve input stream information: %s, skipping...\n",
              av_err2str(ret));
      goto end;
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

struct contains_video_or_audio_ret
contains_video_or_audio(const char *input_file) {
  av_log_set_level(AV_LOG_ERROR);

  AVFormatContext *ifmt_ctx = NULL;
  struct contains_video_or_audio_ret out = {0, 0};

  // For each input
  if ((out.err = avformat_open_input(&ifmt_ctx, input_file, 0, 0)) < 0) {
    fprintf(stderr, "Could not open input file '%s': %s, skipping...\n",
            input_file, av_err2str(out.err));
    goto end;
  }

  // Retrieve input stream information
  if ((out.err = avformat_find_stream_info(ifmt_ctx, 0)) < 0) {
    fprintf(stderr,
            "Failed to retrieve input stream information: %s, skipping...\n",
            av_err2str(out.err));
    goto end;
  }

  av_dump_format(ifmt_ctx, 0, input_file, 0);

  for (unsigned int i = 0; i < ifmt_ctx->nb_streams; i++) {
    AVStream *in_stream = ifmt_ctx->streams[i];
    AVCodecParameters *in_codecpar = in_stream->codecpar;

    if (in_codecpar->codec_type == AVMEDIA_TYPE_VIDEO ||
        in_codecpar->codec_type == AVMEDIA_TYPE_AUDIO) {
      out.contains_video_or_audio = 1;
      goto end;
    }
  }

end:
  if (ifmt_ctx)
    avformat_close_input(&ifmt_ctx);

  if (out.err < 0) {
    if (out.err != AVERROR_EOF) {
      fprintf(stderr, "Error occurred: %s\n", av_err2str(out.err));
    }
    return out;
  }

  return out;
}
