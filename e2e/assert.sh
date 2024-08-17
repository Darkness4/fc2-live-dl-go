#!/bin/bash

set -euo pipefail

compute_ssim() {
  grep SSIM "$1" | sed -n 's/.*All:\([0-9.]*\).*/\1/p'
}

FFMPEG="docker run --rm -v "$(pwd):/in" --workdir /in linuxserver/ffmpeg:7.0.1"
FFPROBE="docker run --rm -v "$(pwd):/in" --workdir /in --entrypoint ffprobe linuxserver/ffmpeg:7.0.1"

# Extract frames of expected video
echo "Extracting frames from test.mp4"
mkdir -p expected_frames
rm -f expected_frames/*
${FFMPEG} -y -loglevel panic -i test.mp4 -vf "fps=2" expected_frames/frame_%04d.png
echo "Frames extracted"

for file in test-output/*.mp4; do
  echo "---Testing $file---"

  ${FFMPEG} -y -loglevel panic -i "$file" -vframes 1 actual_frame.png

  dimensions=$(${FFPROBE} -loglevel panic -v error -select_streams v:0 -show_entries stream=width,height -of csv=s=x:p=0 test.mp4)

  # Resize the expected frame to match the actual frame
  ${FFMPEG} -y -loglevel panic -i actual_frame.png -vf "scale=${dimensions}" actual_frame.rescaled.png

  ###
  # Assert that the expected frame and the actual frame are the same
  ###

  # Extract SSIM and PSNR scores
  rm -f ff.log

  # Find frame with the highest SSIM score
  max_ssim=0
  max_ssim_frame=""
  for frame in expected_frames/*.png; do
    ${FFMPEG} -i "$frame" -i actual_frame.rescaled.png -lavfi "ssim" -f null - 2>&1 | grep Parsed_ >ff.log
    ssim_score=$(compute_ssim ff.log)
    if (($(echo "$ssim_score > $max_ssim" | bc -l))); then
      max_ssim=$ssim_score
      max_ssim_frame=$frame
    fi
  done

  echo "Frame with highest SSIM score: $max_ssim_frame, SSIM: $max_ssim"

  if (($(echo "$max_ssim < 0.85" | bc -l))); then
    echo "SSIM score is too low: $max_ssim"
    exit 1
  fi

  echo "Positive test passed: $file, SSIM: $max_ssim"

  ###
  # Assert that the erroneous frame and the actual frame are different
  ###

  # Vflip the expected frame

  ${FFMPEG} -y -loglevel panic -i "$max_ssim_frame" -vf "vflip" erroneous_frame.png

  ${FFMPEG} -i erroneous_frame.png -i actual_frame.rescaled.png -lavfi "ssim;[0:v][1:v]psnr" -f null - 2>&1 | grep Parsed_ >ff.log

  ssim_score=$(compute_ssim ff.log)
  if (($(echo "$ssim_score > 0.85" | bc -l))); then
    echo "SSIM score is too high: $ssim_score"
    exit 1
  fi

  echo "Negative test passed: $file, SSIM: $ssim_score"
done
