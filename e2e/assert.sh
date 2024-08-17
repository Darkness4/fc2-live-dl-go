#!/bin/bash

set -euo pipefail

compute_ssim() {
  grep ssim "$1" | awk '{print $11}' | sed -e 's/^All://'
}

compute_psnr() {
  grep psnr "$1" | awk '{print $10}' | sed -e 's/^max://'
}

FFMPEG="docker run --rm -v "$(pwd):/in" --workdir /in linuxserver/ffmpeg:7.0.1"
FFPROBE="docker run --rm -v "$(pwd):/in" --workdir /in --entrypoint ffprobe linuxserver/ffmpeg:7.0.1"

${FFMPEG} -loglevel panic -y -i test.mp4 -vframes 1 expected_frame.png
# Flip the expected frame to created an erroneous frame
${FFMPEG} -loglevel panic -y -i expected_frame.png -vf vflip erroneous_frame.png

for file in test-output/*.mp4; do
  ${FFMPEG} -y -loglevel panic -i "$file" -vframes 1 actual_frame.png

  dimensions=$(${FFPROBE} -loglevel panic -v error -select_streams v:0 -show_entries stream=width,height -of csv=s=x:p=0 expected_frame.png)

  # Resize the expected frame to match the actual frame
  ${FFMPEG} -y -loglevel panic -i actual_frame.png -vf "scale=${dimensions}" actual_frame.rescaled.png

  ###
  # Assert that the expected frame and the actual frame are the same
  ###

  # Extract SSIM and PSNR scores
  ${FFMPEG} -i expected_frame.png -i actual_frame.rescaled.png -lavfi "ssim;[0:v][1:v]psnr" -f null - 2>&1 | grep Parsed_ >ff.log

  ssim_score=$(compute_ssim ff.log)
  if (($(echo "$ssim_score < 0.85" | bc -l))); then
    echo "SSIM score is too low: $ssim_score"
    exit 1
  fi

  psnr_score=$(compute_psnr ff.log)
  if (($(echo "$psnr_score < 30" | bc -l))); then
    echo "PSNR score is too low: $psnr_score"
    exit 1
  fi

  echo "Positive test passed: $file, SSIM: $ssim_score, PSNR: $psnr_score"

  ###
  # Assert that the erroneous frame and the actual frame are different
  ###

  ${FFMPEG} -i erroneous_frame.png -i actual_frame.rescaled.png -lavfi "ssim;[0:v][1:v]psnr" -f null - 2>&1 | grep Parsed_ >ff.log

  ssim_score=$(compute_ssim ff.log)
  if (($(echo "$ssim_score > 0.85" | bc -l))); then
    echo "SSIM score is too high: $ssim_score"
    exit 1
  fi

  psnr_score=$(compute_psnr ff.log)
  if (($(echo "$psnr_score > 30" | bc -l))); then
    echo "PSNR score is too high: $psnr_score"
    exit 1
  fi

  echo "Negative test passed: $file, SSIM: $ssim_score, PSNR: $psnr_score"
done
