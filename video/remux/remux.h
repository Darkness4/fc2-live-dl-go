#ifndef CONVERT_H
#define CONVERT_H

/**
 * Convert audio and video streams from an MPEG transport stream to an video
 * file.
 *
 * @param input_file  The input file name
 * @param output_file The output file name
 * @param audio_only Only extract audio
 *
 * @return 0 if the conversion was successful, a negative value on error.
 */
int remux(const char *input_file, const char *output_file, int audio_only);

#endif /* CONVERT_H */
