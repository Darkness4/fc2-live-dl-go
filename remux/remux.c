#include "libavformat/avformat.h"

#include <stdio.h>

int remux(const char *input_file, const char *output_file, int audio_only) {
  AVFormatContext *ifmt_ctx = NULL, *ofmt_ctx = NULL;
  AVPacket pkt;
  AVDictionary *opts = NULL;
  int64_t start_time = AV_NOPTS_VALUE;
  int stream_index = 0;
  int *stream_mapping = NULL;
  int stream_mapping_size = 0;
  int ret;

  if ((ret = avformat_open_input(&ifmt_ctx, input_file, 0, 0)) < 0) {
    fprintf(stderr, "Could not open input file '%s': %s\n", input_file,
            av_err2str(ret));
    goto end;
  }

  // Retrieve input stream information
  if ((ret = avformat_find_stream_info(ifmt_ctx, 0)) < 0) {
    fprintf(stderr, "Failed to retrieve input stream information: %s\n",
            av_err2str(ret));
    goto end;
  }

  av_dump_format(ifmt_ctx, 0, input_file, 0);

  // Open output file
  if ((ret = avformat_alloc_output_context2(&ofmt_ctx, NULL, NULL,
                                            output_file)) < 0) {
    fprintf(stderr, "Could not create output context: %s\n", av_err2str(ret));
    goto end;
  }

  // Alloc array of streams
  stream_mapping_size = ifmt_ctx->nb_streams;
  stream_mapping = av_calloc(stream_mapping_size, sizeof(*stream_mapping));
  if (!stream_mapping) {
    ret = AVERROR(ENOMEM);
    goto end;
  }

  // Add audio and video streams to output context
  for (unsigned int i = 0; i < ifmt_ctx->nb_streams; i++) {
    AVStream *out_stream;
    AVStream *in_stream = ifmt_ctx->streams[i];
    AVCodecParameters *in_codecpar = in_stream->codecpar;
    if (audio_only > 0 && in_codecpar->codec_type != AVMEDIA_TYPE_AUDIO) {
      stream_mapping[i] = -1;
      continue;
    } else if (in_codecpar->codec_type != AVMEDIA_TYPE_AUDIO &&
               in_codecpar->codec_type != AVMEDIA_TYPE_VIDEO &&
               in_codecpar->codec_type != AVMEDIA_TYPE_SUBTITLE) {
      stream_mapping[i] = -1;
      continue;
    }

    stream_mapping[i] = stream_index++;
    out_stream = avformat_new_stream(ofmt_ctx, NULL);
    if (!out_stream) {
      fprintf(stderr, "Failed allocating output stream\n");
      ret = AVERROR_UNKNOWN;
      goto end;
    }
    ret = avcodec_parameters_copy(out_stream->codecpar, in_codecpar);
    if (ret < 0) {
      fprintf(stderr, "Failed to copy codec parameters\n");
      goto end;
    }
    out_stream->codecpar->codec_tag = 0;
  }
  av_dump_format(ofmt_ctx, 0, output_file, 1);

  if (!(ofmt_ctx->oformat->flags & AVFMT_NOFILE)) {
    ret = avio_open(&ofmt_ctx->pb, output_file, AVIO_FLAG_WRITE);
    if (ret < 0) {
      fprintf(stderr, "Could not open output file '%s'", output_file);
      goto end;
    }
  }

  // Set "faststart" option
  av_dict_set(&opts, "movflags", "faststart", 0);
  if ((ret = avformat_write_header(ofmt_ctx, &opts)) < 0) {
    fprintf(stderr, "Error writing output file header: %s\n", av_err2str(ret));
    goto end;
  }

  // Read packets from input file and write to output file
  while (1) {
    AVStream *in_stream, *out_stream;
    // Read packet from input file
    if ((ret = av_read_frame(ifmt_ctx, &pkt)) < 0)
      break;

    in_stream = ifmt_ctx->streams[pkt.stream_index];
    if (pkt.stream_index >= stream_mapping_size ||
        stream_mapping[pkt.stream_index] < 0) {
      av_packet_unref(&pkt);
      continue;
    }

    pkt.stream_index = stream_mapping[pkt.stream_index];
    out_stream = ofmt_ctx->streams[pkt.stream_index];

    // Remember the starting timestamp
    if (start_time == AV_NOPTS_VALUE && pkt.pts != AV_NOPTS_VALUE) {
      start_time = pkt.pts;
    }

    // Adjust packet timestamp
    pkt.pts = av_rescale_q_rnd(pkt.pts - start_time, in_stream->time_base,
                               out_stream->time_base,
                               AV_ROUND_NEAR_INF | AV_ROUND_PASS_MINMAX);
    pkt.dts = av_rescale_q_rnd(pkt.dts - start_time, in_stream->time_base,
                               out_stream->time_base,
                               AV_ROUND_NEAR_INF | AV_ROUND_PASS_MINMAX);
    pkt.duration =
        av_rescale_q(pkt.duration, in_stream->time_base, out_stream->time_base);
    pkt.pos = -1;

    if ((ret = av_interleaved_write_frame(ofmt_ctx, &pkt)) < 0) {
      fprintf(stderr, "Error writing packet to output file: %s\n",
              av_err2str(ret));
      goto end;
    }

    av_packet_unref(&pkt);
  }

  // Write output file trailer
  if (av_write_trailer(ofmt_ctx) < 0) {
    av_log(NULL, AV_LOG_ERROR, "Failed to write output file trailer: %s\n",
           av_err2str(ret));
  }

end:
  // Cleanup
  if (ifmt_ctx)
    avformat_close_input(&ifmt_ctx);
  if (ofmt_ctx && !(ofmt_ctx->oformat->flags & AVFMT_NOFILE))
    avio_closep(&ofmt_ctx->pb);

  if (opts)
    av_dict_free(&opts);

  avformat_free_context(ofmt_ctx);

  if (ret < 0) {
    if (ret != AVERROR_EOF) {
      fprintf(stderr, "Error occurred: %s\n", av_err2str(ret));
    }
    return ret;
  }

  return 0;
}
