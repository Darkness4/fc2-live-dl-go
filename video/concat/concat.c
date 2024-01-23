#include <inttypes.h>
#include <libavformat/avformat.h>
#include <libavutil/avutil.h>
#include <libavutil/log.h>
#include <stdint.h>
#include <stdio.h>

int concat(const char *output_file, size_t input_files_count,
           const char *input_files[], int audio_only) {
  av_log_set_level(AV_LOG_ERROR);

  if (input_files_count == 0) {
    return 0;
  }

  AVFormatContext *ifmt_ctx = NULL, *ofmt_ctx = NULL;
  AVPacket *pkt = NULL;
  AVDictionary *opts = NULL;

  int64_t *prev_duration = NULL;
  int64_t *dts_offset = NULL;

  // 2D array of size input_files_count*stream_mapping_size.
  // stream_mapping is mapping from input stream index to output stream index.
  int **stream_mapping = NULL;
  int *stream_mapping_size = NULL;

  // last_pts and last_dts used for concatenation. Size is
  // input_files_count*stream_mapping_size.
  int64_t **prev_dts = NULL;
  int64_t **prev_pts = NULL;
  int ret;

  // Alloc arrays
  stream_mapping = av_calloc(input_files_count, sizeof(*stream_mapping));
  if (!stream_mapping) {
    ret = AVERROR(ENOMEM);
    goto end;
  }
  stream_mapping_size =
      av_calloc(input_files_count, sizeof(*stream_mapping_size));
  if (!stream_mapping_size) {
    ret = AVERROR(ENOMEM);
    goto end;
  }
  prev_dts = av_calloc(input_files_count, sizeof(*prev_dts));
  if (!prev_dts) {
    ret = AVERROR(ENOMEM);
    goto end;
  }
  prev_pts = av_calloc(input_files_count, sizeof(*prev_pts));
  if (!prev_pts) {
    ret = AVERROR(ENOMEM);
    goto end;
  }

  pkt = av_packet_alloc();
  if (!pkt) {
    fprintf(stderr, "Could not allocate AVPacket\n");
    ret = AVERROR(ENOMEM);
    goto end;
  }

  // Open output file
  if ((ret = avformat_alloc_output_context2(&ofmt_ctx, NULL, NULL,
                                            output_file)) < 0) {
    fprintf(stderr, "Could not create output context: %s\n", av_err2str(ret));
    goto end;
  }

  // For each input
  for (size_t input_idx = 0; input_idx < input_files_count; input_idx++) {
    const char *input_file = input_files[input_idx];
    int stream_index = 0;

    if ((ret = avformat_open_input(&ifmt_ctx, input_file, 0, 0)) < 0) {
      fprintf(stderr, "Could not open input file '%s': %s, aborting...\n",
              input_file, av_err2str(ret));
      goto end;
    }

    // Retrieve input stream information
    if ((ret = avformat_find_stream_info(ifmt_ctx, 0)) < 0) {
      fprintf(stderr,
              "Failed to retrieve input stream information: %s, aborting...\n",
              av_err2str(ret));
      avformat_close_input(&ifmt_ctx);
      goto end;
    }

    av_dump_format(ifmt_ctx, 0, input_file, 0);

    // Alloc array of streams
    stream_mapping_size[input_idx] = ifmt_ctx->nb_streams;
    stream_mapping[input_idx] =
        av_calloc(stream_mapping_size[input_idx], sizeof(*stream_mapping));
    if (!stream_mapping) {
      ret = AVERROR(ENOMEM);
      goto end;
    }
    prev_duration =
        av_calloc(stream_mapping_size[input_idx], sizeof(*prev_duration));
    if (!prev_duration) {
      ret = AVERROR(ENOMEM);
      goto end;
    }
    dts_offset = av_calloc(stream_mapping_size[input_idx], sizeof(*dts_offset));
    if (!dts_offset) {
      ret = AVERROR(ENOMEM);
      goto end;
    }
    prev_dts[input_idx] =
        av_calloc(stream_mapping_size[input_idx], sizeof(**prev_dts));
    if (!prev_dts[input_idx]) {
      ret = AVERROR(ENOMEM);
      goto end;
    }
    prev_pts[input_idx] =
        av_calloc(stream_mapping_size[input_idx], sizeof(**prev_pts));
    if (!prev_pts[input_idx]) {
      ret = AVERROR(ENOMEM);
      goto end;
    }

    // Add audio and video streams to output context.
    // Map streams from input to output.
    for (unsigned int i = 0; i < ifmt_ctx->nb_streams; i++) {
      AVStream *out_stream;
      AVStream *in_stream = ifmt_ctx->streams[i];
      AVCodecParameters *in_codecpar = in_stream->codecpar;

      // Blacklist any no audio/video/sub streams
      if (audio_only > 0 && in_codecpar->codec_type != AVMEDIA_TYPE_AUDIO) {
        fprintf(stderr, "Blacklisted stream #%u (%s)\n", i,
                av_get_media_type_string(in_codecpar->codec_type));
        stream_mapping[input_idx][i] = -1;
        continue;
      } else if (in_codecpar->codec_type != AVMEDIA_TYPE_AUDIO &&
                 in_codecpar->codec_type != AVMEDIA_TYPE_VIDEO &&
                 in_codecpar->codec_type != AVMEDIA_TYPE_SUBTITLE) {
        fprintf(stderr, "Blacklisted stream #%u (%s)\n", i,
                av_get_media_type_string(in_codecpar->codec_type));
        stream_mapping[input_idx][i] = -1;
        continue;
      }

      stream_mapping[input_idx][i] = stream_index++;
      fprintf(stderr, "Input %zu, mapping stream %d (%s) to output stream %d\n",
              input_idx, i, av_get_media_type_string(in_codecpar->codec_type),
              stream_mapping[input_idx][i]);

      // Only create streams based on the first video.
      if (input_idx == 0) {
        out_stream = avformat_new_stream(ofmt_ctx, NULL);
        if (!out_stream) {
          fprintf(stderr, "Failed allocating output stream\n");
          ret = AVERROR_UNKNOWN;
          goto end;
        }
        ret = avcodec_parameters_copy(out_stream->codecpar, in_codecpar);
        if (ret < 0) {
          fprintf(stderr, "Failed to copy codec parameters: %s\n",
                  av_err2str(ret));
          goto end;
        }
        out_stream->codecpar->codec_tag = 0;
        if (in_codecpar->codec_type == AVMEDIA_TYPE_VIDEO) {
          out_stream->time_base = in_stream->time_base;
        } else if (in_codecpar->codec_type == AVMEDIA_TYPE_AUDIO) {
          out_stream->time_base = (AVRational){1, in_codecpar->sample_rate};
        }

        fprintf(stderr, "Created output stream (%s)\n",
                av_get_media_type_string(out_stream->codecpar->codec_type));
      }

      // Set to zero
      prev_duration[i] = 0;
      dts_offset[i] = 0;
      prev_dts[input_idx][i] = AV_NOPTS_VALUE;
      prev_pts[input_idx][i] = AV_NOPTS_VALUE;
    }

    if (input_idx == 0) {
      av_dump_format(ofmt_ctx, 0, output_file, 1);

      if (!(ofmt_ctx->oformat->flags & AVFMT_NOFILE)) {
        ret = avio_open(&ofmt_ctx->pb, output_file, AVIO_FLAG_WRITE);
        if (ret < 0) {
          fprintf(stderr, "Could not open output file '%s': %s\n", output_file,
                  av_err2str(ret));
          goto end;
        }
      }

      // Set "faststart" option
      if ((ret = av_dict_set(&opts, "movflags", "faststart", 0)) < 0) {
        fprintf(stderr, "Failed to set options: %s\n", av_err2str(ret));
        goto end;
      }

      if ((ret = avformat_write_header(ofmt_ctx, &opts)) < 0) {
        fprintf(stderr, "Error writing output file header: %s\n",
                av_err2str(ret));
        goto end;
      }
    }

    // Read packets from input file and write to output file
    while (1) {
      AVStream *in_stream, *out_stream;
      // Read packet from input file
      if ((ret = av_read_frame(ifmt_ctx, pkt)) < 0) {
        // No more packets.
        break;
      }

      // Packet is blacklisted.
      if (pkt->stream_index >= stream_mapping_size[input_idx] ||
          stream_mapping[input_idx][pkt->stream_index] < 0) {
        av_packet_unref(pkt);
        continue;
      }

      in_stream = ifmt_ctx->streams[pkt->stream_index];
      pkt->stream_index = stream_mapping[input_idx][pkt->stream_index];
      out_stream = ofmt_ctx->streams[pkt->stream_index];

      pkt->pts = av_rescale_q_rnd(pkt->pts, in_stream->time_base,
                                  out_stream->time_base,
                                  AV_ROUND_NEAR_INF | AV_ROUND_PASS_MINMAX) +
                 dts_offset[pkt->stream_index];
      pkt->dts = av_rescale_q_rnd(pkt->dts, in_stream->time_base,
                                  out_stream->time_base,
                                  AV_ROUND_NEAR_INF | AV_ROUND_PASS_MINMAX) +
                 dts_offset[pkt->stream_index];
      pkt->duration = av_rescale_q(pkt->duration, in_stream->time_base,
                                   out_stream->time_base);

      // Offset due to concat
      if (input_idx > 0 &&
          prev_pts[input_idx - 1][pkt->stream_index] != AV_NOPTS_VALUE) {
        pkt->pts += prev_pts[input_idx - 1][pkt->stream_index] + 1;
      }
      if (input_idx > 0 &&
          prev_pts[input_idx - 1][pkt->stream_index] != AV_NOPTS_VALUE) {
        pkt->dts += prev_dts[input_idx - 1][pkt->stream_index] + 1;
      }

      // Discontinuity handler
      int64_t delta = 0;
      if (prev_dts[input_idx][pkt->stream_index] == AV_NOPTS_VALUE) {
        // Offset because of initial discontinuity
        if (input_idx > 0 &&
            prev_dts[input_idx - 1][pkt->stream_index] != AV_NOPTS_VALUE) {
          delta = prev_dts[input_idx - 1][pkt->stream_index] + 1 - pkt->dts;
        } else {
          delta = -pkt->dts;
        }

        dts_offset[pkt->stream_index] += delta;
        fprintf(
            stderr,
            "input#%zu, stream #%d, start discontinuity, shifting %" PRId64 ", "
            "new offset=%" PRId64 " packet...\n",
            input_idx, pkt->stream_index, delta, dts_offset[pkt->stream_index]);
      } else if (prev_dts[input_idx][pkt->stream_index] != AV_NOPTS_VALUE &&
                 prev_dts[input_idx][pkt->stream_index] >= pkt->dts) {
        // Offset because of non monotonic packet
        delta = prev_dts[input_idx][pkt->stream_index] - pkt->dts +
                prev_duration[pkt->stream_index];

        dts_offset[pkt->stream_index] += delta;
        fprintf(stderr,
                "input#%zu, stream #%d, discontinuity detected, pkt.prev_dts "
                "(%" PRId64 ") >= pkt.next_dts (%" PRId64 "), shifting %" PRId64
                ", new offset=%" PRId64 " packet...\n",
                input_idx, pkt->stream_index,
                prev_dts[input_idx][pkt->stream_index], pkt->dts, delta,
                dts_offset[pkt->stream_index]);
      }
      pkt->dts += delta;
      pkt->pts += delta;

      // Update the previous decoding timestamp
      prev_pts[input_idx][pkt->stream_index] = pkt->pts;
      prev_dts[input_idx][pkt->stream_index] = pkt->dts;
      prev_duration[pkt->stream_index] = pkt->duration;

      pkt->pos = -1;

      if ((ret = av_interleaved_write_frame(ofmt_ctx, pkt)) < 0) {
        fprintf(stderr, "Error writing packet to output file: %s\n",
                av_err2str(ret));
        av_packet_unref(pkt);
        break;
      }

      av_packet_unref(pkt);
    } // while packets.

    avformat_close_input(&ifmt_ctx);
  } // for each inputs.

  // Write output file trailer
  av_write_trailer(ofmt_ctx);

end:
  // Cleanup
  if (pkt)
    av_packet_free(&pkt);

  if (ifmt_ctx)
    avformat_close_input(&ifmt_ctx);
  if (ofmt_ctx && !(ofmt_ctx->oformat->flags & AVFMT_NOFILE))
    avio_closep(&ofmt_ctx->pb);

  if (ofmt_ctx)
    avformat_free_context(ofmt_ctx);

  av_freep(&prev_duration);
  av_freep(&dts_offset);
  for (size_t i = 0; i < input_files_count; i++) {
    av_freep(&prev_dts[i]);
  }
  av_freep(&prev_dts);
  for (size_t i = 0; i < input_files_count; i++) {
    av_freep(&prev_pts[i]);
  }
  av_freep(&prev_pts);
  for (size_t i = 0; i < input_files_count; i++) {
    av_freep(&stream_mapping[i]);
  }
  av_freep(&stream_mapping);
  av_freep(&stream_mapping_size);

  if (opts)
    av_dict_free(&opts);

  if (ret < 0) {
    if (ret != AVERROR_EOF) {
      fprintf(stderr, "Error occurred: %s\n", av_err2str(ret));
    }
    return ret;
  }

  return 0;
}
